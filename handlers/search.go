package handlers

import (
    "database/sql"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

// SearchTutorsWithAvailability returns tutors matching a skill, with their
// availability already embedded. One SQL query + in-Go grouping = no N+1.
func SearchTutorsWithAvailability(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    skillQuery := c.Query("skill")
    if skillQuery == "" {
        c.JSON(http.StatusOK, []gin.H{})
        return
    }

    // Single JOIN query: fetch every (tutor, availability_slot) row at once.
    // availability columns are nullable because of the LEFT JOIN — a tutor
    // with no availability rows still appears once with NULL slot columns.
    rows, err := db.Query(`
        SELECT
            u.id,
            u.name,
            u.bio,
            u.rating,
            u.total_reviews,
            COALESCE(u.discord_username, ''),
            a.day_of_week,
            a.start_time,
            a.end_time
        FROM users u
        JOIN skills s ON u.id = s.user_id
        LEFT JOIN availability a ON u.id = a.user_id
        WHERE s.skill_name ILIKE $1
        ORDER BY u.id, a.day_of_week, a.start_time
    `, "%"+skillQuery+"%")

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    // Group rows by tutor in Go — avoids json_agg / CTE complexity entirely.
    type AvailSlot struct {
        DayOfWeek int    `json:"day_of_week"`
        StartTime string `json:"start_time"`
        EndTime   string `json:"end_time"`
    }
    type Tutor struct {
        ID              int64       `json:"id"`
        Name            string      `json:"name"`
        Bio             string      `json:"bio"`
        Rating          float64     `json:"rating"`
        TotalReviews    int         `json:"total_reviews"`
        DiscordUsername string      `json:"discord_username"`
        Availability    []AvailSlot `json:"availability"`
    }

    tutorMap := map[int64]*Tutor{}
    tutorOrder := []int64{} // preserve ORDER BY u.id result order

    for rows.Next() {
        var (
            id      int64
            name, bio, discord string
            rating  float64
            reviews int
            // nullable availability columns
            dayOfWeek         sql.NullInt64
            rawStart, rawEnd  sql.NullString
        )
        if err := rows.Scan(&id, &name, &bio, &rating, &reviews, &discord,
            &dayOfWeek, &rawStart, &rawEnd); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        tutor, exists := tutorMap[id]
        if !exists {
            tutor = &Tutor{
                ID:              id,
                Name:            name,
                Bio:             bio,
                Rating:          rating,
                TotalReviews:    reviews,
                DiscordUsername: discord,
                Availability:    []AvailSlot{},
            }
            tutorMap[id] = tutor
            tutorOrder = append(tutorOrder, id)
        }

        if dayOfWeek.Valid && rawStart.Valid && rawEnd.Valid {
            tutor.Availability = append(tutor.Availability, AvailSlot{
                DayOfWeek: int(dayOfWeek.Int64),
                StartTime: normaliseTime(rawStart.String),
                EndTime:   normaliseTime(rawEnd.String),
            })
        }
    }

    result := make([]*Tutor, 0, len(tutorOrder))
    for _, id := range tutorOrder {
        result = append(result, tutorMap[id])
    }

    c.JSON(http.StatusOK, result)
}

// normaliseTime trims a time value to HH:MM regardless of whether Postgres
// returns it as "15:04", "15:04:05", or a full timestamp string.
func normaliseTime(t string) string {
    if strings.Contains(t, "T") {
        parts := strings.SplitN(t, "T", 2)
        if len(parts) == 2 {
            t = strings.TrimSuffix(parts[1], "Z")
        }
    }
    if len(t) > 5 {
        return t[:5]
    }
    return t
}

func SearchTutors(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    skillQuery := c.Query("skill")
    if skillQuery == "" {
        c.JSON(http.StatusOK, []gin.H{})
        return
    }

    rows, err := db.Query(`
        SELECT DISTINCT u.id, u.name, u.bio, u.rating, u.total_reviews, COALESCE(u.discord_username, '')
        FROM users u
        JOIN skills s ON u.id = s.user_id
        WHERE s.skill_name ILIKE $1
        ORDER BY u.id
    `, "%"+skillQuery+"%")

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var tutors []gin.H
    for rows.Next() {
        var id int64
        var name, bio, discord string
        var rating float64
        var totalReviews int
        rows.Scan(&id, &name, &bio, &rating, &totalReviews, &discord)
        tutors = append(tutors, gin.H{
            "id":               id,
            "name":             name,
            "bio":              bio,
            "rating":           rating,
            "total_reviews":    totalReviews,
            "discord_username": discord,
        })
    }
    c.JSON(http.StatusOK, tutors)
}

func GetTutorProfile(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    tutorID := c.Param("id")
    
    var user struct {
        ID              int64
        Name            string
        Email           string
        DiscordUsername sql.NullString
        Bio             sql.NullString
        Rating          float64
        TotalReviews    int
        TotalSessions   int
    }
    
    err := db.QueryRow(`
        SELECT id, name, email, discord_username, bio, rating, total_reviews, total_sessions 
        FROM users WHERE id = $1
    `, tutorID).Scan(
        &user.ID, &user.Name, &user.Email, &user.DiscordUsername,
        &user.Bio, &user.Rating, &user.TotalReviews, &user.TotalSessions,
    )
    
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Tutor not found"})
        return
    }
    
    // Get skills
    skills := []gin.H{}
    skillRows, err := db.Query(`SELECT skill_name, proficiency FROM skills WHERE user_id = $1`, tutorID)
    if err == nil {
        defer skillRows.Close()
        for skillRows.Next() {
            var skillName, proficiency string
            skillRows.Scan(&skillName, &proficiency)
            skills = append(skills, gin.H{
                "name":        skillName,
                "proficiency": proficiency,
            })
        }
    }
    
    // Get availability - extract just the time part from timestamp
    availability := []gin.H{}
    availRows, err := db.Query(`SELECT day_of_week, start_time, end_time FROM availability WHERE user_id = $1`, tutorID)
    if err == nil {
        defer availRows.Close()
        for availRows.Next() {
            var dayOfWeek int
            var startTime, endTime string
            availRows.Scan(&dayOfWeek, &startTime, &endTime)
            
            // Extract just HH:MM from various formats
            startTimeStr := startTime
            endTimeStr := endTime
            
            // If it's a full timestamp like "0000-01-01T10:00:00Z", extract the time part
            if strings.Contains(startTimeStr, "T") {
                parts := strings.Split(startTimeStr, "T")
                if len(parts) > 1 {
                    timePart := strings.Split(parts[1], "Z")[0]
                    startTimeStr = timePart[:5]
                }
            } else if len(startTimeStr) > 5 {
                startTimeStr = startTimeStr[:5]
            }
            
            if strings.Contains(endTimeStr, "T") {
                parts := strings.Split(endTimeStr, "T")
                if len(parts) > 1 {
                    timePart := strings.Split(parts[1], "Z")[0]
                    endTimeStr = timePart[:5]
                }
            } else if len(endTimeStr) > 5 {
                endTimeStr = endTimeStr[:5]
            }
            
            availability = append(availability, gin.H{
                "day_of_week": dayOfWeek,
                "start_time":  startTimeStr,
                "end_time":    endTimeStr,
            })
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "id":               user.ID,
        "name":             user.Name,
        "email":            user.Email,
        "discord_username": user.DiscordUsername.String,
        "bio":              user.Bio.String,
        "rating":           user.Rating,
        "total_reviews":    user.TotalReviews,
        "total_sessions":   user.TotalSessions,
        "skills":           skills,
        "availability":     availability,
    })
}