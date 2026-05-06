package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func SearchTutors(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)

	skill := c.Query("skill")
	date := c.Query("date")

	if skill == "" {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	// optional date filter
	dayOfWeek := -1
	if date != "" {
		t, err := time.Parse("2006-01-02", date)
		if err == nil {
			day := int(t.Weekday())
			if day == 0 {
				dayOfWeek = 6
			} else {
				dayOfWeek = day - 1
			}
		}
	}

	query := `
	SELECT DISTINCT
		u.id,
		u.name,
		COALESCE(u.bio, ''),
		u.rating,
		u.total_reviews,
		COALESCE(u.discord_username, '')
	FROM users u
	JOIN skills s ON u.id = s.user_id
	`

	args := []any{"%" + skill + "%"}

	if dayOfWeek >= 0 {
		query += `
		JOIN availability a ON a.user_id = u.id
		WHERE s.skill_name ILIKE $1
		AND a.day_of_week = $2
		`
		args = append(args, dayOfWeek)
	} else {
		query += `WHERE s.skill_name ILIKE $1`
	}

	query += `
	ORDER BY u.rating DESC
	LIMIT 50
	`

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	tutors := make([]gin.H, 0, 50)

	for rows.Next() {
		var id int64
		var name, bio, discord string
		var rating float64
		var reviews int

		if err := rows.Scan(&id, &name, &bio, &rating, &reviews, &discord); err != nil {
			continue
		}

		tutors = append(tutors, gin.H{
			"id":            id,
			"name":          name,
			"bio":           bio,
			"rating":        rating,
			"total_reviews": reviews,
			"discord":       discord,
		})
	}

	c.JSON(http.StatusOK, tutors)
}

func GetTutorProfile(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	id := c.Param("id")

	var user struct {
		ID       int64
		Name     string
		Email    string
		Discord  sql.NullString
		Bio      sql.NullString
		Rating   float64
		Reviews  int
		Sessions int
	}

	err := db.QueryRow(`
		SELECT id, name, email, discord_username, bio, rating, total_reviews, total_sessions
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Discord,
		&user.Bio,
		&user.Rating,
		&user.Reviews,
		&user.Sessions,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// SKILLS
	skills := []gin.H{}
	rows, _ := db.Query(`
		SELECT skill_name, proficiency
		FROM skills WHERE user_id = $1
	`, id)
	defer rows.Close()

	for rows.Next() {
		var s, p string
		rows.Scan(&s, &p)
		skills = append(skills, gin.H{"name": s, "proficiency": p})
	}

	// AVAILABILITY (NO STRING PARSING!)
	availability := []gin.H{}
	arows, _ := db.Query(`
		SELECT day_of_week, start_time, end_time
		FROM availability WHERE user_id = $1
	`, id)
	defer arows.Close()

	for arows.Next() {
		var d int
		var start, end string
		arows.Scan(&d, &start, &end)

		availability = append(availability, gin.H{
			"day":   d,
			"start": start[:5],
			"end":   end[:5],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"name":         user.Name,
		"email":        user.Email,
		"discord":      user.Discord.String,
		"bio":          user.Bio.String,
		"rating":       user.Rating,
		"reviews":      user.Reviews,
		"sessions":     user.Sessions,
		"skills":       skills,
		"availability": availability,
	})
}
