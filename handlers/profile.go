package handlers

import (
    "database/sql"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var user struct {
        ID              int64
        Name            string
        Email           string
        DiscordUsername sql.NullString
        Bio             sql.NullString
        Rating          float64
        TotalReviews    int
        TotalSessions   int
        IsAdmin         bool
        ProductEmails bool
    }

    err := db.QueryRow(`
        SELECT id, name, email, discord_username, bio, rating, total_reviews, total_sessions, is_admin, marketing_opt_in_at IS NOT NULL
        FROM users WHERE id = $1
    `, userID).Scan(
        &user.ID, &user.Name, &user.Email, &user.DiscordUsername,
        &user.Bio, &user.Rating, &user.TotalReviews, &user.TotalSessions, &user.IsAdmin, &user.ProductEmails,
    )

    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        return
    }

    rows, err := db.Query(`
        SELECT skill_name, proficiency FROM skills WHERE user_id = $1
    `, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch skills"})
        return
    }
    defer rows.Close()

    var skills []map[string]string
    for rows.Next() {
        var name, proficiency string
        rows.Scan(&name, &proficiency)
        skills = append(skills, map[string]string{
            "name":        name,
            "proficiency": proficiency,
        })
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
        "is_admin":         user.IsAdmin,
        "product_emails":   user.ProductEmails,
    })
}

func SetProductEmails(c *gin.Context) {
    var input struct { Enabled *bool `json:"enabled" binding:"required"` }
    if err := c.ShouldBindJSON(&input); err != nil || input.Enabled == nil {
        c.JSON(http.StatusBadRequest,gin.H{"error":"Choose whether to receive product updates"});return
    }
    db:=c.MustGet("db").(*sql.DB)
    if _,err:=db.ExecContext(c.Request.Context(),`UPDATE users SET marketing_opt_in_at=CASE WHEN $1 THEN NOW() ELSE NULL END WHERE id=$2`,*input.Enabled,c.GetInt64("user_id"));err!=nil {
        c.JSON(http.StatusInternalServerError,gin.H{"error":"Could not save email preference"});return
    }
    c.JSON(http.StatusOK,gin.H{"enabled":*input.Enabled})
}

func UpdateProfile(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var input struct {
        DiscordUsername string `json:"discord_username"`
        Bio             string `json:"bio"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    _, err := db.Exec(`
        UPDATE users SET discord_username = $1, bio = $2 WHERE id = $3
    `, input.DiscordUsername, input.Bio, userID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func AddSkill(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var input struct {
        SkillName   string `json:"skill_name" binding:"required"`
        Proficiency string `json:"proficiency" binding:"required,oneof=beginner intermediate advanced"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    _, err := db.Exec(`
        INSERT INTO skills (user_id, skill_name, proficiency) VALUES ($1, $2, $3)
    `, userID, input.SkillName, input.Proficiency)

    if err != nil {
        c.JSON(http.StatusConflict, gin.H{"error": "Skill already exists or invalid"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Skill added successfully"})
}

func RemoveSkill(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    skillName := c.Param("skill")

    result, err := db.Exec(`
        DELETE FROM skills WHERE user_id = $1 AND skill_name = $2
    `, userID, skillName)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove skill"})
        return
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Skill not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Skill removed successfully"})
}

// EditSkill updates only the signed-in user's existing skill.
func EditSkill(c *gin.Context) {
    var input struct {
        SkillName string `json:"skill_name" binding:"required"`
        Proficiency string `json:"proficiency" binding:"required,oneof=beginner intermediate advanced"`
    }
    if err := c.ShouldBindJSON(&input); err != nil { c.JSON(400, gin.H{"error":"Enter a skill and a valid proficiency"}); return }
    name := strings.TrimSpace(input.SkillName)
    if len(name) < 1 || len(name) > 50 { c.JSON(400,gin.H{"error":"Skill name must be 1–50 characters"}); return }
    result,err := c.MustGet("db").(*sql.DB).ExecContext(c.Request.Context(),`UPDATE skills SET skill_name=$1,proficiency=$2 WHERE user_id=$3 AND skill_name=$4`,name,input.Proficiency,c.GetInt64("user_id"),c.Param("skill"))
    if err != nil { c.JSON(409,gin.H{"error":"This skill already exists or could not be updated"}); return }
    count,_:=result.RowsAffected(); if count==0 { c.JSON(404,gin.H{"error":"Skill not found"}); return }
    c.JSON(200,gin.H{"message":"Skill updated"})
}
