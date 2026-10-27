package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	projectdb "L2EStudyLink/db"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// Every test gets its own schema, so the configured database's existing tables
// and other handler suites are never changed by migration fixtures.
func timetableTestDB(t *testing.T, legacy bool) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	schemaName := fmt.Sprintf("timetable_test_%d", time.Now().UnixNano())
	if _, err = database.Exec(`CREATE SCHEMA ` + schemaName); err != nil {
		database.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := database.Exec(`DROP SCHEMA ` + schemaName + ` CASCADE`); err != nil {
			t.Errorf("cleanup schema: %v", err)
		}
		database.Close()
	})
	if _, err = database.Exec(`SET search_path TO ` + schemaName); err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	source := string(schema)
	if legacy {
		source = strings.Replace(source, "timetable_id INT REFERENCES timetable(id) ON DELETE SET NULL,", "timetable_id INT NOT NULL REFERENCES timetable(id) ON DELETE CASCADE,", 1)
		source = strings.Replace(source, "    activity_name VARCHAR(255) NOT NULL DEFAULT '',\n", "", 1)
	}
	if _, err = database.Exec(source); err != nil {
		t.Fatal(err)
	}
	return database
}

func timetableFixture(t *testing.T, database *sql.DB) (int64, int64) {
	t.Helper()
	var user, block int64
	if err := database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Learner','timetable@example.com','test') RETURNING id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`INSERT INTO timetable(user_id,day_of_week,start_time,end_time,activity) VALUES($1,2,'09:00','10:00','Learn Go') RETURNING id`, user).Scan(&block); err != nil {
		t.Fatal(err)
	}
	return user, block
}

func TestTimetableReflectionHistory(t *testing.T) {
	database := timetableTestDB(t, false)
	user, block := timetableFixture(t, database)
	var other int64
	if err := database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Other','other@example.com','test') RETURNING id`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("db", database)
		if c.GetHeader("X-Test-Other") == "yes" {
			c.Set("user_id", other)
		} else {
			c.Set("user_id", user)
		}
	})
	router.POST("/reflections", AddReflection)
	router.GET("/reflections", GetReflections)
	router.DELETE("/timetable/:id", DeleteTimeBlock)
	request := func(method, path, body string, otherUser bool) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if otherUser {
			req.Header.Set("X-Test-Other", "yes")
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	payload := fmt.Sprintf(`{"timetable_id":%d,"log_date":"2026-10-27","summary":"First reflection","audio_url":"/audio/take.webm","activity_name":"Spoofed"}`, block)
	if w := request("POST", "/reflections", payload, true); w.Code != 404 {
		t.Fatalf("foreign block accepted: %d %s", w.Code, w.Body.String())
	}
	if w := request("POST", "/reflections", payload, false); w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	if _, err := database.Exec(`UPDATE timetable SET activity='Renamed Go' WHERE id=$1`, block); err != nil {
		t.Fatal(err)
	}
	update := fmt.Sprintf(`{"timetable_id":%d,"log_date":"2026-10-27","summary":"Updated reflection"}`, block)
	if w := request("POST", "/reflections", update, false); w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	checkHistory := func(deleted bool) {
		t.Helper()
		w := request("GET", "/reflections", "", false)
		if w.Code != 200 {
			t.Fatalf("history: %d %s", w.Code, w.Body.String())
		}
		var history []struct {
			TimetableID  *int64 `json:"timetable_id"`
			Activity     string `json:"activity"`
			ActivityName string `json:"activity_name"`
			Summary      string `json:"summary"`
			AudioURL     string `json:"audio_url"`
			Completed    bool   `json:"completed"`
			Day          int    `json:"day_of_week"`
			Start        string `json:"start_time"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &history); err != nil {
			t.Fatal(err)
		}
		if len(history) != 1 {
			t.Fatalf("expected one reflection: %s", w.Body.String())
		}
		h := history[0]
		if h.Activity != "Learn Go" || h.ActivityName != "Learn Go" || h.Summary != "Updated reflection" || h.AudioURL != "/audio/take.webm" || !h.Completed {
			t.Fatalf("snapshot/content lost: %+v", h)
		}
		if deleted && (h.TimetableID != nil || h.Start != "" || h.Day != 2) {
			t.Fatalf("deleted block metadata: %+v", h)
		}
		if !deleted && (h.TimetableID == nil || *h.TimetableID != block || h.Start != "09:00") {
			t.Fatalf("live block metadata: %+v", h)
		}
	}
	checkHistory(false)
	if w := request("DELETE", fmt.Sprintf("/timetable/%d", block), "", true); w.Code != 404 {
		t.Fatalf("foreign delete: %d", w.Code)
	}
	if w := request("DELETE", fmt.Sprintf("/timetable/%d", block), "", false); w.Code != 200 {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	checkHistory(true)
	if w := request("GET", "/reflections", "", true); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("foreign history: %d %s", w.Code, w.Body.String())
	}
	if w := request("POST", "/reflections", payload, false); w.Code != 404 {
		t.Fatalf("deleted block accepted: %d %s", w.Code, w.Body.String())
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM activity_logs WHERE user_id=$1 AND completed`, user).Scan(&count); err != nil || count != 1 {
		t.Fatalf("completed history count: %d %v", count, err)
	}
	if _, err := database.Exec(`DELETE FROM users WHERE id=$1`, user); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM activity_logs WHERE user_id=$1`, user).Scan(&count); err != nil || count != 0 {
		t.Fatalf("user deletion must still cascade: %d %v", count, err)
	}
}

func TestTimetableReflectionMigration(t *testing.T) {
	database := timetableTestDB(t, true)
	user, block := timetableFixture(t, database)
	if _, err := database.Exec(`ALTER TABLE activity_logs RENAME CONSTRAINT activity_logs_timetable_id_fkey TO legacy_custom_timetable_fk`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO activity_logs(timetable_id,user_id,log_date,summary,audio_url,completed) VALUES($1,$2,'2026-10-27','Existing reflection','/audio/old.webm',true)`, block, user); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := projectdb.MigrateReflectionHistory(context.Background(), database); err != nil {
			t.Fatalf("migration run %d: %v", i, err)
		}
		if _, err := database.Exec(`UPDATE timetable SET activity='Changed after migration' WHERE id=$1`, block); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.Exec(`DELETE FROM timetable WHERE id=$1`, block); err != nil {
		t.Fatal(err)
	}
	var parent *int64
	var activity, summary, audio string
	if err := database.QueryRow(`SELECT timetable_id,activity_name,summary,audio_url FROM activity_logs WHERE user_id=$1`, user).Scan(&parent, &activity, &summary, &audio); err != nil {
		t.Fatal(err)
	}
	if parent != nil || activity != "Learn Go" || summary != "Existing reflection" || audio != "/audio/old.webm" {
		t.Fatalf("migrated history changed: %v %q %q %q", parent, activity, summary, audio)
	}
	// Re-running after deletion must also preserve the orphaned row and snapshot.
	if err := projectdb.MigrateReflectionHistory(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`DROP TABLE activity_logs`); err != nil {
		t.Fatal(err)
	}
	if err := projectdb.MigrateReflectionHistory(context.Background(), database); err != nil {
		t.Fatalf("absent table: %v", err)
	}
}

func TestTimetableReflectionSQLite(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable for SQLite schema regression")
	}
	command := exec.Command(python, "-c", `
import sqlite3
c = sqlite3.connect(':memory:')
c.execute('PRAGMA foreign_keys = ON')
c.executescript(open('../schema.sqlite').read())
c.execute("INSERT INTO users(id,name,email,password_hash) VALUES(1,'Learner','learner@example.com','test')")
c.execute("INSERT INTO timetable(id,user_id,day_of_week,start_time,end_time,activity) VALUES(1,1,2,'09:00','10:00','Learn Go')")
c.execute("INSERT INTO activity_logs(timetable_id,user_id,log_date,activity_name,summary,audio_url,completed) VALUES(1,1,'2026-10-27','Learn Go','Reflection','/audio/test.webm',1)")
c.execute('DELETE FROM timetable WHERE id=1')
assert c.execute('SELECT timetable_id,activity_name,summary,audio_url,completed FROM activity_logs').fetchall() == [(None,'Learn Go','Reflection','/audio/test.webm',1)]
assert c.execute('PRAGMA foreign_key_check').fetchall() == []
c.execute('DELETE FROM users WHERE id=1')
assert c.execute('SELECT COUNT(*) FROM activity_logs').fetchone() == (0,)
`)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("SQLite history preservation: %v\n%s", err, output)
	}
}
