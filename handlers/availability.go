package handlers

import (
    "database/sql"
    "net/http"
    "log"
    "time"

    "github.com/gin-gonic/gin"
)

type AvailabilitySlot struct {
    DayOfWeek int    `json:"day_of_week"`
    StartTime string `json:"start_time"`
    EndTime   string `json:"end_time"`
}

func SetAvailability(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var slots []AvailabilitySlot
    if err := c.ShouldBindJSON(&slots); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    for _, slot := range slots {
        start, startErr := time.Parse("15:04",slot.StartTime)
        end, endErr := time.Parse("15:04",slot.EndTime)
        if slot.DayOfWeek < 0 || slot.DayOfWeek > 6 || startErr != nil || endErr != nil || !start.Before(end) || len(slot.StartTime)!=5 || len(slot.EndTime)!=5 {
            c.JSON(http.StatusBadRequest,gin.H{"error":"Choose a valid day and start/end times to the minute; end must follow start"});return
        }
    }
    tx, err := db.BeginTx(c.Request.Context(),nil)
    if err != nil {c.JSON(500,gin.H{"error":"Failed to save availability"});return}
    defer tx.Rollback()
    _, err = tx.ExecContext(c.Request.Context(),"DELETE FROM availability WHERE user_id = $1", userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear availability"})
        return
    }

    // Insert new slots
    for _, slot := range slots {
        // Ensure time format is HH:MM:SS
        startTime := slot.StartTime
        endTime := slot.EndTime
        
        // Add seconds if missing
        if len(startTime) == 5 {
            startTime = startTime + ":00"
        }
        if len(endTime) == 5 {
            endTime = endTime + ":00"
        }
        
        _, err := tx.ExecContext(c.Request.Context(),`
            INSERT INTO availability (user_id, day_of_week, start_time, end_time) 
            VALUES ($1, $2, $3, $4)
        `, userID, slot.DayOfWeek, startTime, endTime)
        
        if err != nil {
            log.Printf("save availability for user %d: %v",userID,err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save your hours. Please try again."})
            return
        }
    }

    if err:=tx.Commit();err!=nil {c.JSON(500,gin.H{"error":"Failed to save availability"});return}
    c.JSON(http.StatusOK, gin.H{"message": "Availability saved successfully", "slots": slots})
}

func GetAvailability(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.QueryContext(c.Request.Context(), `
        SELECT day_of_week, CAST(start_time AS TEXT), CAST(end_time AS TEXT)
        FROM availability
        WHERE user_id = $1
        ORDER BY day_of_week, start_time
    `, userID)
    if err != nil {
        log.Printf("fetch availability for user %d: %v", userID, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Availability is temporarily unavailable"})
        return
    }
    defer rows.Close()

    slots := []AvailabilitySlot{}
    for rows.Next() {
        var slot AvailabilitySlot
        var startTime, endTime string
        if err := rows.Scan(&slot.DayOfWeek, &startTime, &endTime); err != nil {
            log.Printf("scan availability for user %d: %v", userID, err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read availability"})
            return
        }
        if len(startTime) < 5 || len(endTime) < 5 {
            log.Printf("invalid availability time for user %d", userID)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read availability"})
            return
        }
        slot.StartTime = startTime[:5]
        slot.EndTime = endTime[:5]
        slots = append(slots, slot)
    }
    if err := rows.Err(); err != nil {
        log.Printf("iterate availability for user %d: %v", userID, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read availability"})
        return
    }
    c.JSON(http.StatusOK, slots)
}
