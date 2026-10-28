package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	projectdb "L2EStudyLink/db"
	"github.com/gin-gonic/gin"
)

func TestNotificationsSafeLinks(t *testing.T) {
	for _, value := range []string{"https://evil.test", "//evil.test", "/%2fevil.test", "/%5cevil.test", "/%0aevil", "/%zz", "javascript:alert(1)"} {
		if got := notificationLink(value); got != "/dashboard" {
			t.Errorf("unsafe %q => %q", value, got)
		}
	}
	if notificationLink("/my-bookings?tab=pending") != "/my-bookings?tab=pending" {
		t.Fatal("relative link rejected")
	}
}

func TestNotificationsAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, signed := range []bool{false, true} {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if signed {
				c.Set("user_id", int64(1))
			}
		})
		newNotificationService().register(r.Group("/api/v1"))
		for _, route := range []struct{ method, path string }{{"GET", ""}, {"GET", "/stream"}, {"GET", "/widget"}, {"PUT", "/read"}, {"PUT", "/1/read"}} {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(route.method, "/api/v1/notifications"+route.path, nil))
			want := 401
			if signed {
				want = 503
			}
			if w.Code != want {
				t.Fatalf("%s: %d %s", route.path, w.Code, w.Body)
			}
		}
	}
}

func TestNotificationsStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := newNotificationService()
	s.interval = time.Millisecond
	s.lifetime = time.Second
	streamContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	s.load = func(_ context.Context, _ *sql.DB, user, before int64) (notificationSnapshot, error) {
		if user != 42 || before != 0 {
			t.Errorf("wrong stream scope: %d/%d", user, before)
		}
		calls++
		count := int64(2)
		last := int64(5)
		if calls >= 2 {
			count = 1
		}
		if calls >= 3 {
			count = 3
			last = 6
			cancel()
		}
		return notificationSnapshot{Items: []userNotification{}, UnreadCount: count, LastEventID: last}, nil
	}
	r := gin.New()
	r.GET("/", func(c *gin.Context) { s.writeStream(c, nil, 42) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil).WithContext(streamContext))
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("stream: %d %s", w.Code, w.Body)
	}
	if strings.Count(w.Body.String(), "event: notifications") != 3 {
		t.Fatalf("missing read/new-event update: %s", w.Body)
	}
	if s.total != 0 || len(s.streams) != 0 {
		t.Fatal("stream leaked reservation")
	}
	s.load = func(context.Context, *sql.DB, int64, int64) (notificationSnapshot, error) {
		return notificationSnapshot{}, errors.New("private SQL details")
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 503 || strings.Contains(w.Body.String(), "private") || s.total != 0 {
		t.Fatal("query error leaked details or reservation")
	}
	if !s.acquire(42) || !s.acquire(42) || s.acquire(42) {
		t.Fatal("per-user cap failed")
	}
	s.release(42)
	s.release(42)
	for i := int64(1); i <= 32; i++ {
		if !s.acquire(i) {
			t.Fatal("early global limit")
		}
	}
	if s.acquire(100) {
		t.Fatal("global limit failed")
	}
	for i := int64(1); i <= 32; i++ {
		s.release(i)
	}
}

func TestNotificationsSQLiteEvents(t *testing.T) {
	script := `import sqlite3
from pathlib import Path
c=sqlite3.connect(':memory:')
c.execute('PRAGMA foreign_keys=ON')
s=Path('../schema.sqlite').read_text()
c.executescript(s); c.executescript(s)
for n in range(1,4): c.execute("INSERT INTO users(id,name,email,password_hash) VALUES(?,?,?,'test')",(n,'User',str(n)+'@example.com'))
c.execute("INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic) VALUES(1,2,'2026-10-28','09:00','10:00','Go')")
assert c.execute('SELECT COUNT(*) FROM notifications').fetchone()[0]==2
c.execute("UPDATE bookings SET status='confirmed'")
c.execute("UPDATE bookings SET status='confirmed'")
assert c.execute('SELECT COUNT(*) FROM notifications').fetchone()[0]==4
c.execute("INSERT INTO buddy_connections(requester_id,recipient_id) VALUES(1,2)")
c.execute("UPDATE buddy_connections SET status='active'")
c.execute("INSERT INTO activity_logs(user_id,log_date,summary,completed) VALUES(1,'2026-10-28','PRIVATE REFLECTION',1)")
c.execute("UPDATE activity_logs SET completed=1")
assert c.execute('SELECT COUNT(*) FROM notifications').fetchone()[0]==9
assert c.execute('SELECT COUNT(*) FROM notifications WHERE user_id=3').fetchone()[0]==0
assert c.execute("SELECT COUNT(*) FROM notifications WHERE message LIKE '%PRIVATE%'").fetchone()[0]==0
c.commit()
before=c.execute('SELECT COUNT(*) FROM notifications').fetchone()[0]
c.execute("INSERT INTO activity_logs(user_id,log_date,completed) VALUES(1,'2026-10-29',1)"); c.rollback()
assert c.execute('SELECT COUNT(*) FROM notifications').fetchone()[0]==before
c.execute('DELETE FROM users WHERE id=1')
assert c.execute('SELECT COUNT(*) FROM notifications WHERE user_id=1').fetchone()[0]==0
assert not c.execute('PRAGMA foreign_key_check').fetchall()
`
	if output, err := exec.Command("python3", "-c", script).CombinedOutput(); err != nil {
		t.Fatalf("SQLite events: %v\n%s", err, output)
	}
}

func TestNotificationsBrowserHelpers(t *testing.T) {
	script := `const assert=require('node:assert/strict');const n=require('../static/js/notifications.js');
for(const url of ['//evil','/%2fevil','/%5cevil','/%0aevil','https://evil','/%zz']) assert.equal(n.notificationSafeLink(url),'/dashboard');
assert.equal(n.notificationSafeLink('/buddies'),'/buddies'); assert.equal(n.notificationBadge(100),'99+');
let events=[];let buffer=n.notificationReadFrames('event: notifications\r\ndata: {"unread_count":2}\r',e=>events.push(e));
buffer=n.notificationReadFrames(buffer+'\n\r\n: keep-alive\n\n',e=>events.push(e));assert.equal(events.length,1);assert.equal(events[0].unread_count,2);assert.equal(buffer,'');
assert.throws(()=>n.notificationReadFrames('x'.repeat(131073),()=>{}));`
	if output, err := exec.Command("node", "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("browser helpers: %v\n%s", err, output)
	}
}

func TestNotificationsDatabaseFlow(t *testing.T) {
	database := timetableTestDB(t, false)
	user, block := timetableFixture(t, database)
	var other int64
	if err := database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Other','notification@example.com','test') RETURNING id`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := projectdb.MigrateNotifications(context.Background(), database); err != nil {
			t.Fatal(err)
		}
	}
	execSQL := func(query string, args ...any) {
		t.Helper()
		if _, err := database.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	execSQL(`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic) VALUES($1,$2,'2026-10-28','09:00','10:00','Go')`, user, other)
	execSQL(`UPDATE bookings SET status='confirmed'`)
	execSQL(`UPDATE bookings SET status='confirmed'`)
	execSQL(`INSERT INTO buddy_connections(requester_id,recipient_id) VALUES($1,$2)`, user, other)
	execSQL(`UPDATE buddy_connections SET status='active'`)
	execSQL(`INSERT INTO activity_logs(user_id,timetable_id,log_date,summary,completed) VALUES($1,$2,'2026-10-28','PRIVATE REFLECTION',true)`, user, block)
	execSQL(`UPDATE activity_logs SET completed=true`)
	if err := projectdb.MigrateNotifications(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	snapshot, err := loadNotifications(context.Background(), database, user, 0)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.UnreadCount != 4 {
		t.Fatalf("event count: %+v", snapshot)
	}
	for _, item := range snapshot.Items {
		if strings.Contains(item.Message, "PRIVATE") {
			t.Fatal("reflection content leaked")
		}
	}
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO activity_logs(user_id,log_date,completed) VALUES($1,'2026-10-29',true)`, user); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	afterRollback, err := loadNotifications(context.Background(), database, user, 0)
	if err != nil || afterRollback.UnreadCount != snapshot.UnreadCount {
		t.Fatalf("rolled back event persisted: %+v %v", afterRollback, err)
	}
	r := gin.New()
	r.LoadHTMLGlob("../templates/*.html")
	r.Use(func(c *gin.Context) {
		c.Set("db", database)
		id := user
		if c.GetHeader("Other") == "yes" {
			id = other
		}
		c.Set("user_id", id)
	})
	newNotificationService().register(r.Group("/api/v1"))
	request := func(method, path, body string, otherUser bool) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/v1/notifications"+path, strings.NewReader(body))
		if otherUser {
			req.Header.Set("Other", "yes")
		}
		r.ServeHTTP(w, req)
		return w
	}
	if w := request("PUT", fmt.Sprintf("/%d/read", snapshot.Items[0].ID), "", true); w.Code != 404 {
		t.Fatalf("cross-user read: %d", w.Code)
	}
	if own := request("PUT", fmt.Sprintf("/%d/read", snapshot.Items[0].ID), "", false); own.Code != 204 {
		t.Fatalf("own mark read: %d", own.Code)
	}
	if own := request("PUT", fmt.Sprintf("/%d/read", snapshot.Items[0].ID), "", false); own.Code != 204 {
		t.Fatalf("idempotent mark read: %d", own.Code)
	}
	execSQL(`INSERT INTO notifications(user_id,event_key,title) VALUES($1,'new-arrival','New')`, user)
	if w := request("PUT", "/read", fmt.Sprintf(`{"through_id":%d}`, snapshot.LastEventID), false); w.Code != 204 {
		t.Fatalf("mark read: %d %s", w.Code, w.Body)
	}
	if widget := request("GET", "/widget", "", false); widget.Code != 200 || !strings.Contains(widget.Body.String(), `id="notification-widget"`) {
		t.Fatalf("widget missing: %d", widget.Code)
	}
	w := request("GET", "", "", false)
	var updated notificationSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.UnreadCount != 1 {
		t.Fatalf("new arrival marked read: %+v", updated)
	}
	for i := 0; i < 23; i++ {
		execSQL(`INSERT INTO notifications(user_id,event_key,title) VALUES($1,$2,'Page')`, user, fmt.Sprint("page", i))
	}
	updated, err = loadNotifications(context.Background(), database, user, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Items) != 20 || !updated.HasMore {
		t.Fatalf("pagination: %+v", updated)
	}
	older, err := loadNotifications(context.Background(), database, user, updated.Items[19].ID)
	if err != nil || len(older.Items) != 8 {
		t.Fatalf("older: %+v %v", older, err)
	}
	if w := request("PUT", "/read", `{"through_id":1,"user_id":2}`, false); w.Code != 400 {
		t.Fatal("unknown input accepted")
	}
	execSQL(`UPDATE users SET is_suspended=true WHERE id=$1`, user)
	if w := request("GET", "", "", false); w.Code != 403 {
		t.Fatalf("suspended account: %d", w.Code)
	}
}
