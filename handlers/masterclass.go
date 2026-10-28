package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

type masterclassInput struct {
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	StartsAt        time.Time `json:"starts_at"`
	DurationMinutes int       `json:"duration_minutes"`
	Capacity        int       `json:"capacity"`
}

func (input *masterclassInput) valid(now time.Time) bool {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	return utf8.RuneCountInString(input.Title) >= 3 && utf8.RuneCountInString(input.Title) <= 100 && utf8.RuneCountInString(input.Description) <= 2000 && input.StartsAt.After(now) && !input.StartsAt.After(now.AddDate(1, 0, 0)) && input.DurationMinutes >= 15 && input.DurationMinutes <= 240 && input.Capacity >= 2 && input.Capacity <= 100
}

type masterclassView struct {
	ID                int64     `json:"id"`
	TutorID           int64     `json:"tutor_id"`
	TutorName         string    `json:"tutor_name"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	StartsAt          time.Time `json:"starts_at"`
	DurationMinutes   int       `json:"duration_minutes"`
	Capacity          int       `json:"capacity"`
	Reserved          int       `json:"reserved"`
	Status            string    `json:"status"`
	ReservationStatus string    `json:"reservation_status"`
}

func masterclassActor(c *gin.Context) (*sql.DB, int64, bool) {
	// Share the current-account check, including suspension, with notifications.
	return notificationActor(c)
}
func MasterclassPage(c *gin.Context) {
	if _, _, ok := masterclassActor(c); !ok {
		return
	}
	c.HTML(http.StatusOK, "masterclass.html", nil)
}
func ListMasterclasses(c *gin.Context) {
	database, user, ok := masterclassActor(c)
	if !ok {
		return
	}
	before := int64(0)
	if raw := c.Query("before_id"); raw != "" {
		var err error
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before <= 0 {
			c.JSON(400, gin.H{"error": "Invalid class cursor"})
			return
		}
	}
	query := strings.TrimSpace(c.Query("q"))
	if utf8.RuneCountInString(query) > 100 {
		c.JSON(400, gin.H{"error": "Search must be at most 100 characters"})
		return
	}
	mine := c.Query("mine") == "true"
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := database.QueryContext(ctx, `SELECT m.id,m.tutor_id,u.name,m.title,m.description,m.starts_at,m.duration_minutes,m.capacity,m.status,COALESCE(a.status,''),
 (SELECT COUNT(*) FROM masterclass_attendees r WHERE r.masterclass_id=m.id AND r.status='reserved')
 FROM masterclasses m JOIN users u ON u.id=m.tutor_id
 LEFT JOIN masterclass_attendees a ON a.masterclass_id=m.id AND a.user_id=$1
 WHERE NOT COALESCE(u.is_suspended,false) AND ($2::bigint=0 OR m.id<$2)
 AND (m.tutor_id=$1 OR a.user_id IS NOT NULL OR (NOT $3::boolean AND m.status='scheduled' AND m.starts_at>NOW()))
 AND (NOT $3::boolean OR m.tutor_id=$1 OR a.user_id IS NOT NULL)
 AND ($4='' OR STRPOS(LOWER(m.title||' '||m.description),LOWER($4))>0)
 ORDER BY m.id DESC LIMIT 21`, user, before, mine, query)
	if err != nil {
		c.JSON(503, gin.H{"error": "Could not load classes. Please try again."})
		return
	}
	defer rows.Close()
	items := make([]masterclassView, 0)
	for rows.Next() {
		var item masterclassView
		if err := rows.Scan(&item.ID, &item.TutorID, &item.TutorName, &item.Title, &item.Description, &item.StartsAt, &item.DurationMinutes, &item.Capacity, &item.Status, &item.ReservationStatus, &item.Reserved); err != nil {
			c.JSON(503, gin.H{"error": "Could not load classes"})
			return
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		c.JSON(503, gin.H{"error": "Could not load classes"})
		return
	}
	hasMore := len(items) > 20
	if hasMore {
		items = items[:20]
	}
	c.JSON(200, gin.H{"items": items, "has_more": hasMore, "user_id": user})
}
func CreateMasterclass(c *gin.Context) {
	database, user, ok := masterclassActor(c)
	if !ok {
		return
	}
	var input masterclassInput
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 12000)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF || !input.valid(time.Now()) {
		c.JSON(400, gin.H{"error": "Use a 3–100 character title, description up to 2000 characters, future date within one year, 15–240 minutes, and 2–100 seats."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var id int64
	err := database.QueryRowContext(ctx, `INSERT INTO masterclasses(tutor_id,title,description,starts_at,duration_minutes,capacity) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, user, input.Title, input.Description, input.StartsAt.UTC(), input.DurationMinutes, input.Capacity).Scan(&id)
	if err != nil {
		c.JSON(503, gin.H{"error": "Could not create class. Please try again."})
		return
	}
	c.JSON(201, gin.H{"id": id, "status": "scheduled"})
}
func ReserveMasterclass(c *gin.Context)           { changeMasterclass(c, "reserve") }
func CancelMasterclassReservation(c *gin.Context) { changeMasterclass(c, "leave") }
func CancelMasterclass(c *gin.Context)            { changeMasterclass(c, "cancel") }

// Every seat mutation takes the same parent-row lock before reading capacity.
// The database transaction also retains one reservation row per class/account.
func changeMasterclass(c *gin.Context, action string) {
	database, user, ok := masterclassActor(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "Invalid class"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(503, gin.H{"error": "Reservations are temporarily unavailable"})
		return
	}
	defer tx.Rollback()
	var tutor int64
	var starts time.Time
	var capacity int
	var status string
	err = tx.QueryRowContext(ctx, `SELECT tutor_id,starts_at,capacity,status FROM masterclasses WHERE id=$1 FOR UPDATE`, id).Scan(&tutor, &starts, &capacity, &status)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"error": "Class not found"})
		return
	}
	if err != nil {
		c.JSON(503, gin.H{"error": "Could not load class"})
		return
	}
	if action == "cancel" && tutor != user {
		c.JSON(403, gin.H{"error": "Only the host can cancel this class"})
		return
	}
	if action != "cancel" && tutor == user {
		c.JSON(400, gin.H{"error": "The host does not need a seat"})
		return
	}
	var hostActive bool
	if err = tx.QueryRowContext(ctx, `SELECT NOT COALESCE(is_suspended,false) FROM users WHERE id=$1`, tutor).Scan(&hostActive); err != nil || !hostActive {
		c.JSON(409, gin.H{"error": "Class is unavailable"})
		return
	}
	if action == "cancel" && status == "cancelled" {
		c.JSON(200, gin.H{"status": "cancelled"})
		return
	}
	// Compare with the database clock after obtaining the lock. A waiting
	// reservation cannot slip through just because its request began earlier.
	var future bool
	if err = tx.QueryRowContext(ctx, `SELECT $1::timestamptz>clock_timestamp()`, starts).Scan(&future); err != nil {
		c.JSON(503, gin.H{"error": "Could not check class time"})
		return
	}
	if status != "scheduled" || !future {
		c.JSON(409, gin.H{"error": "This class is cancelled or has already started"})
		return
	}
	responseStatus := "cancelled"
	code := http.StatusOK
	switch action {
	case "cancel":
		_, err = tx.ExecContext(ctx, `UPDATE masterclasses SET status='cancelled' WHERE id=$1`, id)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE masterclass_attendees SET status='cancelled',updated_at=NOW() WHERE masterclass_id=$1 AND status='reserved'`, id)
		}
	case "leave":
		var result sql.Result
		result, err = tx.ExecContext(ctx, `UPDATE masterclass_attendees SET status='cancelled',updated_at=NOW() WHERE masterclass_id=$1 AND user_id=$2`, id, user)
		if err == nil {
			var count int64
			count, err = result.RowsAffected()
			if err == nil && count == 0 {
				c.JSON(404, gin.H{"error": "Reservation not found"})
				return
			}
		}
	case "reserve":
		var existing string
		err = tx.QueryRowContext(ctx, `SELECT status FROM masterclass_attendees WHERE masterclass_id=$1 AND user_id=$2`, id, user).Scan(&existing)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			c.JSON(503, gin.H{"error": "Could not check reservation"})
			return
		}
		if existing == "reserved" {
			c.JSON(200, gin.H{"status": "reserved"})
			return
		}
		var reserved int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM masterclass_attendees WHERE masterclass_id=$1 AND status='reserved'`, id).Scan(&reserved); err != nil {
			c.JSON(503, gin.H{"error": "Could not check seats"})
			return
		}
		if reserved >= capacity {
			c.JSON(409, gin.H{"error": "This class is full"})
			return
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO masterclass_attendees(masterclass_id,user_id) VALUES($1,$2) ON CONFLICT(masterclass_id,user_id) DO UPDATE SET status='reserved',updated_at=NOW()`, id, user)
		responseStatus = "reserved"
		code = http.StatusCreated
	}
	if err != nil {
		c.JSON(503, gin.H{"error": "Could not update reservation. Please try again."})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(503, gin.H{"error": "Could not confirm reservation. Refresh to check its status."})
		return
	}
	c.JSON(code, gin.H{"status": responseStatus})
}
