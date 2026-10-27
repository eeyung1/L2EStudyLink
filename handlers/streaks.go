package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type StreakStats struct {
	CurrentStreak    int            `json:"current_streak"`
	LongestStreak    int            `json:"longest_streak"`
	TotalActiveDays  int            `json:"total_active_days"`
	TotalReflections int            `json:"total_reflections"`
	TotalSessions    int            `json:"total_sessions"`
	Today            string         `json:"today"`
	DailyCounts      map[string]int `json:"daily_counts"`
}

func GetUserStreaks(c *gin.Context) {
	userID := c.GetInt64("user_id")
	dbVal, exists := c.Get("db")
	if !exists || dbVal == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}
	db, ok := dbVal.(*sql.DB)
	if !ok || db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}

	dailyCounts := make(map[string]int)
	totalReflections := 0
	totalSessions := 0

	// 1. Query reflection activity dates
	reflectionRows, err := db.QueryContext(c.Request.Context(), `
		SELECT to_char(log_date, 'YYYY-MM-DD') AS day, COUNT(*)
		FROM activity_logs
		WHERE user_id = $1
		GROUP BY 1
	`, userID)
	if err == nil {
		defer reflectionRows.Close()
		for reflectionRows.Next() {
			var day string
			var count int
			if scanErr := reflectionRows.Scan(&day, &count); scanErr == nil && day != "" {
				dailyCounts[day] += count
				totalReflections += count
			}
		}
	} else {
		log.Printf("GetUserStreaks activity_logs query error: %v", err)
	}

	// 2. Query completed booking session dates
	sessionRows, err := db.QueryContext(c.Request.Context(), `
		SELECT to_char(session_date, 'YYYY-MM-DD') AS day, COUNT(*)
		FROM bookings
		WHERE (tutor_id = $1 OR student_id = $1) AND status = 'completed'
		GROUP BY 1
	`, userID)
	if err == nil {
		defer sessionRows.Close()
		for sessionRows.Next() {
			var day string
			var count int
			if scanErr := sessionRows.Scan(&day, &count); scanErr == nil && day != "" {
				dailyCounts[day] += count
				totalSessions += count
			}
		}
	} else {
		log.Printf("GetUserStreaks bookings query error: %v", err)
	}

	// Determine client's "today"
	todayStr := c.Query("today")
	todayDate, parseErr := time.Parse("2006-01-02", todayStr)
	if parseErr != nil {
		todayDate = time.Now().UTC()
		todayStr = todayDate.Format("2006-01-02")
	} else {
		todayDate = todayDate.UTC()
	}

	currentStreak, longestStreak := CalculateStreaks(dailyCounts, todayDate)

	stats := StreakStats{
		CurrentStreak:    currentStreak,
		LongestStreak:    longestStreak,
		TotalActiveDays:  len(dailyCounts),
		TotalReflections: totalReflections,
		TotalSessions:    totalSessions,
		Today:            todayStr,
		DailyCounts:      dailyCounts,
	}

	c.JSON(http.StatusOK, stats)
}

// CalculateStreaks computes current active streak and all-time longest streak
func CalculateStreaks(dailyCounts map[string]int, today time.Time) (currentStreak int, longestStreak int) {
	if len(dailyCounts) == 0 {
		return 0, 0
	}

	// Parse all unique active dates
	var dates []time.Time
	for dStr, count := range dailyCounts {
		if count > 0 {
			if parsed, err := time.Parse("2006-01-02", dStr); err == nil {
				dates = append(dates, parsed.UTC())
			}
		}
	}

	if len(dates) == 0 {
		return 0, 0
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	// Calculate Longest Streak
	longest := 1
	currentSeq := 1
	for i := 1; i < len(dates); i++ {
		prev := dates[i-1]
		curr := dates[i]
		diffHours := curr.Sub(prev).Hours()
		// Round to nearest day to account for daylight savings if any
		diffDays := int((diffHours + 1) / 24)
		if diffDays == 1 {
			currentSeq++
			if currentSeq > longest {
				longest = currentSeq
			}
		} else if diffDays > 1 {
			currentSeq = 1
		}
	}
	longestStreak = longest

	// Calculate Current Streak
	todayMid := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	todayStr := todayMid.Format("2006-01-02")
	yesterdayStr := todayMid.AddDate(0, 0, -1).Format("2006-01-02")

	startFrom := todayMid
	if dailyCounts[todayStr] > 0 {
		startFrom = todayMid
	} else if dailyCounts[yesterdayStr] > 0 {
		startFrom = todayMid.AddDate(0, 0, -1)
	} else {
		return 0, longestStreak
	}

	streak := 0
	checkDate := startFrom
	for {
		cStr := checkDate.Format("2006-01-02")
		if dailyCounts[cStr] > 0 {
			streak++
			checkDate = checkDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	currentStreak = streak
	return currentStreak, longestStreak
}
