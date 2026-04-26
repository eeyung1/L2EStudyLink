package handlers

import (
    "database/sql"
    "net/http"
    "strings"
    "time"
    "os"

    "github.com/gin-gonic/gin"
    "L2EStudyLink/notifications"
)

type BookingRequest struct {
    TutorID     int64  `json:"tutor_id" binding:"required"`
    Date        string `json:"date" binding:"required"`
    StartTime   string `json:"start_time" binding:"required"`
    EndTime     string `json:"end_time" binding:"required"`
    Topic       string `json:"topic" binding:"required"`
    MeetingType string `json:"meeting_type" binding:"required,oneof=in_person online"`
}

func CreateBooking(c *gin.Context) {
    studentID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    notifications.Init(os.Getenv("DISCORD_WEBHOOK_URL"))

    var input BookingRequest
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if input.TutorID == studentID {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot book yourself"})
        return
    }

    var count int
    err := db.QueryRow(`
        SELECT COUNT(*) FROM bookings 
        WHERE tutor_id = $1 AND session_date = $2 AND start_time = $3 
        AND status NOT IN ('cancelled', 'no_show')
    `, input.TutorID, input.Date, input.StartTime).Scan(&count)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check availability"})
        return
    }

    if count > 0 {
        c.JSON(http.StatusConflict, gin.H{"error": "Slot already booked"})
        return
    }

    var bookingID int64
    err = db.QueryRow(`
        INSERT INTO bookings (tutor_id, student_id, session_date, start_time, end_time, topic, meeting_type)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, input.TutorID, studentID, input.Date, input.StartTime, input.EndTime, input.Topic, input.MeetingType).Scan(&bookingID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
        return
    }

    var tutorName, tutorDiscord, studentName, studentDiscord string
    db.QueryRow("SELECT name, COALESCE(discord_username, '') FROM users WHERE id = $1", input.TutorID).Scan(&tutorName, &tutorDiscord)
    db.QueryRow("SELECT name, COALESCE(discord_username, '') FROM users WHERE id = $1", studentID).Scan(&studentName, &studentDiscord)

    notifications.SendBookingNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, input.Date, input.StartTime, input.Topic)

    c.JSON(http.StatusCreated, gin.H{
        "message":    "Booking created successfully",
        "booking_id": bookingID,
    })
}

func GetMyBookings(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.Query(`
        SELECT b.id, b.tutor_id, t.name as tutor_name, b.student_id, s.name as student_name,
               b.session_date, b.start_time, b.end_time, b.topic, b.meeting_type, b.status,
               b.created_at
        FROM bookings b
        JOIN users t ON b.tutor_id = t.id
        JOIN users s ON b.student_id = s.id
        WHERE b.tutor_id = $1 OR b.student_id = $1
        ORDER BY b.session_date DESC, b.start_time DESC
    `, userID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
        return
    }
    defer rows.Close()

    var bookings []gin.H
    for rows.Next() {
        var id, tutorID, studentID int64
        var tutorName, studentName, topic, meetingType, status string
        var sessionDate, startTime, endTime string
        var createdAt time.Time

        rows.Scan(&id, &tutorID, &tutorName, &studentID, &studentName,
            &sessionDate, &startTime, &endTime, &topic, &meetingType, &status, &createdAt)

        role := "student"
        if tutorID == userID {
            role = "tutor"
        }

        bookings = append(bookings, gin.H{
            "id":           id,
            "role":         role,
            "tutor_id":     tutorID,
            "tutor_name":   tutorName,
            "student_id":   studentID,
            "student_name": studentName,
            "date":         sessionDate,
            "start_time":   startTime,
            "end_time":     endTime,
            "topic":        topic,
            "meeting_type": meetingType,
            "status":       status,
            "created_at":   createdAt,
        })
    }

    c.JSON(http.StatusOK, bookings)
}

func CancelBooking(c *gin.Context) {
    userID := c.GetInt64("user_id")
    bookingID := c.Param("id")
    db := c.MustGet("db").(*sql.DB)

    notifications.Init(os.Getenv("DISCORD_WEBHOOK_URL"))

    var tutorID, studentID int64
    var sessionDate, startTime string
    err := db.QueryRow(`
        SELECT tutor_id, student_id, session_date, start_time 
        FROM bookings WHERE id = $1
    `, bookingID).Scan(&tutorID, &studentID, &sessionDate, &startTime)

    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
        return
    }

    if tutorID != userID && studentID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
        return
    }

    // Parse date (handle both YYYY-MM-DD and timestamp formats)
    dateStr := sessionDate
    if len(dateStr) > 10 {
        dateStr = dateStr[:10]
    }
    
    // Parse start time (handle both HH:MM and timestamp formats)
    timeStr := startTime
    if len(timeStr) > 5 {
        if strings.Contains(timeStr, "T") {
            parts := strings.Split(timeStr, "T")
            if len(parts) > 1 {
                timeStr = parts[1][:5]
            }
        } else {
            timeStr = timeStr[:5]
        }
    }
    
    sessionDateTime, err := time.Parse("2006-01-02 15:04", dateStr+" "+timeStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format: " + err.Error()})
        return
    }

    if time.Until(sessionDateTime) < 2*time.Hour {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Must cancel at least 2 hours before session"})
        return
    }

    _, err = db.Exec("UPDATE bookings SET status = 'cancelled' WHERE id = $1", bookingID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel booking"})
        return
    }

    var tutorName, studentName, tutorDiscord, studentDiscord string
    db.QueryRow("SELECT name, COALESCE(discord_username, '') FROM users WHERE id = $1", tutorID).Scan(&tutorName, &tutorDiscord)
    db.QueryRow("SELECT name, COALESCE(discord_username, '') FROM users WHERE id = $1", studentID).Scan(&studentName, &studentDiscord)

    notifications.SendBookingCancelledNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, dateStr, timeStr)

    c.JSON(http.StatusOK, gin.H{"message": "Booking cancelled successfully"})
}

func UpdateBookingStatus(c *gin.Context) {
    userID := c.GetInt64("user_id")
    bookingID := c.Param("id")
    db := c.MustGet("db").(*sql.DB)
    
    notifications.Init(os.Getenv("DISCORD_WEBHOOK_URL"))
    
    var input struct {
        Status string `json:"status" binding:"required,oneof=confirmed cancelled"`
    }
    
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    var tutorID, studentID int64
    var sessionDate, startTime string
    err := db.QueryRow("SELECT tutor_id, student_id, session_date, start_time FROM bookings WHERE id = $1", bookingID).Scan(&tutorID, &studentID, &sessionDate, &startTime)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
        return
    }
    
    if tutorID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Only the tutor can confirm bookings"})
        return
    }
    
    _, err = db.Exec("UPDATE bookings SET status = $1 WHERE id = $2", input.Status, bookingID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
        return
    }
    
    var tutorName, studentName, tutorDiscord, studentDiscord string
    db.QueryRow("SELECT name, COALESCE(discord_username, '') FROM users WHERE id = $1", tutorID).Scan(&tutorName, &tutorDiscord)
    db.QueryRow("SELECT name, COALESCE(discord_username, '') FROM users WHERE id = $1", studentID).Scan(&studentName, &studentDiscord)
    
    if input.Status == "confirmed" {
        notifications.SendBookingAcceptedNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, sessionDate, startTime)
    } else if input.Status == "cancelled" {
        notifications.SendBookingCancelledNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, sessionDate, startTime)
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Booking " + input.Status})
}
