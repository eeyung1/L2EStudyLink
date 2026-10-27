package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"

	"L2EStudyLink/config"
	"L2EStudyLink/middleware"
)

// Public proof-of-work portfolio. A fellow opts in by claiming a unique
// public handle and flipping portfolio_public on; until then /u/:handle
// answers 404 so nothing is revealed about private accounts.

var portfolioHandlePattern = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)

var reservedPortfolioHandles = map[string]bool{
	"admin": true, "api": true, "u": true, "page": true, "static": true,
	"me": true, "login": true, "signup": true, "health": true, "uploads": true,
}

// optionalUserID trusts an already-injected test identity first, then falls
// back to the same Bearer/cookie token extraction as middleware.AuthRequired.
// An absent or invalid token is anonymous, never an error: public portfolios
// must render for signed-out visitors.
func optionalUserID(c *gin.Context) int64 {
	if id := c.GetInt64("user_id"); id > 0 {
		return id
	}
	tokenString := ""
	if parts := strings.Fields(c.GetHeader("Authorization")); len(parts) == 2 && parts[0] == "Bearer" && !strings.HasPrefix(parts[1], "cookie-session") {
		tokenString = parts[1]
	}
	if tokenString == "" {
		tokenString, _ = c.Cookie(middleware.SessionCookieName)
	}
	if tokenString == "" {
		return 0
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, http.ErrNoCookie
		}
		return config.JWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return 0
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0
	}
	value, ok := claims["user_id"].(float64)
	if !ok || value <= 0 {
		return 0
	}
	return int64(value)
}

// buildPortfolioPayload aggregates the public record for one fellow.
// ownerID is the account being viewed; viewerID is the requester (0 = anon).
// visible reports whether the viewer may see it.
func buildPortfolioPayload(db *sql.DB, ownerID, viewerID int64) (gin.H, bool, error) {
	var handle sql.NullString
	var name, bio string
	var rating float64
	var totalReviews, totalSessions int
	var portfolioPublic bool
	err := db.QueryRow(`
		SELECT COALESCE(public_handle,''), name, COALESCE(bio,''), rating, total_reviews, total_sessions, portfolio_public
		FROM users WHERE id = $1`, ownerID).
		Scan(&handle, &name, &bio, &rating, &totalReviews, &totalSessions, &portfolioPublic)
	if err != nil {
		return nil, false, err
	}
	if !portfolioPublic && ownerID != viewerID {
		return nil, false, nil // private: caller answers 404
	}

	// Verified hours come from mutually-completed bookings on either side.
	var completedSessions int
	var verifiedMinutes float64
	err = db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(EXTRACT(EPOCH FROM (end_time - start_time)) / 60), 0)
		FROM bookings WHERE (tutor_id = $1 OR student_id = $1) AND status = 'completed'`, ownerID).
		Scan(&completedSessions, &verifiedMinutes)
	if err != nil {
		return nil, false, err
	}

	rows, err := db.Query(`SELECT log_date, COUNT(*) FROM activity_logs WHERE user_id = $1 GROUP BY log_date ORDER BY log_date`, ownerID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	// date -> activity count, the shape renderHeatmap expects.
	activityDays := map[string]int{}
	var dayList []time.Time
	for rows.Next() {
		var day time.Time
		var logCount int
		if err := rows.Scan(&day, &logCount); err != nil {
			return nil, false, err
		}
		dayList = append(dayList, day)
		activityDays[day.Format("2006-01-02")] = logCount
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	currentStreak, longestStreak := portfolioStreaks(dayList, time.Now())

	skillRows, err := db.Query(`
		SELECT skill_name, proficiency FROM skills WHERE user_id = $1
		ORDER BY CASE proficiency WHEN 'advanced' THEN 0 WHEN 'intermediate' THEN 1 ELSE 2 END, skill_name`, ownerID)
	if err != nil {
		return nil, false, err
	}
	defer skillRows.Close()
	skills := make([]gin.H, 0)
	for skillRows.Next() {
		var skillName, proficiency string
		if err := skillRows.Scan(&skillName, &proficiency); err != nil {
			return nil, false, err
		}
		skills = append(skills, gin.H{"name": skillName, "proficiency": proficiency})
	}
	if err := skillRows.Err(); err != nil {
		return nil, false, err
	}

	reviewRows, err := db.Query(`
		SELECT u.name, r.rating, r.comment, COALESCE(r.would_book_again, FALSE), r.created_at
		FROM reviews r JOIN users u ON u.id = r.reviewer_id
		WHERE r.reviewee_id = $1 ORDER BY r.created_at DESC LIMIT 10`, ownerID)
	if err != nil {
		return nil, false, err
	}
	defer reviewRows.Close()
	reviews := make([]gin.H, 0)
	for reviewRows.Next() {
		var reviewerName, comment string
		var reviewRating int
		var wouldBookAgain bool
		var createdAt time.Time
		if err := reviewRows.Scan(&reviewerName, &reviewRating, &comment, &wouldBookAgain, &createdAt); err != nil {
			return nil, false, err
		}
		reviews = append(reviews, gin.H{
			"reviewer": reviewerName, "rating": reviewRating, "comment": comment,
			"would_book_again": wouldBookAgain, "date": createdAt.Format("2006-01-02"),
		})
	}
	if err := reviewRows.Err(); err != nil {
		return nil, false, err
	}

	projectRows, err := db.Query(`
		SELECT p.title, p.description, p.roles_needed, p.repo_url
		FROM projects p JOIN project_members m ON m.project_id = p.id
		WHERE m.user_id = $1 AND p.status = 'closed' ORDER BY p.id DESC LIMIT 12`, ownerID)
	if err != nil {
		return nil, false, err
	}
	defer projectRows.Close()
	projects := make([]gin.H, 0)
	for projectRows.Next() {
		var title, description, rolesNeeded, repoURL string
		if err := projectRows.Scan(&title, &description, &rolesNeeded, &repoURL); err != nil {
			return nil, false, err
		}
		projects = append(projects, gin.H{
			"title": title, "description": description, "roles_needed": rolesNeeded, "repo_url": repoURL,
		})
	}
	if err := projectRows.Err(); err != nil {
		return nil, false, err
	}

	return gin.H{
		"handle":             handle.String,
		"name":               name,
		"bio":                bio,
		"rating":             rating,
		"total_reviews":      totalReviews,
		"total_sessions":     totalSessions,
		"portfolio_public":   portfolioPublic,
		"is_owner":           ownerID == viewerID,
		"verified_hours":     float64(int(verifiedMinutes/6+0.5)) / 10, // one decimal place
		"completed_sessions": completedSessions,
		"current_streak":     currentStreak,
		"longest_streak":     longestStreak,
		"activity_days":      activityDays,
		"skills":             skills,
		"reviews":            reviews,
		"projects":           projects,
	}, true, nil
}

// portfolioStreaks walks distinct activity days (ascending) and returns the
// current consecutive run ending today or yesterday, and the longest run seen.
func portfolioStreaks(days []time.Time, today time.Time) (int, int) {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	longest, run := 0, 0
	var prev time.Time
	for _, day := range days {
		day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
		if !prev.IsZero() && day.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > longest {
			longest = run
		}
		prev = day
	}
	current := 0
	if len(days) > 0 {
		last := time.Date(days[len(days)-1].Year(), days[len(days)-1].Month(), days[len(days)-1].Day(), 0, 0, 0, 0, time.UTC)
		if today.Sub(last) <= 24*time.Hour {
			current = 1
			for i := len(days) - 1; i > 0; i-- {
				day := time.Date(days[i].Year(), days[i].Month(), days[i].Day(), 0, 0, 0, 0, time.UTC)
				previous := time.Date(days[i-1].Year(), days[i-1].Month(), days[i-1].Day(), 0, 0, 0, 0, time.UTC)
				if day.Sub(previous) != 24*time.Hour {
					break
				}
				current++
			}
		}
	}
	return current, longest
}

// GetPublicPortfolio serves GET /api/v1/portfolio/:handle without auth.
func GetPublicPortfolio(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	handle := strings.ToLower(strings.TrimSpace(c.Param("handle")))
	if !portfolioHandlePattern.MatchString(handle) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Portfolio not found"})
		return
	}
	var ownerID int64
	err := db.QueryRow(`SELECT id FROM users WHERE lower(public_handle) = $1`, handle).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Portfolio not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load portfolio"})
		return
	}
	payload, visible, err := buildPortfolioPayload(db, ownerID, optionalUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load portfolio"})
		return
	}
	if !visible {
		c.JSON(http.StatusNotFound, gin.H{"error": "Portfolio not found"})
		return
	}
	c.JSON(http.StatusOK, payload)
}

// GetMyPortfolio serves the authenticated owner preview at
// GET /api/v1/portfolio/me, available before a handle is claimed.
func GetMyPortfolio(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	ownerID := c.GetInt64("user_id")
	if ownerID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in required"})
		return
	}
	payload, _, err := buildPortfolioPayload(db, ownerID, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load portfolio"})
		return
	}
	c.JSON(http.StatusOK, payload)
}

// ClaimPortfolioHandle serves PUT /api/v1/portfolio/handle {handle}.
func ClaimPortfolioHandle(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	ownerID := c.GetInt64("user_id")
	if ownerID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in required"})
		return
	}
	var input struct {
		Handle string `json:"handle"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	handle := strings.ToLower(strings.TrimSpace(input.Handle))
	if !portfolioHandlePattern.MatchString(handle) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use 3-30 lowercase letters, numbers or underscores."})
		return
	}
	if reservedPortfolioHandles[handle] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "That handle is reserved. Pick another."})
		return
	}
	var claimed string
	err := db.QueryRow(`UPDATE users SET public_handle = $1 WHERE id = $2 RETURNING public_handle`, handle, ownerID).Scan(&claimed)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "That handle is already taken."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save your handle"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"handle": claimed, "url": "/u/" + claimed})
}

// SetPortfolioVisibility serves PUT /api/v1/portfolio/visibility {public}.
func SetPortfolioVisibility(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	ownerID := c.GetInt64("user_id")
	if ownerID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in required"})
		return
	}
	var input struct {
		Public bool `json:"public"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if input.Public {
		var hasHandle bool
		if err := db.QueryRow(`SELECT public_handle IS NOT NULL FROM users WHERE id = $1`, ownerID).Scan(&hasHandle); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update visibility"})
			return
		}
		if !hasHandle {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Claim a public handle before going public."})
			return
		}
	}
	var isPublic bool
	err := db.QueryRow(`UPDATE users SET portfolio_public = $1 WHERE id = $2 RETURNING portfolio_public`, input.Public, ownerID).Scan(&isPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update visibility"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"portfolio_public": isPublic})
}
