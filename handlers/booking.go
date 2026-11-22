package handlers

import (
    "database/sql"
    "net/http"
    "strings"
    "time"
    "os"

    "github.com/gin-gonic/gin"
    "L2EStudyLink/email"
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

    input.Topic=strings.TrimSpace(input.Topic)
    date,dateErr:=time.Parse("2006-01-02",input.Date)
    parseClock:=func(value string)(time.Time,error){if len(value)==8&&strings.HasSuffix(value,":00"){value=value[:5]};return time.Parse("15:04",value)}
    start,startErr:=parseClock(input.StartTime);end,endErr:=parseClock(input.EndTime)
    if studentID<=0||input.TutorID<=0||dateErr!=nil||startErr!=nil||endErr!=nil||!start.Before(end)||len([]rune(input.Topic))<1||len([]rune(input.Topic))>200 {
        c.JSON(400,gin.H{"error":"Choose a valid date, published time slot and topic of 1–200 characters"});return
    }
    input.Date=date.Format("2006-01-02");input.StartTime=start.Format("15:04");input.EndTime=end.Format("15:04")
    tx,err:=db.BeginTx(c.Request.Context(),nil)
    if err!=nil {c.JSON(503,gin.H{"error":"Booking is temporarily unavailable"});return};defer tx.Rollback()
    // Lock both participants in a stable order, so concurrent requests cannot double-book either person.
    first,second:=input.TutorID,studentID;if first>second {first,second=second,first}
    for _,id:=range []int64{first,second}{if _,err=tx.ExecContext(c.Request.Context(),`SELECT pg_advisory_xact_lock(7241,$1::integer)`,id);err!=nil {c.JSON(503,gin.H{"error":"Could not check this slot"});return}}
    var active int
    if err=tx.QueryRowContext(c.Request.Context(),`SELECT count(*) FROM users WHERE id IN ($1,$2) AND COALESCE(is_suspended,FALSE)=FALSE`,input.TutorID,studentID).Scan(&active);err!=nil||active!=2 {c.JSON(400,gin.H{"error":"Both participants must have active accounts"});return}
    var future,published,conflict bool
    if err=tx.QueryRowContext(c.Request.Context(),`SELECT $1::date+$2::time > LOCALTIMESTAMP`,input.Date,input.StartTime).Scan(&future);err!=nil||!future {c.JSON(400,gin.H{"error":"Choose a session that starts in the future using the booking clock"});return}
    if err=tx.QueryRowContext(c.Request.Context(),`SELECT EXISTS(SELECT 1 FROM availability WHERE user_id=$1 AND day_of_week=EXTRACT(ISODOW FROM $2::date)::int-1 AND start_time=$3::time AND end_time=$4::time)`,input.TutorID,input.Date,input.StartTime,input.EndTime).Scan(&published);err!=nil||!published {c.JSON(400,gin.H{"error":"Choose a published tutoring availability slot"});return}
    if err=tx.QueryRowContext(c.Request.Context(),`SELECT EXISTS(SELECT 1 FROM bookings WHERE session_date=$1::date AND status NOT IN ('cancelled','no_show') AND (tutor_id IN ($2,$3) OR student_id IN ($2,$3)) AND start_time<$5::time AND end_time>$4::time)`,input.Date,input.TutorID,studentID,input.StartTime,input.EndTime).Scan(&conflict);err!=nil {c.JSON(503,gin.H{"error":"Could not check availability"});return}
    if conflict {c.JSON(409,gin.H{"error":"A participant already has a session overlapping this time. Choose another slot."});return}
    var bookingID int64
    err=tx.QueryRowContext(c.Request.Context(),`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,meeting_type) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,input.TutorID,studentID,input.Date,input.StartTime,input.EndTime,input.Topic,input.MeetingType).Scan(&bookingID)
    if err!=nil {c.JSON(500,gin.H{"error":"Failed to create booking"});return}
    if err=tx.Commit();err!=nil {c.JSON(500,gin.H{"error":"Could not confirm the booking request"});return}

    // Get tutor and student details
    var tutorName, tutorEmail, tutorDiscord, studentName, studentEmail, studentDiscord string
    db.QueryRow("SELECT name, email, COALESCE(discord_username, '') FROM users WHERE id = $1", input.TutorID).Scan(&tutorName, &tutorEmail, &tutorDiscord)
    db.QueryRow("SELECT name, email, COALESCE(discord_username, '') FROM users WHERE id = $1", studentID).Scan(&studentName, &studentEmail, &studentDiscord)

    // Send Discord notification
    notifications.SendBookingNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, input.Date, input.StartTime, input.Topic)

    // Send email notifications
    go func() {
        email.SendBookingNotification(tutorEmail, tutorName, tutorName, studentName, input.Date, input.StartTime, input.Topic)
        email.SendBookingNotification(studentEmail, studentName, tutorName, studentName, input.Date, input.StartTime, input.Topic)
    }()

    c.JSON(http.StatusCreated, gin.H{
        "message":    "Booking created successfully",
        "booking_id": bookingID,
    })
}

func GetMyBookings(c *gin.Context) {
    c.Header("Cache-Control","private, no-store")
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.Query(`
        SELECT b.id, b.tutor_id, t.name as tutor_name, b.student_id, s.name as student_name,
               b.session_date, b.start_time, b.end_time, b.topic, b.meeting_type, b.status,
               b.created_at, EXISTS(SELECT 1 FROM reviews r WHERE r.booking_id=b.id AND r.reviewer_id=$1),
               (b.session_date+b.start_time>LOCALTIMESTAMP),(b.session_date+b.end_time<=LOCALTIMESTAMP),current_setting('TimeZone')
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
        var reviewed,upcoming,ended bool
        var timezone string

        if err:=rows.Scan(&id, &tutorID, &tutorName, &studentID, &studentName,
            &sessionDate, &startTime, &endTime, &topic, &meetingType, &status, &createdAt,&reviewed,&upcoming,&ended,&timezone);err!=nil {c.JSON(500,gin.H{"error":"Failed to fetch bookings"});return}

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
            "reviewed_by_me": reviewed,
            "timezone": timezone,
            "can_cancel": upcoming && (status=="pending"||status=="confirmed"),
            "can_accept": upcoming && status=="pending" && role=="tutor",
            "can_decline": upcoming && status=="pending" && role=="tutor",
            "can_complete": ended && status=="confirmed" && role=="tutor",
            "can_no_show": ended && status=="confirmed" && role=="tutor",
            "can_review": status=="completed" && role=="student" && !reviewed,
        })
    }

    if err:=rows.Err();err!=nil {c.JSON(500,gin.H{"error":"Failed to fetch bookings"});return}
    if bookings==nil {bookings=[]gin.H{}}
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

    // Keep the stored local session date and time for notifications.
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
    
    result, err := db.Exec("UPDATE bookings SET status = 'cancelled' WHERE id = $1 AND status IN ('pending','confirmed') AND session_date + start_time > LOCALTIMESTAMP", bookingID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel booking"})
        return
    }
    changed,_:=result.RowsAffected()
    if changed==0 {c.JSON(http.StatusConflict,gin.H{"error":"Only upcoming pending or confirmed sessions can be cancelled"});return}

    // Get tutor and student details
    var tutorName, tutorEmail, tutorDiscord, studentName, studentEmail, studentDiscord string
    db.QueryRow("SELECT name, email, COALESCE(discord_username, '') FROM users WHERE id = $1", tutorID).Scan(&tutorName, &tutorEmail, &tutorDiscord)
    db.QueryRow("SELECT name, email, COALESCE(discord_username, '') FROM users WHERE id = $1", studentID).Scan(&studentName, &studentEmail, &studentDiscord)

    // Send Discord notification
    notifications.SendBookingCancelledNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, dateStr, timeStr)

    // Send email notifications
    go func() {
        email.SendBookingCancelledNotification(tutorEmail, tutorName, tutorName, studentName, dateStr, timeStr)
        email.SendBookingCancelledNotification(studentEmail, studentName, tutorName, studentName, dateStr, timeStr)
    }()

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
    
    result, err := db.Exec("UPDATE bookings SET status = $1 WHERE id = $2 AND status IN ('pending','confirmed') AND session_date + start_time > LOCALTIMESTAMP AND (($1='confirmed' AND status='pending') OR $1='cancelled')", input.Status, bookingID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
        return
    }
    changed,_:=result.RowsAffected()
    if changed==0 {c.JSON(http.StatusConflict,gin.H{"error":"Booking status can no longer be changed"});return}
    
    // Get tutor and student details
    var tutorName, tutorEmail, tutorDiscord, studentName, studentEmail, studentDiscord string
    db.QueryRow("SELECT name, email, COALESCE(discord_username, '') FROM users WHERE id = $1", tutorID).Scan(&tutorName, &tutorEmail, &tutorDiscord)
    db.QueryRow("SELECT name, email, COALESCE(discord_username, '') FROM users WHERE id = $1", studentID).Scan(&studentName, &studentEmail, &studentDiscord)
    
    // Parse date and time for display
    dateStr := sessionDate
    if len(dateStr) > 10 {
        dateStr = dateStr[:10]
    }
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
    
    if input.Status == "confirmed" {
        // Send Discord notification
        notifications.SendBookingAcceptedNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, dateStr, timeStr)
        
        // Send email notifications
        go func() {
            email.SendBookingAcceptedNotification(tutorEmail, tutorName, tutorName, studentName, dateStr, timeStr)
            email.SendBookingAcceptedNotification(studentEmail, studentName, tutorName, studentName, dateStr, timeStr)
        }()
    } else if input.Status == "cancelled" {
        // Send Discord notification
        notifications.SendBookingCancelledNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, dateStr, timeStr)
        
        // Send email notifications
        go func() {
            email.SendBookingCancelledNotification(tutorEmail, tutorName, tutorName, studentName, dateStr, timeStr)
            email.SendBookingCancelledNotification(studentEmail, studentName, tutorName, studentName, dateStr, timeStr)
        }()
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Booking " + input.Status})
}

// Booking dates/clock times are stored without offsets; expose their actual shared database clock.
func GetBookingClock(c *gin.Context) {
    var zone,today,now string
    err:=c.MustGet("db").(*sql.DB).QueryRowContext(c.Request.Context(),`SELECT current_setting('TimeZone'),to_char(CURRENT_DATE,'YYYY-MM-DD'),to_char(LOCALTIMESTAMP,'YYYY-MM-DD HH24:MI:SS')`).Scan(&zone,&today,&now)
    c.Header("Cache-Control","no-store")
    if err!=nil {c.JSON(503,gin.H{"error":"Booking clock is unavailable"});return}
    c.JSON(200,gin.H{"timezone":zone,"today":today,"now":now})
}
