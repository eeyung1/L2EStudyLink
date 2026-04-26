package handlers

import (
    "database/sql"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

func SearchTutors(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    
    skillQuery := c.Query("skill")
    if skillQuery == "" {
        c.JSON(http.StatusOK, []gin.H{})
        return
    }
    
    minRating := c.DefaultQuery("min_rating", "0")
    minRatingFloat, _ := strconv.ParseFloat(minRating, 64)
    
    // PostgreSQL uses boolean type, but our column is INTEGER (0/1)
    query := `
        SELECT DISTINCT u.id, u.name, u.bio, u.rating, u.total_reviews, COALESCE(u.discord_username, '')
        FROM users u
        JOIN skills s ON u.id = s.user_id
        WHERE s.skill_name ILIKE $1
        AND u.rating >= $2
        AND u.is_suspended = 0
        ORDER BY u.rating DESC, u.total_reviews DESC
        LIMIT 20
    `
    
    rows, err := db.Query(query, "%"+skillQuery+"%", minRatingFloat)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed: " + err.Error()})
        return
    }
    defer rows.Close()
    
    var tutors []gin.H
    for rows.Next() {
        var id int64
        var name, bio, discordUsername string
        var rating float64
        var totalReviews int
        
        err := rows.Scan(&id, &name, &bio, &rating, &totalReviews, &discordUsername)
        if err != nil {
            continue
        }
        
        tutors = append(tutors, gin.H{
            "id":               id,
            "name":             name,
            "bio":              bio,
            "rating":           rating,
            "total_reviews":    totalReviews,
            "discord_username": discordUsername,
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
        FROM users WHERE id = $1 AND is_suspended = 0
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
    
    // Get availability
    availability := []gin.H{}
    availRows, err := db.Query(`SELECT day_of_week, start_time, end_time FROM availability WHERE user_id = $1`, tutorID)
    if err == nil {
        defer availRows.Close()
        for availRows.Next() {
            var dayOfWeek int
            var startTime, endTime string
            availRows.Scan(&dayOfWeek, &startTime, &endTime)
            availability = append(availability, gin.H{
                "day_of_week": dayOfWeek,
                "start_time":  startTime,
                "end_time":    endTime,
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
