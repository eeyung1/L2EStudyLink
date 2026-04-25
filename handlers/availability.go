package handlers

import (
    "database/sql"
    "net/http"

    "github.com/gin-gonic/gin"
)

type AvailabilitySlot struct {
    DayOfWeek int    `json:"day_of_week"` // 0=Monday, 6=Sunday
    StartTime string `json:"start_time"`  // "14:00"
    EndTime   string `json:"end_time"`    // "16:00"
}

func SetAvailability(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var slots []AvailabilitySlot
    if err := c.ShouldBindJSON(&slots); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Delete existing availability
    _, err := db.Exec("DELETE FROM availability WHERE user_id = $1", userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing availability"})
        return
    }

    // Insert new slots
    for _, slot := range slots {
        _, err := db.Exec(`
            INSERT INTO availability (user_id, day_of_week, start_time, end_time) 
            VALUES ($1, $2, $3, $4)
        `, userID, slot.DayOfWeek, slot.StartTime, slot.EndTime)
        
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save availability"})
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{"message": "Availability saved successfully"})
}

func GetAvailability(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.Query(`
        SELECT day_of_week, start_time, end_time 
        FROM availability 
        WHERE user_id = $1 
        ORDER BY day_of_week, start_time
    `, userID)
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch availability"})
        return
    }
    defer rows.Close()

    var slots []AvailabilitySlot
    for rows.Next() {
        var slot AvailabilitySlot
        rows.Scan(&slot.DayOfWeek, &slot.StartTime, &slot.EndTime)
        slots = append(slots, slot)
    }

    c.JSON(http.StatusOK, slots)
}
