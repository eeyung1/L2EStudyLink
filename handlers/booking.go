package handlers

import (
    "database/sql"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

type BookingRequest struct {
    TutorID     int64  `json:"tutor_id" binding:"required"`
    Date        string `json:"date" binding:"required"`        // YYYY-MM-DD
    StartTime   string `json:"start_time" binding:"required"`  // HH:MM
    EndTime     string `json:"end_time" binding:"required"`
    Topic       string `json:"topic" binding:"required"`
    MeetingType string `json:"meeting_type" binding:"required,oneof=in_person online"`
}

func CreateBooking(c *gin.Context) {
    studentID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var input BookingRequest
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Prevent self-booking
    if input.TutorID == studentID {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot book yourself"})
        return
    }

    // Check if slot is already booked
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

    // Create booking
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

    c.JSON(http.StatusCreated, gin.H{
        "message":    "Booking created successfully",
        "booking_id": bookingID,
    })
}

func GetMyBookings(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    // Get bookings where user is tutor OR student
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

    // Get booking details
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

    // Check if user is part of this booking
    if tutorID != userID && studentID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
        return
    }

    // Convert session_date + start_time to time.Time for comparison
    sessionDateTime, err := time.Parse("2006-01-02 15:04", sessionDate+" "+startTime)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid date format"})
        return
    }

    // Check if cancellation is >2 hours before session
    if time.Until(sessionDateTime) < 2*time.Hour {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Must cancel at least 2 hours before session"})
        return
    }

    // Update booking status
    _, err = db.Exec("UPDATE bookings SET status = 'cancelled' WHERE id = $1", bookingID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel booking"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Booking cancelled successfully"})
}

func UpdateBookingStatus(c *gin.Context) {
    userID := c.GetInt64("user_id")
    bookingID := c.Param("id")
    db := c.MustGet("db").(*sql.DB)
    
    var input struct {
        Status string `json:"status" binding:"required,oneof=confirmed cancelled"`
    }
    
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Check if user is the tutor for this booking
    var tutorID int64
    err := db.QueryRow("SELECT tutor_id FROM bookings WHERE id = $1", bookingID).Scan(&tutorID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
        return
    }
    
    if tutorID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Only the tutor can confirm bookings"})
        return
    }
    
    // Update status
    _, err = db.Exec("UPDATE bookings SET status = $1 WHERE id = $2", input.Status, bookingID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Booking " + input.Status})
}
