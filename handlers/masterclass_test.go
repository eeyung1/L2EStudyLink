package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	projectdb "L2EStudyLink/db"
	"github.com/gin-gonic/gin"
)

func TestMasterclassInput(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	good := masterclassInput{Title: " Go API ", Description: " Practice ", StartsAt: now.Add(time.Hour), DurationMinutes: 60, Capacity: 2}
	if !good.valid(now) || good.Title != "Go API" || good.Description != "Practice" {
		t.Fatal("valid input not normalized")
	}
	for _, mutate := range []func(*masterclassInput){func(i *masterclassInput) { i.Title = "aa" }, func(i *masterclassInput) { i.Title = strings.Repeat("界", 101) }, func(i *masterclassInput) { i.Description = strings.Repeat("界", 2001) }, func(i *masterclassInput) { i.StartsAt = now }, func(i *masterclassInput) { i.StartsAt = now.AddDate(2, 0, 0) }, func(i *masterclassInput) { i.DurationMinutes = 14 }, func(i *masterclassInput) { i.DurationMinutes = 241 }, func(i *masterclassInput) { i.Capacity = 1 }, func(i *masterclassInput) { i.Capacity = 101 }} {
		input := good
		mutate(&input)
		if input.valid(now) {
			t.Fatalf("invalid accepted: %+v", input)
		}
	}
}
func masterclassTestRouter(database *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.LoadHTMLGlob("../templates/*.html")
	r.Use(func(c *gin.Context) {
		if database != nil {
			c.Set("db", database)
		}
		id, _ := strconv.ParseInt(c.GetHeader("User"), 10, 64)
		c.Set("user_id", id)
	})
	r.GET("/masterclasses", MasterclassPage)
	g := r.Group("/api/v1/masterclasses")
	g.GET("", ListMasterclasses)
	g.POST("", CreateMasterclass)
	g.POST("/:id/rsvp", ReserveMasterclass)
	g.DELETE("/:id/rsvp", CancelMasterclassReservation)
	g.PUT("/:id/cancel", CancelMasterclass)
	return r
}
func masterclassRequest(r *gin.Engine, user int64, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("User", fmt.Sprint(user))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}
func TestMasterclassAuthentication(t *testing.T) {
	r := masterclassTestRouter(nil)
	for _, path := range []string{"/masterclasses", "/api/v1/masterclasses", "/api/v1/masterclasses/1/rsvp", "/api/v1/masterclasses/1/cancel"} {
		method := "GET"
		if strings.HasSuffix(path, "rsvp") {
			method = "POST"
		}
		if strings.HasSuffix(path, "cancel") {
			method = "PUT"
		}
		if w := masterclassRequest(r, 0, method, path, ""); w.Code != 401 {
			t.Fatalf("anonymous %s: %d", path, w.Code)
		}
		if w := masterclassRequest(r, 1, method, path, ""); w.Code != 503 {
			t.Fatalf("missing database %s: %d", path, w.Code)
		}
	}
}
func TestMasterclassSQLite(t *testing.T) {
	script := `import sqlite3
from pathlib import Path
c=sqlite3.connect(':memory:');c.execute('PRAGMA foreign_keys=ON')
s=Path('../schema.sqlite').read_text();c.executescript(s);c.executescript(s)
for n in range(1,4):c.execute("INSERT INTO users(id,name,email,password_hash) VALUES(?,?,?,'test')",(n,'User',str(n)+'@example.com'))
c.execute("INSERT INTO masterclasses(tutor_id,title,starts_at,duration_minutes,capacity) VALUES(1,'Learn Go','2027-01-01T10:00:00Z',60,2)")
c.execute('INSERT INTO masterclass_attendees(masterclass_id,user_id) VALUES(1,2)')
try:
 c.execute('INSERT INTO masterclass_attendees(masterclass_id,user_id) VALUES(1,2)');raise AssertionError('duplicate seat accepted')
except sqlite3.IntegrityError:pass
for column,value in [('capacity',1),('duration_minutes',0),('status','invalid')]:
 try:
  c.execute('UPDATE masterclasses SET '+column+'=? WHERE id=1',(value,));raise AssertionError('invalid '+column)
 except sqlite3.IntegrityError:pass
c.execute("UPDATE masterclass_attendees SET status='cancelled' WHERE user_id=2")
c.execute("INSERT INTO masterclass_attendees(masterclass_id,user_id) VALUES(1,2) ON CONFLICT(masterclass_id,user_id) DO UPDATE SET status='reserved'")
assert c.execute('SELECT COUNT(*) FROM masterclass_attendees').fetchone()[0]==1
c.commit();c.execute("UPDATE masterclass_attendees SET status='cancelled'");c.rollback()
assert c.execute('SELECT status FROM masterclass_attendees').fetchone()[0]=='reserved'
c.execute('DELETE FROM masterclasses WHERE id=1');assert not c.execute('SELECT * FROM masterclass_attendees').fetchall()
assert not c.execute('PRAGMA foreign_key_check').fetchall()
`
	if out, err := exec.Command("python3", "-c", script).CombinedOutput(); err != nil {
		t.Fatalf("SQLite: %v %s", err, out)
	}
}
func TestMasterclassBrowserScript(t *testing.T) {
	script := `const fs=require('node:fs');const html=fs.readFileSync('../templates/masterclass.html','utf8');const scripts=[...html.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)].map(x=>x[1]).filter(Boolean);for(const script of scripts)new Function(script);const group=html.split('<script id="masterclass-client">')[1].split('</script>')[0];if(group.includes('innerHTML'))throw Error('Unsafe group class rendering');if(!html.includes('id="bookingPanel"'))throw Error('Existing individual booking flow removed');`
	if out, err := exec.Command("node", "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("browser syntax: %v %s", err, out)
	}
}
func TestMasterclassFlow(t *testing.T) {
	database := timetableTestDB(t, false)
	host, _ := timetableFixture(t, database)
	for i := 0; i < 2; i++ {
		if err := projectdb.MigrateMasterclasses(context.Background(), database); err != nil {
			t.Fatal(err)
		}
	}
	var student, other int64
	for index, target := range []*int64{&student, &other} {
		if err := database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Learner',$1,'test') RETURNING id`, fmt.Sprintf("class%d@example.com", index)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	r := masterclassTestRouter(database)
	body := fmt.Sprintf(`{"title":"Go API","description":"Practice <script>text</script>","starts_at":%q,"duration_minutes":60,"capacity":2}`, time.Now().Add(24*time.Hour).Format(time.RFC3339))
	w := masterclassRequest(r, host, "POST", "/api/v1/masterclasses", body)
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/v1/masterclasses/%d", created.ID)
	check := func(user int64, method, suffix string, want int) {
		t.Helper()
		w := masterclassRequest(r, user, method, path+suffix, "")
		if w.Code != want {
			t.Fatalf("%s %s user %d: %d %s", method, suffix, user, w.Code, w.Body)
		}
	}
	check(host, "POST", "/rsvp", 400)
	check(student, "PUT", "/cancel", 403)
	check(student, "POST", "/rsvp", 201)
	check(student, "POST", "/rsvp", 200)
	check(other, "POST", "/rsvp", 201)
	w = masterclassRequest(r, student, "GET", "/api/v1/masterclasses?mine=true", "")
	var listing struct {
		Items []masterclassView `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listing); err != nil {
		t.Fatal(err)
	}
	if len(listing.Items) != 1 || listing.Items[0].ReservationStatus != "reserved" || listing.Items[0].Reserved != 2 {
		t.Fatalf("listing %s", w.Body)
	}
	check(student, "DELETE", "/rsvp", 200)
	check(student, "DELETE", "/rsvp", 200)
	check(student, "POST", "/rsvp", 201)
	check(host, "PUT", "/cancel", 200)
	check(host, "PUT", "/cancel", 200)
	check(other, "POST", "/rsvp", 409)
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM masterclass_attendees WHERE masterclass_id=$1 AND status='cancelled'`, created.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("cancel preserves history: %d %v", count, err)
	}
	if w := masterclassRequest(r, student, "GET", "/masterclasses", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "Group masterclasses") {
		t.Fatal("page unavailable")
	}
	if w := masterclassRequest(r, host, "POST", "/api/v1/masterclasses", strings.Replace(body, `"capacity":2`, `"capacity":2,"tutor_id":999`, 1)); w.Code != 400 {
		t.Fatal("caller selected host")
	}

	// A reservation waiting on a started class must not be admitted.
	var ended int64
	if err := database.QueryRow(`INSERT INTO masterclasses(tutor_id,title,starts_at,duration_minutes,capacity) VALUES($1,'Past class',NOW()-INTERVAL '1 hour',60,2) RETURNING id`, host).Scan(&ended); err != nil {
		t.Fatal(err)
	}
	if past := masterclassRequest(r, student, "POST", fmt.Sprintf("/api/v1/masterclasses/%d/rsvp", ended), ""); past.Code != 409 {
		t.Fatalf("past class RSVP: %d", past.Code)
	}
	if absent := masterclassRequest(r, student, "DELETE", fmt.Sprintf("/api/v1/masterclasses/%d/rsvp", ended+99999), ""); absent.Code != 404 {
		t.Fatalf("unknown class: %d", absent.Code)
	}
	if invalid := masterclassRequest(r, student, "GET", "/api/v1/masterclasses?before_id=-1", ""); invalid.Code != 400 {
		t.Fatal("invalid cursor accepted")
	}
	// Capacity conflicts do not insert or overwrite another learner's row.
	var full int64
	if err := database.QueryRow(`INSERT INTO masterclasses(tutor_id,title,starts_at,duration_minutes,capacity) VALUES($1,'Full class',NOW()+INTERVAL '1 day',60,2) RETURNING id`, host).Scan(&full); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO masterclass_attendees(masterclass_id,user_id) VALUES($1,$2),($1,$3)`, full, student, other); err != nil {
		t.Fatal(err)
	}
	var third int64
	if err := database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Third','thirdclass@example.com','test') RETURNING id`).Scan(&third); err != nil {
		t.Fatal(err)
	}
	if over := masterclassRequest(r, third, "POST", fmt.Sprintf("/api/v1/masterclasses/%d/rsvp", full), ""); over.Code != 409 {
		t.Fatalf("full class: %d", over.Code)
	}
	if _, err := database.Exec(`UPDATE users SET is_suspended=true WHERE id=$1`, host); err != nil {
		t.Fatal(err)
	}
	if unavailable := masterclassRequest(r, third, "POST", fmt.Sprintf("/api/v1/masterclasses/%d/rsvp", full), ""); unavailable.Code != 409 {
		t.Fatalf("suspended host: %d", unavailable.Code)
	}
	if _, err := database.Exec(`UPDATE users SET is_suspended=true WHERE id=$1`, student); err != nil {
		t.Fatal(err)
	}
	check(student, "POST", "/rsvp", 403)
}

func TestMasterclassConcurrentLastSeats(t *testing.T) {
	fixture := timetableTestDB(t, false)
	host, _ := timetableFixture(t, fixture)
	var schema string
	if err := fixture.QueryRow(`SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		dsn = parsed.String()
	} else {
		dsn += " search_path=" + schema
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	database.SetMaxOpenConns(12)
	var class int64
	if err := fixture.QueryRow(`INSERT INTO masterclasses(tutor_id,title,starts_at,duration_minutes,capacity) VALUES($1,'Concurrent Go',NOW()+INTERVAL '1 day',60,2) RETURNING id`, host).Scan(&class); err != nil {
		t.Fatal(err)
	}
	users := make([]int64, 12)
	for i := range users {
		if err := fixture.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Learner',$1,'test') RETURNING id`, fmt.Sprintf("race%d@example.com", i)).Scan(&users[i]); err != nil {
			t.Fatal(err)
		}
	}
	r := masterclassTestRouter(database)
	start := make(chan struct{})
	results := make(chan int, 12)
	var wg sync.WaitGroup
	for _, user := range users {
		wg.Add(1)
		go func(user int64) {
			defer wg.Done()
			<-start
			results <- masterclassRequest(r, user, "POST", fmt.Sprintf("/api/v1/masterclasses/%d/rsvp", class), "").Code
		}(user)
	}
	close(start)
	wg.Wait()
	close(results)
	reserved := 0
	for code := range results {
		switch code {
		case 201:
			reserved++
		case 409:
		default:
			t.Errorf("unexpected concurrent RSVP status %d", code)
		}
	}
	if reserved != 2 {
		t.Fatalf("reserved %d of two seats", reserved)
	}
	var count int
	if err := fixture.QueryRow(`SELECT COUNT(*) FROM masterclass_attendees WHERE masterclass_id=$1 AND status='reserved'`, class).Scan(&count); err != nil || count != 2 {
		t.Fatalf("oversubscribed: %d %v", count, err)
	}
}
