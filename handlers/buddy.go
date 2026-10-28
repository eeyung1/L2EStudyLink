package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type buddyStreakSummary struct {
	CurrentStreak   int `json:"current_streak"`
	LongestStreak   int `json:"longest_streak"`
	TotalActiveDays int `json:"total_active_days"`
}

type buddyPerson struct {
	ConnectionID int64               `json:"connection_id,omitempty"`
	UserID       int64               `json:"user_id"`
	Name         string              `json:"name"`
	Handle       string              `json:"handle,omitempty"`
	Bio          string              `json:"bio,omitempty"`
	Status       string              `json:"status,omitempty"`
	CreatedAt    time.Time           `json:"created_at,omitempty"`
	Streak       *buddyStreakSummary `json:"streak,omitempty"`
}

type buddyDashboard struct {
	Me          buddyStreakSummary `json:"me"`
	Active      []buddyPerson      `json:"active"`
	Incoming    []buddyPerson      `json:"incoming"`
	Outgoing    []buddyPerson      `json:"outgoing"`
	Suggestions []buddyPerson      `json:"suggestions"`
	Today       string             `json:"today"`
}

func buddyDatabase(c *gin.Context) (*sql.DB, bool) {
	value, ok := c.Get("db")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return nil, false
	}
	database, ok := value.(*sql.DB)
	if !ok || database == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return nil, false
	}
	return database, true
}

func buddyActor(c *gin.Context, database *sql.DB) (int64, bool) {
	actor := c.GetInt64("user_id")
	if actor <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in required"})
		return 0, false
	}
	var active bool
	err := database.QueryRowContext(c.Request.Context(), `SELECT NOT is_suspended FROM users WHERE id=$1`, actor).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account unavailable"})
		return 0, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify account"})
		return 0, false
	}
	if !active {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account unavailable"})
		return 0, false
	}
	return actor, true
}

func buddyToday(value string) (time.Time, string) {
	today, err := time.Parse("2006-01-02", value)
	if err != nil {
		today = time.Now().UTC()
	}
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	return today, today.Format("2006-01-02")
}

func loadBuddyStreak(ctx context.Context, database *sql.DB, userID int64, today time.Time) (buddyStreakSummary, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT active_day, COUNT(*)
		FROM (
			SELECT log_date AS active_day FROM activity_logs WHERE user_id=$1
			UNION ALL
			SELECT session_date AS active_day FROM bookings
			WHERE (tutor_id=$1 OR student_id=$1) AND status='completed'
		) activity
		GROUP BY active_day
		ORDER BY active_day`, userID)
	if err != nil {
		return buddyStreakSummary{}, err
	}
	defer rows.Close()

	dailyCounts := make(map[string]int)
	for rows.Next() {
		var day time.Time
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return buddyStreakSummary{}, err
		}
		dailyCounts[day.Format("2006-01-02")] = count
	}
	if err := rows.Err(); err != nil {
		return buddyStreakSummary{}, err
	}
	current, longest := CalculateStreaks(dailyCounts, today)
	return buddyStreakSummary{
		CurrentStreak:   current,
		LongestStreak:   longest,
		TotalActiveDays: len(dailyCounts),
	}, nil
}

// BuddyPage renders the authenticated accountability dashboard.
func BuddyPage(c *gin.Context) {
	c.HTML(http.StatusOK, "buddy.html", nil)
}

// ListBuddies returns accepted buddies, pending requests and available fellows.
// Streak records are only disclosed for an accepted buddy relationship.
func ListBuddies(c *gin.Context) {
	database, ok := buddyDatabase(c)
	if !ok {
		return
	}
	actor, ok := buddyActor(c, database)
	if !ok {
		return
	}
	today, todayString := buddyToday(c.Query("today"))
	myStreak, err := loadBuddyStreak(c.Request.Context(), database, actor, today)
	if err != nil {
		log.Printf("buddy own streak: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load streaks"})
		return
	}

	result := buddyDashboard{
		Me:          myStreak,
		Active:      []buddyPerson{},
		Incoming:    []buddyPerson{},
		Outgoing:    []buddyPerson{},
		Suggestions: []buddyPerson{},
		Today:       todayString,
	}

	rows, err := database.QueryContext(c.Request.Context(), `
		SELECT bc.id, bc.requester_id, bc.recipient_id, bc.status, bc.created_at,
			u.id, u.name, COALESCE(u.public_handle,''), COALESCE(u.bio,'')
		FROM buddy_connections bc
		JOIN users u ON u.id = CASE WHEN bc.requester_id=$1 THEN bc.recipient_id ELSE bc.requester_id END
		WHERE (bc.requester_id=$1 OR bc.recipient_id=$1)
		  AND bc.status IN ('pending','active')
		ORDER BY CASE bc.status WHEN 'active' THEN 0 ELSE 1 END, bc.created_at DESC`, actor)
	if err != nil {
		log.Printf("list buddy connections: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load buddies"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var person buddyPerson
		var requesterID, recipientID int64
		if err := rows.Scan(&person.ConnectionID, &requesterID, &recipientID, &person.Status, &person.CreatedAt, &person.UserID, &person.Name, &person.Handle, &person.Bio); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load buddies"})
			return
		}
		switch person.Status {
		case "active":
			streak, err := loadBuddyStreak(c.Request.Context(), database, person.UserID, today)
			if err != nil {
				log.Printf("buddy streak for %d: %v", person.UserID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load streak comparison"})
				return
			}
			person.Streak = &streak
			result.Active = append(result.Active, person)
		case "pending":
			if recipientID == actor {
				result.Incoming = append(result.Incoming, person)
			} else {
				result.Outgoing = append(result.Outgoing, person)
			}
		}
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load buddies"})
		return
	}

	suggestionRows, err := database.QueryContext(c.Request.Context(), `
		SELECT u.id, u.name, COALESCE(u.public_handle,''), COALESCE(u.bio,'')
		FROM users u
		WHERE u.id<>$1 AND u.is_suspended=FALSE
		  AND NOT EXISTS (
			SELECT 1 FROM buddy_connections bc
			WHERE (bc.requester_id=$1 AND bc.recipient_id=u.id)
			   OR (bc.recipient_id=$1 AND bc.requester_id=u.id)
		  )
		ORDER BY u.name, u.id
		LIMIT 50`, actor)
	if err != nil {
		log.Printf("buddy suggestions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load fellows"})
		return
	}
	defer suggestionRows.Close()
	for suggestionRows.Next() {
		var person buddyPerson
		if err := suggestionRows.Scan(&person.UserID, &person.Name, &person.Handle, &person.Bio); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load fellows"})
			return
		}
		result.Suggestions = append(result.Suggestions, person)
	}
	if err := suggestionRows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load fellows"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// RequestBuddy opens one pending request for an unordered pair of users.
func RequestBuddy(c *gin.Context) {
	database, ok := buddyDatabase(c)
	if !ok {
		return
	}
	actor, ok := buddyActor(c, database)
	if !ok {
		return
	}
	var input struct {
		BuddyID int64 `json:"buddy_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.BuddyID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Choose a fellow"})
		return
	}
	if input.BuddyID == actor {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Choose another fellow"})
		return
	}
	var targetAvailable bool
	err := database.QueryRowContext(c.Request.Context(), `SELECT NOT is_suspended FROM users WHERE id=$1`, input.BuddyID).Scan(&targetAvailable)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fellow unavailable"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify fellow"})
		return
	}
	if !targetAvailable {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fellow unavailable"})
		return
	}
	var connectionID int64
	err = database.QueryRowContext(c.Request.Context(), `
		INSERT INTO buddy_connections(requester_id,recipient_id)
		VALUES($1,$2)
		ON CONFLICT DO NOTHING
		RETURNING id`, actor, input.BuddyID).Scan(&connectionID)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusConflict, gin.H{"error": "A buddy connection already exists"})
		return
	}
	if err != nil {
		log.Printf("request buddy: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not send buddy request"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"connection_id": connectionID, "status": "pending"})
}

// RespondBuddy allows only the request recipient to accept or decline once.
func RespondBuddy(c *gin.Context) {
	database, ok := buddyDatabase(c)
	if !ok {
		return
	}
	actor, ok := buddyActor(c, database)
	if !ok {
		return
	}
	var input struct {
		ConnectionID int64  `json:"connection_id"`
		Status       string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.ConnectionID <= 0 || (input.Status != "active" && input.Status != "declined") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Choose active or declined"})
		return
	}

	tx, err := database.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update buddy request"})
		return
	}
	defer tx.Rollback()
	var requesterID, recipientID int64
	var existingStatus string
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT requester_id, recipient_id, status
		FROM buddy_connections WHERE id=$1 FOR UPDATE`, input.ConnectionID).
		Scan(&requesterID, &recipientID, &existingStatus)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Buddy request not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load buddy request"})
		return
	}
	if recipientID != actor {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the recipient can respond"})
		return
	}
	if existingStatus != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "Buddy request already decided"})
		return
	}
	if input.Status == "active" {
		var requesterAvailable bool
		err = tx.QueryRowContext(c.Request.Context(), `SELECT NOT is_suspended FROM users WHERE id=$1`, requesterID).Scan(&requesterAvailable)
		if err != nil || !requesterAvailable {
			c.JSON(http.StatusConflict, gin.H{"error": "Requester is no longer available"})
			return
		}
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE buddy_connections SET status=$1,updated_at=NOW() WHERE id=$2`, input.Status, input.ConnectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update buddy request"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save buddy request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"connection_id": input.ConnectionID, "status": input.Status})
}
