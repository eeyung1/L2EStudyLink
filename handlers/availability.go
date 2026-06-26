package handlers

import (
    "database/sql"
    "net/http"

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

    // Wrap DELETE + INSERT in a transaction so a failed insert
    // never leaves the user with zero availability rows.
    tx, err := db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
        return
    }
    // If anything below fails, roll back — restoring the old rows.
    defer tx.Rollback()

    if _, err := tx.Exec("DELETE FROM availability WHERE user_id = $1", userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear availability"})
        return
    }

    for _, slot := range slots {
        startTime := slot.StartTime
        endTime := slot.EndTime

        // Normalise HH:MM → HH:MM:SS for Postgres TIME columns
        if len(startTime) == 5 {
            startTime = startTime + ":00"
        }
        if len(endTime) == 5 {
            endTime = endTime + ":00"
        }

        _, err := tx.Exec(`
            INSERT INTO availability (user_id, day_of_week, start_time, end_time)
            VALUES ($1, $2, $3, $4)
        `, userID, slot.DayOfWeek, startTime, endTime)

        if err != nil {
            // Rollback is called by defer — old rows are restored.
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save slot: " + err.Error()})
            return
        }
    }

    if err := tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit availability"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Availability saved successfully", "slots": slots})
}

func GetAvailability(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    rows, err := db.Query(`
        SELECT day_of_week,
               TO_CHAR(start_time, 'HH24:MI') AS start_time,
               TO_CHAR(end_time,   'HH24:MI') AS end_time
        FROM availability
        WHERE user_id = $1
        ORDER BY day_of_week, start_time
    `, userID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch availability"})
        return
    }
    defer rows.Close()

    slots := []AvailabilitySlot{}
    for rows.Next() {
        var slot AvailabilitySlot
        // Scan directly into strings — TO_CHAR already formatted them as HH:MM
        if err := rows.Scan(&slot.DayOfWeek, &slot.StartTime, &slot.EndTime); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read slot: " + err.Error()})
            return
        }
        slots = append(slots, slot)
    }

    c.JSON(http.StatusOK, slots)
}
