package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	projectdb "L2EStudyLink/db"
)

func TestBuddyFlow(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	usersSchema := string(schema)[:strings.Index(string(schema), "-- Skills table")]
	if _, err = database.Exec(usersSchema); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", dsn)
	if err = projectdb.InitDB(); err != nil {
		t.Fatalf("startup migration: %v", err)
	}
	defer projectdb.CloseDB()
	var buddyTable bool
	if err = database.QueryRow(`SELECT to_regclass('buddy_connections') IS NOT NULL`).Scan(&buddyTable); err != nil || !buddyTable {
		t.Fatalf("startup buddy table: exists=%v err=%v", buddyTable, err)
	}
	if _, err = database.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}

	stamp := time.Now().UnixNano()
	var alice, bob, carol, dave int64
	for i, target := range []*int64{&alice, &bob, &carol, &dave} {
		err = database.QueryRow(`INSERT INTO users(name,email,password_hash,bio,public_handle) VALUES($1,$2,'test',$3,$4) RETURNING id`,
			[]string{"Alice", "Bob", "Carol", "Dave"}[i],
			fmt.Sprintf("buddy-%d-%d@example.com", stamp, i),
			fmt.Sprintf("Buddy test profile %d", i),
			fmt.Sprintf("buddy_%d_%d", stamp%1000000, i)).Scan(target)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer database.Exec(`DELETE FROM users WHERE id IN ($1,$2,$3,$4)`, alice, bob, carol, dave)
	if _, err = database.Exec(`UPDATE users SET is_suspended=TRUE WHERE id=$1`, carol); err != nil {
		t.Fatal(err)
	}
	for _, setup := range []struct {
		user int64
		day  string
	}{{alice, "2026-10-26"}, {alice, "2026-10-27"}, {alice, "2026-10-28"}, {bob, "2026-10-27"}, {bob, "2026-10-28"}} {
		if _, err = database.Exec(`INSERT INTO activity_logs(user_id,log_date,summary) VALUES($1,$2::date,'Buddy progress')`, setup.user, setup.day); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = database.Exec(`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,meeting_type,status) VALUES($1,$2,'2026-10-26','09:00','10:00','Buddy session','online','completed')`, bob, dave); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		var userID int64
		fmt.Sscan(c.GetHeader("X-Test-User"), &userID)
		c.Set("user_id", userID)
		c.Set("db", database)
		c.Next()
	})
	router.GET("/buddies", ListBuddies)
	router.POST("/buddies/request", RequestBuddy)
	router.POST("/buddies/respond", RespondBuddy)
	type response struct {
		code int
		body map[string]any
	}
	call := func(userID int64, method, path, body string) response {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Test-User", fmt.Sprint(userID))
		writer := httptest.NewRecorder()
		router.ServeHTTP(writer, request)
		var payload map[string]any
		if err := json.Unmarshal(writer.Body.Bytes(), &payload); err != nil {
			t.Fatalf("%s %s: decode %d %q: %v", method, path, writer.Code, writer.Body.String(), err)
		}
		return response{code: writer.Code, body: payload}
	}
	people := func(payload map[string]any, key string) []any {
		t.Helper()
		items, ok := payload[key].([]any)
		if !ok {
			t.Fatalf("%s is not an array: %#v", key, payload[key])
		}
		return items
	}
	hasUser := func(items []any, userID int64) bool {
		for _, item := range items {
			person, _ := item.(map[string]any)
			if person["user_id"] == float64(userID) {
				return true
			}
		}
		return false
	}

	t.Run("dashboard offers eligible fellows and own streak", func(t *testing.T) {
		result := call(alice, "GET", "/buddies?today=2026-10-28", "")
		if result.code != 200 {
			t.Fatalf("dashboard: %d %#v", result.code, result.body)
		}
		suggestions := people(result.body, "suggestions")
		if !hasUser(suggestions, bob) || !hasUser(suggestions, dave) || hasUser(suggestions, carol) || hasUser(suggestions, alice) {
			t.Fatalf("suggestions: %#v", suggestions)
		}
		me := result.body["me"].(map[string]any)
		if me["current_streak"] != float64(3) || me["longest_streak"] != float64(3) {
			t.Fatalf("own streak: %#v", me)
		}
	})

	t.Run("request validation and directional inboxes", func(t *testing.T) {
		if got := call(0, "POST", "/buddies/request", fmt.Sprintf(`{"buddy_id":%d}`, bob)); got.code != 401 {
			t.Fatalf("anonymous: %d %#v", got.code, got.body)
		}
		if got := call(carol, "POST", "/buddies/request", fmt.Sprintf(`{"buddy_id":%d}`, bob)); got.code != 403 {
			t.Fatalf("suspended actor: %d %#v", got.code, got.body)
		}
		if got := call(alice, "POST", "/buddies/request", fmt.Sprintf(`{"buddy_id":%d}`, alice)); got.code != 400 {
			t.Fatalf("self request: %d %#v", got.code, got.body)
		}
		if got := call(alice, "POST", "/buddies/request", fmt.Sprintf(`{"buddy_id":%d}`, carol)); got.code != 404 {
			t.Fatalf("suspended target: %d %#v", got.code, got.body)
		}
		created := call(alice, "POST", "/buddies/request", fmt.Sprintf(`{"buddy_id":%d}`, bob))
		if created.code != 201 || created.body["status"] != "pending" {
			t.Fatalf("create: %d %#v", created.code, created.body)
		}
		connectionID := int64(created.body["connection_id"].(float64))
		if duplicate := call(bob, "POST", "/buddies/request", fmt.Sprintf(`{"buddy_id":%d}`, alice)); duplicate.code != 409 {
			t.Fatalf("reverse duplicate: %d %#v", duplicate.code, duplicate.body)
		}
		aliceView := call(alice, "GET", "/buddies?today=2026-10-28", "")
		bobView := call(bob, "GET", "/buddies?today=2026-10-28", "")
		if !hasUser(people(aliceView.body, "outgoing"), bob) || !hasUser(people(bobView.body, "incoming"), alice) {
			t.Fatalf("directional requests: alice=%#v bob=%#v", aliceView.body, bobView.body)
		}

		respondBody := fmt.Sprintf(`{"connection_id":%d,"status":"active"}`, connectionID)
		if got := call(dave, "POST", "/buddies/respond", respondBody); got.code != 403 {
			t.Fatalf("unrelated response: %d %#v", got.code, got.body)
		}
		if got := call(alice, "POST", "/buddies/respond", respondBody); got.code != 403 {
			t.Fatalf("requester response: %d %#v", got.code, got.body)
		}
		if got := call(bob, "POST", "/buddies/respond", fmt.Sprintf(`{"connection_id":%d,"status":"accepted"}`, connectionID)); got.code != 400 {
			t.Fatalf("invalid response status: %d %#v", got.code, got.body)
		}
		accepted := call(bob, "POST", "/buddies/respond", respondBody)
		if accepted.code != 200 || accepted.body["status"] != "active" {
			t.Fatalf("accept: %d %#v", accepted.code, accepted.body)
		}
		if replay := call(bob, "POST", "/buddies/respond", respondBody); replay.code != 409 {
			t.Fatalf("replay: %d %#v", replay.code, replay.body)
		}
	})

	t.Run("accepted buddies receive streak comparison", func(t *testing.T) {
		aliceView := call(alice, "GET", "/buddies?today=2026-10-28", "")
		active := people(aliceView.body, "active")
		if len(active) != 1 || !hasUser(active, bob) {
			t.Fatalf("active buddies: %#v", active)
		}
		bobProfile := active[0].(map[string]any)
		streak := bobProfile["streak"].(map[string]any)
		if streak["current_streak"] != float64(3) || streak["longest_streak"] != float64(3) {
			t.Fatalf("buddy streak: %#v", streak)
		}
		if len(people(aliceView.body, "incoming")) != 0 || len(people(aliceView.body, "outgoing")) != 0 {
			t.Fatalf("accepted request remained pending: %#v", aliceView.body)
		}
	})

	t.Run("recipient can decline", func(t *testing.T) {
		created := call(alice, "POST", "/buddies/request", fmt.Sprintf(`{"buddy_id":%d}`, dave))
		if created.code != 201 {
			t.Fatalf("request Dave: %d %#v", created.code, created.body)
		}
		connectionID := int64(created.body["connection_id"].(float64))
		declined := call(dave, "POST", "/buddies/respond", fmt.Sprintf(`{"connection_id":%d,"status":"declined"}`, connectionID))
		if declined.code != 200 || declined.body["status"] != "declined" {
			t.Fatalf("decline: %d %#v", declined.code, declined.body)
		}
		var status string
		if err := database.QueryRow(`SELECT status FROM buddy_connections WHERE id=$1`, connectionID).Scan(&status); err != nil || status != "declined" {
			t.Fatalf("stored decline: %q %v", status, err)
		}
	})
}

func TestBuddySQLiteSchema(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable for SQLite schema regression")
	}
	script := `
import sqlite3
c = sqlite3.connect(':memory:')
c.execute('PRAGMA foreign_keys=ON')
schema = open('../schema.sqlite').read()
c.executescript(schema)
c.executescript(schema)
for i in range(1, 4):
    c.execute("INSERT INTO users(id,name,email,password_hash) VALUES(?,?,?,'x')", (i, f'U{i}', f'u{i}@example.com'))
c.execute("INSERT INTO buddy_connections(requester_id,recipient_id) VALUES(1,2)")
try:
    c.execute("INSERT INTO buddy_connections(requester_id,recipient_id) VALUES(2,1)")
    raise SystemExit('reverse duplicate buddy pair accepted')
except sqlite3.IntegrityError:
    pass
try:
    c.execute("INSERT INTO buddy_connections(requester_id,recipient_id) VALUES(3,3)")
    raise SystemExit('self buddy connection accepted')
except sqlite3.IntegrityError:
    pass
assert c.execute("SELECT status FROM buddy_connections").fetchone()[0] == 'pending'
print('sqlite buddy schema ok')
`
	command := exec.Command("python3", "-c", script)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("SQLite buddy schema: %v\n%s", err, output)
	}
}
