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

// TestPortfolio covers the public proof-of-work portfolio: opt-in handle
// claiming, privacy gating, and the weekly-record aggregates.
func TestPortfolio(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", dsn)
	// Startup migration must be idempotent on a schema that already has the
	// portfolio columns.
	if err := projectdb.InitDB(); err != nil {
		t.Fatalf("startup migration: %v", err)
	}
	defer projectdb.CloseDB()

	stamp := fmt.Sprintf("%d", time.Now().UnixNano())
	var alice, bob, carol, dave int64
	for i, target := range []*int64{&alice, &bob, &carol, &dave} {
		err = db.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES($1,$2,'test') RETURNING id`,
			fmt.Sprintf("Portfolio fellow %s-%d", stamp, i), fmt.Sprintf("portfolio-%s-%d@example.com", stamp, i)).Scan(target)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer db.Exec(`DELETE FROM users WHERE id IN ($1,$2,$3,$4)`, alice, bob, carol, dave)

	if _, err := db.Exec(`UPDATE users SET public_handle='alice_p', portfolio_public=TRUE WHERE id=$1`, alice); err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct{ name, level string }{{"Go", "advanced"}, {"French", "beginner"}} {
		if _, err := db.Exec(`INSERT INTO skills(user_id,skill_name,proficiency) VALUES($1,$2,$3)`, alice, s.name, s.level); err != nil {
			t.Fatal(err)
		}
	}
	// Completed bookings on both sides: 60 + 30 minutes; a pending one is excluded.
	for _, b := range []struct {
		tutor, student int64
		daysAgo        int
		start, end     string
		status         string
	}{{alice, bob, 5, "09:00", "10:00", "completed"},
		{bob, alice, 4, "14:00", "14:30", "completed"},
		{alice, bob, -1, "09:00", "10:00", "confirmed"}} {
		if _, err := db.Exec(`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,meeting_type,status) VALUES($1,$2,CURRENT_DATE-$3::integer,$4,$5,'Portfolio session','online',$6)`,
			b.tutor, b.student, b.daysAgo, b.start, b.end, b.status); err != nil {
			t.Fatal(err)
		}
	}
	// Three consecutive reflection days ending today.
	for i := 0; i < 3; i++ {
		if _, err := db.Exec(`INSERT INTO activity_logs(user_id,log_date,summary) VALUES($1,CURRENT_DATE-$2,'Studied')`, alice, i); err != nil {
			t.Fatal(err)
		}
	}
	// One tutor review from bob (using his completed booking with Alice) and a closed project Alice completed.
	var reviewBooking int64
	if err := db.QueryRow(`SELECT id FROM bookings WHERE tutor_id=$1 AND student_id=$2 AND topic='Portfolio session' AND status='completed'`, bob, alice).Scan(&reviewBooking); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO reviews(booking_id,reviewer_id,reviewee_id,rating,comment,would_book_again) VALUES($1,$2,$3,5,'Great study mentor',TRUE)`, reviewBooking, bob, alice); err != nil {
		t.Fatal(err)
	}
	var projectID int64
	if err := db.QueryRow(`INSERT INTO projects(owner_id,title,description,roles_needed,time_commitment,status) VALUES($1,'Exam prep toolkit','Shared revision notes','writers','2 weeks','closed') RETURNING id`, bob).Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO project_members(project_id,user_id) VALUES($1,$2)`, projectID, alice); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		var uid int64
		fmt.Sscan(c.GetHeader("X-Test-User"), &uid)
		c.Set("user_id", uid)
		c.Next()
	})
	router.GET("/portfolio/:handle", GetPublicPortfolio)
	router.GET("/portfolio/me", GetMyPortfolio)
	router.PUT("/portfolio/handle", ClaimPortfolioHandle)
	router.PUT("/portfolio/visibility", SetPortfolioVisibility)
	call := func(who int64, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if who > 0 {
			req.Header.Set("X-Test-User", fmt.Sprint(who))
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var payload map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &payload)
		return w, payload
	}

	t.Run("public portfolio aggregates", func(t *testing.T) {
		w, payload := call(0, "GET", "/portfolio/alice_p", "")
		if w.Code != 200 {
			t.Fatalf("anon view: %d %s", w.Code, w.Body.String())
		}
		if payload["handle"] != "alice_p" || payload["is_owner"] != false || payload["portfolio_public"] != true {
			t.Fatalf("identity: %v", payload)
		}
		if payload["verified_hours"] != 1.5 || payload["completed_sessions"] != float64(2) {
			t.Fatalf("verified record: %v %v", payload["verified_hours"], payload["completed_sessions"])
		}
		if payload["current_streak"] != float64(3) || payload["longest_streak"] != float64(3) {
			t.Fatalf("streaks: %v %v", payload["current_streak"], payload["longest_streak"])
		}
		skills, _ := payload["skills"].([]any)
		if len(skills) != 2 || skills[0].(map[string]any)["name"] != "Go" {
			t.Fatalf("skills: %v", payload["skills"])
		}
		reviews, _ := payload["reviews"].([]any)
		if len(reviews) != 1 || reviews[0].(map[string]any)["rating"] != float64(5) {
			t.Fatalf("reviews: %v", payload["reviews"])
		}
		projects, _ := payload["projects"].([]any)
		if len(projects) != 1 || projects[0].(map[string]any)["title"] != "Exam prep toolkit" {
			t.Fatalf("projects: %v", payload["projects"])
		}
	})

	t.Run("private portfolios stay hidden", func(t *testing.T) {
		if _, err := db.Exec(`UPDATE users SET portfolio_public=FALSE WHERE id=$1`, alice); err != nil {
			t.Fatal(err)
		}
		defer db.Exec(`UPDATE users SET portfolio_public=TRUE WHERE id=$1`, alice)
		if w, _ := call(0, "GET", "/portfolio/alice_p", ""); w.Code != 404 {
			t.Fatalf("anon on private: %d", w.Code)
		}
		if w, _ := call(carol, "GET", "/portfolio/alice_p", ""); w.Code != 404 {
			t.Fatalf("stranger on private: %d", w.Code)
		}
		w, payload := call(alice, "GET", "/portfolio/alice_p", "")
		if w.Code != 200 || payload["is_owner"] != true {
			t.Fatalf("owner on private: %d %v", w.Code, payload)
		}
		if w, _ := call(alice, "GET", "/portfolio/me", ""); w.Code != 200 {
			t.Fatalf("owner preview: %d", w.Code)
		}
	})

	t.Run("unknown and malformed handles 404", func(t *testing.T) {
		if w, _ := call(0, "GET", "/portfolio/nobody-here", ""); w.Code != 404 {
			t.Fatalf("unknown: %d", w.Code)
		}
		if w, _ := call(0, "GET", "/portfolio/UPPER", ""); w.Code != 404 {
			t.Fatalf("uppercase handle: %d", w.Code)
		}
	})

	t.Run("handle claiming validates and enforces uniqueness", func(t *testing.T) {
		if w, _ := call(carol, "PUT", "/portfolio/handle", `{"handle":"Carol X!"}`); w.Code != 400 {
			t.Fatalf("invalid handle: %d", w.Code)
		}
		if w, _ := call(carol, "PUT", "/portfolio/handle", `{"handle":"admin"}`); w.Code != 400 {
			t.Fatalf("reserved handle: %d", w.Code)
		}
		w, payload := call(carol, "PUT", "/portfolio/handle", `{"handle":"Carol_X"}`)
		if w.Code != 200 || payload["handle"] != "carol_x" {
			t.Fatalf("claim: %d %v", w.Code, payload)
		}
		if w, _ := call(alice, "PUT", "/portfolio/handle", `{"handle":"CAROL_X"}`); w.Code != 409 {
			t.Fatalf("duplicate (case-insensitive): %d", w.Code)
		}
		if w, _ := call(0, "PUT", "/portfolio/handle", `{"handle":"anon_handle"}`); w.Code != 401 {
			t.Fatalf("anonymous claim: %d", w.Code)
		}
	})

	t.Run("visibility toggle requires a handle", func(t *testing.T) {
		if w, _ := call(dave, "PUT", "/portfolio/visibility", `{"public":true}`); w.Code != 400 {
			t.Fatalf("public without handle: %d", w.Code)
		}
		w, payload := call(carol, "PUT", "/portfolio/visibility", `{"public":false}`)
		if w.Code != 200 || payload["portfolio_public"] != false {
			t.Fatalf("toggle off: %d %v", w.Code, payload)
		}
		w, payload = call(carol, "PUT", "/portfolio/visibility", `{"public":true}`)
		if w.Code != 200 || payload["portfolio_public"] != true {
			t.Fatalf("toggle on: %d %v", w.Code, payload)
		}
		if w, _ := call(0, "GET", "/portfolio/carol_x", ""); w.Code != 200 {
			t.Fatalf("public after toggle: %d", w.Code)
		}
	})
}

// TestPortfolioSQLiteSchema keeps schema.sqlite in parity for the portfolio
// columns, including the case-insensitive handle uniqueness rule.
func TestPortfolioSQLiteSchema(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable for SQLite schema regression")
	}
	script := `
import sqlite3
c = sqlite3.connect(':memory:')
c.executescript(open('../schema.sqlite').read())
cols = {r[1] for r in c.execute('PRAGMA table_info(users)')}
assert 'public_handle' in cols and 'portfolio_public' in cols, cols
c.execute("INSERT INTO users(name,email,password_hash) VALUES('A','a@example.com','x')")
c.execute("INSERT INTO users(name,email,password_hash) VALUES('B','b@example.com','x')")
c.execute("UPDATE users SET public_handle='zed_1' WHERE email='a@example.com'")
try:
    c.execute("UPDATE users SET public_handle='ZED_1' WHERE email='b@example.com'")
    raise SystemExit('case-insensitive handle uniqueness not enforced')
except sqlite3.IntegrityError:
    pass
print('sqlite portfolio schema ok')
`
	cmd := exec.Command("python3", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SQLite portfolio schema: %v\n%s", err, out)
	}
}
