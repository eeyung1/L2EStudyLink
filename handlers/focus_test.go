package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"L2EStudyLink/focus"
	"L2EStudyLink/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type focusTestClock struct {
	sync.Mutex
	now time.Time
}

func (c *focusTestClock) read() time.Time     { c.Lock(); defer c.Unlock(); return c.now }
func (c *focusTestClock) add(d time.Duration) { c.Lock(); defer c.Unlock(); c.now = c.now.Add(d) }
func focusFixture(t *testing.T) (*focusService, *gin.Engine, *focusTestClock) {
	t.Helper()
	clock := &focusTestClock{now: time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)}
	store, err := focus.NewStore(focus.Config{Clock: clock.read, PresenceTTL: time.Hour, IdleTTL: 2 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	service := newFocusService(store)
	service.name = func(c *gin.Context, id int64) (string, error) {
		if c.GetInt64("user_id") != id {
			t.Error("identity mismatch")
		}
		return fmt.Sprintf("Trusted learner %d", id), nil
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.GetHeader("X-Test-User"), 10, 64)
		if id > 0 {
			c.Set("user_id", id)
		}
	})
	service.register(router.Group("/api/v1"))
	return service, router, clock
}
func focusCall(router http.Handler, user int64, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user > 0 {
		req.Header.Set("X-Test-User", fmt.Sprint(user))
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
func focusCreate(t *testing.T, router http.Handler, user int64) focus.Snapshot {
	t.Helper()
	w := focusCall(router, user, "POST", "/api/v1/focus/rooms", `{"title":"Quiet Go practice","target":"Private study intention"}`)
	if w.Code != 201 {
		t.Fatalf("create %d: %s", w.Code, w.Body.String())
	}
	var room focus.Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	return room
}

func TestFocusLifecycleIdentityAndPrivacy(t *testing.T) {
	service, router, clock := focusFixture(t)
	for _, route := range []struct{ method, path string }{{"GET", "/api/v1/focus/rooms"}, {"POST", "/api/v1/focus/rooms"}, {"POST", "/api/v1/focus/rooms/missing/join"}, {"PUT", "/api/v1/focus/rooms/missing/target"}, {"DELETE", "/api/v1/focus/rooms/missing/membership"}, {"GET", "/api/v1/focus/stream"}} {
		if w := focusCall(router, 0, route.method, route.path, ""); w.Code != 401 {
			t.Fatalf("anonymous %s: %d", route.path, w.Code)
		}
	}
	for _, body := range []string{`{"title":"x","target":"x","user_id":99}`, `{"title":"x","target":"x","display_name":"Spoof"}`, strings.Repeat("x", 2049), `{} {}`} {
		if w := focusCall(router, 1, "POST", "/api/v1/focus/rooms", body); w.Code != 400 {
			t.Fatalf("invalid body: %d %s", w.Code, w.Body.String())
		}
	}
	room := focusCreate(t, router, 1)
	if len(room.Participants) != 1 || room.Participants[0].UserID != 1 || room.Participants[0].DisplayName != "Trusted learner 1" {
		t.Fatalf("identity: %+v", room)
	}
	w := focusCall(router, 2, "GET", "/api/v1/focus/rooms", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "Private study intention") || strings.Contains(w.Body.String(), "Trusted learner") || !strings.Contains(w.Body.String(), room.ID) {
		t.Fatalf("discovery privacy: %d %s", w.Code, w.Body.String())
	}
	path := "/api/v1/focus/rooms/" + room.ID
	if w := focusCall(router, 2, "PUT", path+"/target", `{"target":"Hijack"}`); w.Code != 404 {
		t.Fatalf("nonmember target: %d", w.Code)
	}
	if w := focusCall(router, 2, "DELETE", path+"/membership", ""); w.Code != 404 {
		t.Fatalf("nonmember leave: %d", w.Code)
	}
	clock.add(25 * time.Minute)
	w = focusCall(router, 2, "POST", path+"/join", `{"target":"Work on parsing"}`)
	if w.Code != 200 {
		t.Fatalf("join: %d %s", w.Code, w.Body.String())
	}
	var joined focus.Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &joined); err != nil {
		t.Fatal(err)
	}
	if joined.Timer.Phase != focus.Break || !joined.Epoch.Equal(room.Epoch) {
		t.Fatal("late join restarted clock")
	}
	if w := focusCall(router, 2, "POST", path+"/join", `{"target":"Updated goal"}`); w.Code != 200 {
		t.Fatalf("idempotent join: %d", w.Code)
	}
	w = focusCall(router, 2, "PUT", path+"/target", `{"target":"Write tests"}`)
	if w.Code != 200 {
		t.Fatalf("target: %d %s", w.Code, w.Body.String())
	}
	actual, _ := service.rooms.Get(room.ID)
	if p, _ := member(actual, 2); p.Target != "Write tests" {
		t.Fatal("target not saved")
	}
	if w := focusCall(router, 2, "POST", "/api/v1/focus/rooms", `{"title":"Another","target":"x"}`); w.Code != 409 {
		t.Fatalf("multiple rooms: %d", w.Code)
	}
	if w := focusCall(router, 2, "DELETE", path+"/membership", ""); w.Code != 204 {
		t.Fatalf("leave: %d", w.Code)
	}
	actual, _ = service.rooms.Get(room.ID)
	if _, ok := member(actual, 2); ok {
		t.Fatal("left participant retained")
	}
	if w := focusCall(router, 2, "POST", path+"/join", `{"target":"Return"}`); w.Code != 200 {
		t.Fatalf("rejoin: %d", w.Code)
	}
}

type focusStreamRecorder struct {
	*httptest.ResponseRecorder
	onFlush func()
	fail    bool
}

func (w *focusStreamRecorder) Flush() {
	w.ResponseRecorder.Flush()
	if w.onFlush != nil {
		w.onFlush()
	}
}
func (w *focusStreamRecorder) Write(p []byte) (int, error) {
	if w.fail {
		return 0, errors.New("disconnected")
	}
	return w.ResponseRecorder.Write(p)
}

func TestFocusStreamMembershipClockAndPresence(t *testing.T) {
	service, router, clock := focusFixture(t)
	room := focusCreate(t, router, 1)
	if w := focusCall(router, 2, "GET", "/api/v1/focus/stream?room_id="+room.ID, ""); w.Code != 404 || strings.Contains(w.Body.String(), "Private study") {
		t.Fatalf("nonmember stream: %d %s", w.Code, w.Body.String())
	}
	service.tick = time.Millisecond
	service.heartbeat = time.Hour
	service.lifetime = time.Second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recorder := &focusStreamRecorder{ResponseRecorder: httptest.NewRecorder()}
	flushes := 0
	recorder.onFlush = func() {
		flushes++
		if flushes == 1 {
			clock.add(25 * time.Minute)
			if _, err := service.rooms.Join(room.ID, focus.ParticipantInput{UserID: 2, DisplayName: "Peer\nevent: injected", Target: "<script>unsafe</script>"}); err != nil {
				t.Error(err)
			}
		}
		if flushes >= 3 {
			cancel()
		}
	}
	req := httptest.NewRequest("GET", "/api/v1/focus/stream?room_id="+room.ID, nil).WithContext(ctx)
	req.Header.Set("X-Test-User", "1")
	router.ServeHTTP(recorder, req)
	body := recorder.Body.String()
	if recorder.Header().Get("Content-Type") != "text/event-stream" || recorder.Header().Get("X-Accel-Buffering") != "no" || strings.Count(body, "event: presence\n") != 2 || !strings.Contains(body, `"phase":"break"`) || strings.Contains(body, "\nevent: injected") {
		t.Fatalf("SSE framing/presence/clock: %s", body)
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.totalStreams != 0 || len(service.streams) != 0 {
		t.Fatal("stream slot leaked after cancellation")
	}
}

func TestFocusStreamExpiryDisconnectAndLimits(t *testing.T) {
	for _, mode := range []string{"lifetime", "leave", "write failure"} {
		t.Run(mode, func(t *testing.T) {
			service, router, _ := focusFixture(t)
			room := focusCreate(t, router, 1)
			service.tick = time.Millisecond
			service.heartbeat = time.Millisecond
			service.lifetime = 5 * time.Millisecond
			recorder := &focusStreamRecorder{ResponseRecorder: httptest.NewRecorder(), fail: mode == "write failure"}
			if mode == "leave" {
				once := false
				recorder.onFlush = func() {
					if !once {
						once = true
						if err := service.rooms.Leave(room.ID, 1); err != nil {
							t.Error(err)
						}
					}
				}
			}
			req := httptest.NewRequest("GET", "/api/v1/focus/stream?room_id="+room.ID, nil)
			req.Header.Set("X-Test-User", "1")
			router.ServeHTTP(recorder, req)
			service.mu.Lock()
			slots := service.totalStreams
			service.mu.Unlock()
			if slots != 0 {
				t.Fatal("stream resources retained")
			}
		})
	}
	service, router, _ := focusFixture(t)
	room := focusCreate(t, router, 1)
	if !service.acquireStream(1) || !service.acquireStream(1) || service.acquireStream(1) {
		t.Fatal("per-user stream cap failed")
	}
	if w := focusCall(router, 1, "GET", "/api/v1/focus/stream?room_id="+room.ID, ""); w.Code != 429 {
		t.Fatalf("stream cap response: %d", w.Code)
	}
	service.releaseStream(1)
	service.releaseStream(1)
	for id := int64(1); id <= 64; id++ {
		if !service.acquireStream(id) {
			t.Fatal("global stream cap denied early")
		}
	}
	if service.acquireStream(65) {
		t.Fatal("global stream cap exceeded")
	}
	for id := int64(1); id <= 64; id++ {
		service.releaseStream(id)
	}
}

func TestFocusAuthCookieAndPage(t *testing.T) {
	t.Setenv("JWT_SECRET", "focus-test-signing-key")
	service, _, _ := focusFixture(t)
	router := gin.New()
	router.LoadHTMLFiles("../templates/focus.html")
	router.GET("/focus", middleware.AuthRequired, FocusPage)
	service.register(router.Group("/api/v1", middleware.AuthRequired))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 1, "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("focus-test-signing-key"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/focus", nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "focus-room.js") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("page: %d %s", w.Code, w.Body.String())
	}
	req = httptest.NewRequest("POST", "/api/v1/focus/rooms", strings.NewReader(`{"title":"Focus","target":"Go"}`))
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("cookie CSRF accepted: %d", w.Code)
	}
	req = httptest.NewRequest("POST", "/api/v1/focus/rooms", strings.NewReader(`{"title":"Focus","target":"Go"}`))
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
	req.Header.Set("Origin", "http://example.com")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("same-origin cookie create: %d %s", w.Code, w.Body.String())
	}
	var room focus.Snapshot
	json.Unmarshal(w.Body.Bytes(), &room)
	ctx, cancel := context.WithCancel(context.Background())
	recorder := &focusStreamRecorder{ResponseRecorder: httptest.NewRecorder(), onFlush: cancel}
	req = httptest.NewRequest("GET", "/api/v1/focus/stream?room_id="+room.ID, nil).WithContext(ctx)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
	router.ServeHTTP(recorder, req)
	cancel()
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "event: presence") {
		t.Fatal("cookie SSE failed")
	}
}

func TestFocusExpiredMembershipIndex(t *testing.T) {
	service, router, clock := focusFixture(t)
	room := focusCreate(t, router, 1)
	clock.add(time.Hour)
	if w := focusCall(router, 2, "POST", "/api/v1/focus/rooms/"+room.ID+"/join", `{"target":"Go"}`); w.Code != 200 {
		t.Fatalf("join expired room: %d %s", w.Code, w.Body.String())
	}
	service.mu.Lock()
	_, stale := service.memberships[1]
	service.mu.Unlock()
	if stale {
		t.Fatal("expired index entry retained")
	}
}

func TestFocusBrowserParserAndCountdown(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	source, err := os.ReadFile("../static/js/focus-room.js")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "innerHTML") || strings.Contains(string(source), "access_token=") {
		t.Fatal("unsafe rendering or URL authentication")
	}
	path, err := filepath.Abs("../static/js/focus-room.js")
	if err != nil {
		t.Fatal(err)
	}
	script := `
const assert=require('node:assert/strict');
const {focusReadEvents,focusSeconds,focusPhaseKey}=require(process.argv[1]);
const events=[];
let pending=focusReadEvents('event: clock\r\ndata: {"phase":"focus"}\r',(...e)=>events.push(e));
pending=focusReadEvents(pending+'\n\r\nevent: presence\ndata: {"target":"<script>hello</script>"}\n\n',(...e)=>events.push(e));
assert.equal(pending,'');assert.equal(events.length,2);assert.equal(events[0][0],'clock');assert.equal(events[1][1].target,'<script>hello</script>');
assert.equal(focusSeconds({duration:1500000,at:100},100),1500);
assert.equal(focusSeconds({duration:1500000,at:100},1599),1499);
assert.equal(focusSeconds({duration:1500000,at:100},1600000),0);
assert.notEqual(focusPhaseKey({cycle:1,phase:'focus'}),focusPhaseKey({cycle:2,phase:'focus'}));
assert.throws(()=>focusReadEvents('x'.repeat(131073),()=>{}));
assert.throws(()=>focusReadEvents('data: invalid\n\n',()=>{}));
`
	if output, err := exec.Command(node, "-e", script, path).CombinedOutput(); err != nil {
		t.Fatalf("browser helpers: %v %s", err, output)
	}
}

func TestFocusDatabaseIdentity(t *testing.T) {
	database := timetableTestDB(t, false)
	owner, _ := timetableFixture(t, database)
	store, err := focus.NewStore(focus.Config{})
	if err != nil {
		t.Fatal(err)
	}
	service := newFocusService(store)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("db", database); c.Set("user_id", owner) })
	service.register(router.Group("/api/v1"))
	room := focusCreate(t, router, owner)
	if room.Participants[0].DisplayName != "Learner" || room.Participants[0].UserID != owner {
		t.Fatalf("database identity: %+v", room)
	}
}

func TestFocusConcurrentStreamAndTargetEdits(t *testing.T) {
	service, router, _ := focusFixture(t)
	room := focusCreate(t, router, 1)
	service.tick = time.Millisecond
	service.heartbeat = time.Millisecond
	service.lifetime = time.Second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest("GET", "/api/v1/focus/stream?room_id="+room.ID, nil).WithContext(ctx)
	req.Header.Set("X-Test-User", "1")
	done := make(chan struct{})
	go func() { defer close(done); router.ServeHTTP(httptest.NewRecorder(), req) }()
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			w := focusCall(router, 1, "PUT", "/api/v1/focus/rooms/"+room.ID+"/target", fmt.Sprintf(`{"target":"Target %d"}`, index))
			if w.Code != 200 {
				t.Errorf("concurrent target: %d", w.Code)
			}
			if w := focusCall(router, 2, "GET", "/api/v1/focus/rooms", ""); w.Code != 200 {
				t.Errorf("concurrent list: %d", w.Code)
			}
		}(i)
	}
	group.Wait()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not stop")
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.totalStreams != 0 {
		t.Fatal("stream count leaked")
	}
}
