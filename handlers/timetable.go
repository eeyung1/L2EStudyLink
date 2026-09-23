package handlers

import (
    "database/sql"
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
)

type TimeBlock struct {
    ID        int    `json:"id"`
    DayOfWeek int    `json:"day_of_week"`
    StartTime string `json:"start_time"`
    EndTime   string `json:"end_time"`
    Activity  string `json:"activity"`
    Goal      string `json:"goal"`
}

type ActivityLog struct {
    ID          int    `json:"id"`
    TimetableID int    `json:"timetable_id"`
    LogDate     string `json:"log_date"`
    Summary     string `json:"summary"`
    Challenges  string `json:"challenges"`
    Learnings   string `json:"learnings"`
    Completed   bool   `json:"completed"`
}

func GetTimetable(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.QueryContext(c.Request.Context(), `
        SELECT id, day_of_week, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), activity, COALESCE(goal, '')
        FROM timetable
        WHERE user_id = $1
        ORDER BY day_of_week, start_time
    `, userID)

    if err != nil {
        log.Printf("GetTimetable query: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load timetable"})
        return
    }
    defer rows.Close()

    blocks := make([]TimeBlock, 0)
    for rows.Next() {
        var b TimeBlock
        if err := rows.Scan(&b.ID, &b.DayOfWeek, &b.StartTime, &b.EndTime, &b.Activity, &b.Goal); err != nil {
            log.Printf("GetTimetable scan: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load timetable"})
            return
        }
        blocks = append(blocks, b)
    }

    if err := rows.Err(); err != nil {
        log.Printf("GetTimetable rows: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load timetable"})
        return
    }
    c.JSON(http.StatusOK, blocks)
}

func AddTimeBlock(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var input TimeBlock
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Do NOT use the ID from frontend - let database generate it
    var id int64
    err := db.QueryRow(`
        INSERT INTO timetable (user_id, day_of_week, start_time, end_time, activity, goal)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `, userID, input.DayOfWeek, input.StartTime, input.EndTime, input.Activity, input.Goal).Scan(&id)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add: " + err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Time block added", "id": id})
}

func DeleteTimeBlock(c *gin.Context) {
    userID := c.GetInt64("user_id")
    blockID := c.Param("id")
    db := c.MustGet("db").(*sql.DB)

    result, err := db.Exec("DELETE FROM timetable WHERE id = $1 AND user_id = $2", blockID, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Time block not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

func AddReflection(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var input ActivityLog
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Check if reflection already exists for this timetable_id and log_date
    var existingID int64
    err := db.QueryRow(`
        SELECT id FROM activity_logs 
        WHERE timetable_id = $1 AND log_date = $2 AND user_id = $3
    `, input.TimetableID, input.LogDate, userID).Scan(&existingID)

    if err == nil {
        // Update existing reflection
        _, err = db.Exec(`
            UPDATE activity_logs 
            SET summary = $1, challenges = $2, learnings = $3, completed = true
            WHERE id = $4
        `, input.Summary, input.Challenges, input.Learnings, existingID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reflection"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"message": "Reflection updated", "id": existingID})
        return
    }

    // Insert new reflection - let database auto-generate ID
    var id int64
    err = db.QueryRow(`
        INSERT INTO activity_logs (timetable_id, user_id, log_date, summary, challenges, learnings, completed)
        VALUES ($1, $2, $3, $4, $5, $6, true)
        RETURNING id
    `, input.TimetableID, userID, input.LogDate, input.Summary, input.Challenges, input.Learnings).Scan(&id)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save reflection: " + err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Reflection saved", "id": id})
}

func GetReflections(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.QueryContext(c.Request.Context(), `
        SELECT al.id, al.timetable_id, to_char(al.log_date, 'YYYY-MM-DD'), al.summary,
               al.challenges, al.learnings, al.completed,
               t.activity, t.day_of_week, to_char(t.start_time, 'HH24:MI'), to_char(t.end_time, 'HH24:MI')
        FROM activity_logs al
        JOIN timetable t ON al.timetable_id = t.id
        WHERE al.user_id = $1
        ORDER BY al.log_date DESC
    `, userID)

    if err != nil {
        log.Printf("GetReflections query: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load reflections"})
        return
    }
    defer rows.Close()

    reflections := make([]gin.H, 0)
    for rows.Next() {
        var id, timetableID int
        var logDate string
        var summary, challenges, learnings string
        var completed bool
        var activity string
        var dayOfWeek int
        var startTime, endTime string

        if err := rows.Scan(&id, &timetableID, &logDate, &summary, &challenges, &learnings, &completed,
            &activity, &dayOfWeek, &startTime, &endTime); err != nil {
            log.Printf("GetReflections scan: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load reflections"})
            return
        }

        reflections = append(reflections, gin.H{
            "id":           id,
            "timetable_id": timetableID,
            "log_date":     logDate,
            "summary":      summary,
            "challenges":   challenges,
            "learnings":    learnings,
            "completed":    completed,
            "activity":     activity,
            "day_of_week":  dayOfWeek,
            "start_time":   startTime,
            "end_time":     endTime,
        })
    }

    if err := rows.Err(); err != nil {
        log.Printf("GetReflections rows: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load reflections"})
        return
    }
    c.JSON(http.StatusOK, reflections)
}
