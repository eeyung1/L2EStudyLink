package handlers

import (
    "database/sql"
    "net/http"
    "time"

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

// Get user's timetable for current week
func GetTimetable(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.Query(`
        SELECT id, day_of_week, start_time, end_time, activity, COALESCE(goal, '')
        FROM timetable
        WHERE user_id = $1
        ORDER BY day_of_week, start_time
    `, userID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch timetable"})
        return
    }
    defer rows.Close()

    var blocks []TimeBlock
    for rows.Next() {
        var b TimeBlock
        var startTime, endTime time.Time
        rows.Scan(&b.ID, &b.DayOfWeek, &startTime, &endTime, &b.Activity, &b.Goal)
        b.StartTime = startTime.Format("15:04")
        b.EndTime = endTime.Format("15:04")
        blocks = append(blocks, b)
    }

    c.JSON(http.StatusOK, blocks)
}

// Add a time block to timetable
func AddTimeBlock(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var input TimeBlock
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var id int64
    err := db.QueryRow(`
        INSERT INTO timetable (user_id, day_of_week, start_time, end_time, activity, goal)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `, userID, input.DayOfWeek, input.StartTime, input.EndTime, input.Activity, input.Goal).Scan(&id)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add time block"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Time block added", "id": id})
}

// Delete a time block
func DeleteTimeBlock(c *gin.Context) {
    userID := c.GetInt64("user_id")
    blockID := c.Param("id")
    db := c.MustGet("db").(*sql.DB)

    result, err := db.Exec("DELETE FROM timetable WHERE id = $1 AND user_id = $2", blockID, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
        return
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Time block not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// Add reflection log for a completed time block
func AddReflection(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var input ActivityLog
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var id int64
    err := db.QueryRow(`
        INSERT INTO activity_logs (timetable_id, user_id, log_date, summary, challenges, learnings, completed)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, input.TimetableID, userID, input.LogDate, input.Summary, input.Challenges, input.Learnings, true).Scan(&id)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save reflection"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Reflection saved", "id": id})
}

// Get reflections for a user
func GetReflections(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.Query(`
        SELECT al.id, al.timetable_id, al.log_date, al.summary, al.challenges, al.learnings, al.completed,
               t.activity, t.day_of_week, t.start_time, t.end_time
        FROM activity_logs al
        JOIN timetable t ON al.timetable_id = t.id
        WHERE al.user_id = $1
        ORDER BY al.log_date DESC
    `, userID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reflections"})
        return
    }
    defer rows.Close()

    var reflections []gin.H
    for rows.Next() {
        var id, timetableID int
        var logDate string
        var summary, challenges, learnings string
        var completed bool
        var activity string
        var dayOfWeek int
        var startTime, endTime time.Time

        rows.Scan(&id, &timetableID, &logDate, &summary, &challenges, &learnings, &completed,
            &activity, &dayOfWeek, &startTime, &endTime)

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
            "start_time":   startTime.Format("15:04"),
            "end_time":     endTime.Format("15:04"),
        })
    }

    c.JSON(http.StatusOK, reflections)
}
