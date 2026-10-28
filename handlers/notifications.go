package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type userNotification struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	LinkURL   string    `json:"link_url"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
type notificationSnapshot struct {
	Items       []userNotification `json:"items"`
	UnreadCount int64              `json:"unread_count"`
	LastEventID int64              `json:"last_event_id"`
	HasMore     bool               `json:"has_more"`
}
type notificationService struct {
	mu                 sync.Mutex
	streams            map[int64]int
	total              int
	interval, lifetime time.Duration
	load               func(context.Context, *sql.DB, int64, int64) (notificationSnapshot, error)
}

func newNotificationService() *notificationService {
	return &notificationService{streams: make(map[int64]int), interval: 3 * time.Second, lifetime: 2 * time.Minute, load: loadNotifications}
}

// RegisterNotificationRoutes belongs on the AuthRequired API group. Triggers
// persist events with the source transaction; streams read committed state
// without holding a database connection while waiting between updates.
func RegisterNotificationRoutes(group *gin.RouterGroup) { newNotificationService().register(group) }
func (s *notificationService) register(group *gin.RouterGroup) {
	group.GET("/notifications", s.list)
	group.GET("/notifications/widget", s.widget)
	group.GET("/notifications/stream", s.stream)
	group.PUT("/notifications/read", s.markAll)
	group.PUT("/notifications/:id/read", s.markOne)
}
func notificationActor(c *gin.Context) (*sql.DB, int64, bool) {
	c.Header("Cache-Control", "no-store")
	id := c.GetInt64("user_id")
	if id <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in required"})
		return nil, 0, false
	}
	value, ok := c.Get("db")
	database, valid := value.(*sql.DB)
	if !ok || !valid || database == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notifications are temporarily unavailable"})
		return nil, 0, false
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var active bool
	err := database.QueryRowContext(ctx, `SELECT NOT COALESCE(is_suspended,false) FROM users WHERE id=$1`, id).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !active) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account unavailable"})
		return nil, 0, false
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notifications are temporarily unavailable"})
		return nil, 0, false
	}
	return database, id, true
}
func notificationLink(value string) string {
	parsed, err := url.Parse(value)
	decoded, decodeErr := url.PathUnescape(value)
	if err != nil || decodeErr != nil || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.HasPrefix(decoded, "//") || strings.ContainsAny(decoded, "\\\r\n\t") || parsed.Host != "" || parsed.Scheme != "" {
		return "/dashboard"
	}
	return value
}
func loadNotifications(ctx context.Context, database *sql.DB, userID, before int64) (notificationSnapshot, error) {
	result := notificationSnapshot{Items: make([]userNotification, 0)}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FILTER(WHERE is_read=false),COALESCE(MAX(id),0) FROM notifications WHERE user_id=$1`, userID).Scan(&result.UnreadCount, &result.LastEventID); err != nil {
		return result, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,title,LEFT(message,1000),link_url,is_read,created_at FROM notifications WHERE user_id=$1 AND ($2::bigint=0 OR id<$2) ORDER BY id DESC LIMIT 21`, userID, before)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item userNotification
		if err := rows.Scan(&item.ID, &item.Title, &item.Message, &item.LinkURL, &item.IsRead, &item.CreatedAt); err != nil {
			rows.Close()
			return result, err
		}
		item.LinkURL = notificationLink(item.LinkURL)
		result.Items = append(result.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if len(result.Items) > 20 {
		result.HasMore = true
		result.Items = result.Items[:20]
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
func (s *notificationService) widget(c *gin.Context) {
	if _, _, ok := notificationActor(c); !ok {
		return
	}
	c.HTML(http.StatusOK, "layout.html", nil)
}
func (s *notificationService) list(c *gin.Context) {
	database, id, ok := notificationActor(c)
	if !ok {
		return
	}
	before := int64(0)
	if raw := c.Query("before_id"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification cursor"})
			return
		}
		before = value
	}
	result, err := s.load(c.Request.Context(), database, id, before)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Unable to load notifications. Please try again."})
		return
	}
	c.JSON(http.StatusOK, result)
}
func (s *notificationService) markOne(c *gin.Context) {
	database, id, ok := notificationActor(c)
	if !ok {
		return
	}
	notificationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || notificationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	result, err := database.ExecContext(ctx, `UPDATE notifications SET is_read=true WHERE id=$1 AND user_id=$2`, notificationID, id)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Could not mark notification read"})
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Could not mark notification read"})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
func (s *notificationService) markAll(c *gin.Context) {
	database, id, ok := notificationActor(c)
	if !ok {
		return
	}
	var input struct {
		ThroughID int64 `json:"through_id"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF || input.ThroughID < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Choose the notifications to mark read"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if _, err := database.ExecContext(ctx, `UPDATE notifications SET is_read=true WHERE user_id=$1 AND id<=$2 AND is_read=false`, id, input.ThroughID); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Could not mark notifications read"})
		return
	}
	c.Status(http.StatusNoContent)
}
func (s *notificationService) acquire(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.total >= 32 || s.streams[id] >= 2 {
		return false
	}
	s.total++
	s.streams[id]++
	return true
}
func (s *notificationService) release(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.total--
	s.streams[id]--
	if s.streams[id] == 0 {
		delete(s.streams, id)
	}
}
func (s *notificationService) stream(c *gin.Context) {
	database, id, ok := notificationActor(c)
	if !ok {
		return
	}
	s.writeStream(c, database, id)
}
func (s *notificationService) writeStream(c *gin.Context, database *sql.DB, id int64) {
	if !s.acquire(id) {
		c.Header("Retry-After", "15")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Live updates are busy. Your notifications are still available."})
		return
	}
	defer s.release(id)
	current, err := s.load(c.Request.Context(), database, id, 0)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Unable to load notifications"})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("X-Accel-Buffering", "no")
	controller := http.NewResponseController(c.Writer)
	defer controller.SetWriteDeadline(time.Time{})
	send := func(data notificationSnapshot) bool {
		body, err := json.Marshal(data)
		if err != nil {
			return false
		}
		_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
		defer controller.SetWriteDeadline(time.Time{})
		if _, err := fmt.Fprintf(c.Writer, "event: notifications\ndata: %s\n\n", body); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	if !send(current) {
		return
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	lifetime := time.NewTimer(s.lifetime)
	defer lifetime.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-lifetime.C:
			return
		case <-keepalive.C:
			_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := io.WriteString(c.Writer, ": keep-alive\n\n"); err != nil {
				return
			}
			if controller.Flush() != nil {
				return
			}
			_ = controller.SetWriteDeadline(time.Time{})
		case <-ticker.C:
			next, err := s.load(c.Request.Context(), database, id, 0)
			if err != nil {
				return
			}
			if next.LastEventID != current.LastEventID || next.UnreadCount != current.UnreadCount {
				if !send(next) {
					return
				}
				current = next
			}
		}
	}
}
