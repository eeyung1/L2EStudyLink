package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"L2EStudyLink/focus"
	"github.com/gin-gonic/gin"
)

// Rooms are ephemeral and shared within this application process. Deployments
// with multiple instances require room affinity or shared storage (see focus/README.md).
type focusService struct {
	rooms                     *focus.Store
	name                      func(*gin.Context, int64) (string, error)
	mu                        sync.Mutex
	memberships               map[int64]string
	streams                   map[int64]int
	totalStreams              int
	tick, heartbeat, lifetime time.Duration
}

func newFocusService(rooms *focus.Store) *focusService {
	return &focusService{rooms: rooms, name: focusDisplayName, memberships: make(map[int64]string), streams: make(map[int64]int), tick: time.Second, heartbeat: 15 * time.Second, lifetime: 2 * time.Minute}
}

// RegisterFocusRoutes installs one shared store for these authenticated routes.
// Call once on the AuthRequired group, never on the public API group.
func RegisterFocusRoutes(protected *gin.RouterGroup) {
	rooms, err := focus.NewStore(focus.Config{})
	if err != nil {
		panic(err)
	}
	newFocusService(rooms).register(protected)
}
func (s *focusService) register(group *gin.RouterGroup) {
	group.GET("/focus/rooms", s.list)
	group.POST("/focus/rooms", s.create)
	group.POST("/focus/rooms/:id/join", s.join)
	group.PUT("/focus/rooms/:id/target", s.target)
	group.DELETE("/focus/rooms/:id/membership", s.leave)
	group.GET("/focus/stream", s.stream)
}
func FocusPage(c *gin.Context) {
	if c.GetInt64("user_id") <= 0 {
		c.Redirect(http.StatusSeeOther, "/page/login")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "focus.html", nil)
}
func focusUser(c *gin.Context) (int64, bool) {
	c.Header("Cache-Control", "no-store")
	id := c.GetInt64("user_id")
	if id <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in required"})
		return 0, false
	}
	return id, true
}
func focusDisplayName(c *gin.Context, id int64) (string, error) {
	value, exists := c.Get("db")
	database, ok := value.(*sql.DB)
	if !exists || !ok || database == nil {
		return "", errors.New("database unavailable")
	}
	var name string
	err := database.QueryRowContext(c.Request.Context(), `SELECT name FROM users WHERE id=$1 AND COALESCE(is_suspended,false)=false`, id).Scan(&name)
	if err != nil {
		return "", err
	}
	runes := []rune(strings.TrimSpace(name))
	if len(runes) > 80 {
		runes = runes[:80]
	}
	if len(runes) == 0 {
		return "Learner", nil
	}
	return string(runes), nil
}
func focusBody(c *gin.Context, value any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(new(any)) != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter a valid room name and study target"})
		return false
	}
	return true
}
func focusError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, focus.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use a room name up to 80 characters and a study target up to 160 characters"})
	case errors.Is(err, focus.ErrRoomNotFound), errors.Is(err, focus.ErrNotMember):
		c.JSON(http.StatusNotFound, gin.H{"error": "This room or membership is no longer available"})
	case errors.Is(err, focus.ErrRoomFull), errors.Is(err, focus.ErrCapacity):
		c.JSON(http.StatusConflict, gin.H{"error": "There is no space right now. Try another room."})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Focus rooms are temporarily unavailable"})
	}
}
func member(snapshot focus.Snapshot, id int64) (focus.Participant, bool) {
	for _, p := range snapshot.Participants {
		if p.UserID == id {
			return p, true
		}
	}
	return focus.Participant{}, false
}

// Call while holding s.mu. Stale engine presence also clears the HTTP index.
func (s *focusService) currentRoom(id int64) string {
	roomID := s.memberships[id]
	if roomID == "" {
		return ""
	}
	room, err := s.rooms.Get(roomID)
	if err == nil {
		if _, ok := member(room, id); ok {
			return roomID
		}
	}
	delete(s.memberships, id)
	return ""
}

// Keep the identity index bounded by the engine's live roster. Caller holds mu.
func (s *focusService) pruneMemberships() {
	rooms, err := s.rooms.List()
	if err != nil {
		return
	}
	active := make(map[string]map[int64]bool, len(rooms))
	for _, room := range rooms {
		active[room.ID] = make(map[int64]bool, len(room.Participants))
		for _, p := range room.Participants {
			active[room.ID][p.UserID] = true
		}
	}
	for id, roomID := range s.memberships {
		if !active[roomID][id] {
			delete(s.memberships, id)
		}
	}
}
func (s *focusService) list(c *gin.Context) {
	id, ok := focusUser(c)
	if !ok {
		return
	}
	s.mu.Lock()
	current := s.currentRoom(id)
	rooms, err := s.rooms.List()
	s.mu.Unlock()
	if err != nil {
		focusError(c, err)
		return
	}
	// Discovery never exposes participant names or targets to nonmembers.
	items := make([]gin.H, 0, len(rooms))
	for _, room := range rooms {
		items = append(items, gin.H{"id": room.ID, "title": room.Title, "participant_count": len(room.Participants), "timer": room.Timer})
	}
	c.JSON(http.StatusOK, gin.H{"rooms": items, "current_room_id": current})
}
func (s *focusService) create(c *gin.Context) {
	id, ok := focusUser(c)
	if !ok {
		return
	}
	var input struct {
		Title  string `json:"title"`
		Target string `json:"target"`
	}
	if !focusBody(c, &input) {
		return
	}
	name, err := s.name(c, id)
	if err != nil {
		focusError(c, err)
		return
	}
	s.mu.Lock()
	if s.currentRoom(id) != "" {
		s.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "Leave your current room before creating another"})
		return
	}
	s.pruneMemberships()
	room, err := s.rooms.Create(input.Title, focus.ParticipantInput{UserID: id, DisplayName: name, Target: input.Target})
	if err != nil {
		s.mu.Unlock()
		focusError(c, err)
		return
	}
	s.memberships[id] = room.ID
	s.mu.Unlock()
	c.JSON(http.StatusCreated, room)
}
func (s *focusService) join(c *gin.Context) {
	id, ok := focusUser(c)
	if !ok {
		return
	}
	var input struct {
		Target string `json:"target"`
	}
	if !focusBody(c, &input) {
		return
	}
	name, err := s.name(c, id)
	if err != nil {
		focusError(c, err)
		return
	}
	s.mu.Lock()
	s.pruneMemberships()
	roomID := c.Param("id")
	if current := s.currentRoom(id); current != "" && current != roomID {
		s.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "Leave your current room before joining another"})
		return
	}
	room, err := s.rooms.Join(roomID, focus.ParticipantInput{UserID: id, DisplayName: name, Target: input.Target})
	if err != nil {
		s.mu.Unlock()
		focusError(c, err)
		return
	}
	s.memberships[id] = room.ID
	s.mu.Unlock()
	c.JSON(http.StatusOK, room)
}
func (s *focusService) target(c *gin.Context) {
	id, ok := focusUser(c)
	if !ok {
		return
	}
	var input struct {
		Target string `json:"target"`
	}
	if !focusBody(c, &input) {
		return
	}
	s.mu.Lock()
	room, err := s.rooms.Heartbeat(c.Param("id"), id, input.Target)
	s.mu.Unlock()
	if err != nil {
		focusError(c, err)
		return
	}
	c.JSON(http.StatusOK, room)
}
func (s *focusService) leave(c *gin.Context) {
	id, ok := focusUser(c)
	if !ok {
		return
	}
	s.mu.Lock()
	err := s.rooms.Leave(c.Param("id"), id)
	if err != nil {
		s.mu.Unlock()
		focusError(c, err)
		return
	}
	if s.memberships[id] == c.Param("id") {
		delete(s.memberships, id)
	}
	s.mu.Unlock()
	c.Status(http.StatusNoContent)
}
func (s *focusService) acquireStream(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.totalStreams >= 64 || s.streams[id] >= 2 {
		return false
	}
	s.totalStreams++
	s.streams[id]++
	return true
}
func (s *focusService) releaseStream(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalStreams--
	s.streams[id]--
	if s.streams[id] == 0 {
		delete(s.streams, id)
	}
}

func (s *focusService) stream(c *gin.Context) {
	id, ok := focusUser(c)
	if !ok {
		return
	}
	roomID := c.Query("room_id")
	room, err := s.rooms.Get(roomID)
	if err != nil {
		focusError(c, err)
		return
	}
	if _, ok := member(room, id); !ok {
		focusError(c, focus.ErrNotMember)
		return
	}
	if !s.acquireStream(id) {
		c.Header("Retry-After", "5")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many live connections. Close another focus tab and try again."})
		return
	}
	defer s.releaseStream(id)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("X-Accel-Buffering", "no")
	controller := http.NewResponseController(c.Writer)
	defer controller.SetWriteDeadline(time.Time{})
	send := func(event string, value any) bool {
		body, err := json.Marshal(value)
		if err != nil {
			return false
		}
		// Bound slow clients. Unsupported deadlines on recorder writers are harmless.
		_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, body); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	// Presence comparison excludes LastSeenAt to avoid broadcasts for heartbeats.
	roster := func(room focus.Snapshot) string {
		rows := make([]string, 0, len(room.Participants))
		for _, p := range room.Participants {
			encoded, _ := json.Marshal([]any{p.UserID, p.DisplayName, p.Target, p.JoinedAt})
			rows = append(rows, string(encoded))
		}
		return strings.Join(rows, "\n")
	}
	lastRoster := roster(room)
	if !send("presence", room) {
		return
	}
	ticks := time.NewTicker(s.tick)
	defer ticks.Stop()
	pulse := time.NewTicker(s.heartbeat)
	defer pulse.Stop()
	lifetime := time.NewTimer(s.lifetime)
	defer lifetime.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-lifetime.C:
			return // reconnect periodically to revalidate session auth
		case <-pulse.C:
			// Serialize read/refresh with target edits so an old heartbeat
			// snapshot cannot overwrite the learner's newly saved target.
			s.mu.Lock()
			current, err := s.rooms.Get(roomID)
			participant, present := member(current, id)
			if err == nil && present {
				_, err = s.rooms.Heartbeat(roomID, id, participant.Target)
			}
			s.mu.Unlock()
			if err != nil {
				send("room_closed", gin.H{})
				return
			}
			if !present {
				send("membership_expired", gin.H{})
				return
			}
		case <-ticks.C:
			room, err := s.rooms.Get(roomID)
			if err != nil {
				send("room_closed", gin.H{})
				return
			}
			if _, ok := member(room, id); !ok {
				send("membership_expired", gin.H{})
				return
			}
			if changed := roster(room); changed != lastRoster {
				if !send("presence", room) {
					return
				}
				lastRoster = changed
			}
			if !send("clock", room.Timer) {
				return
			}
		}
	}
}
