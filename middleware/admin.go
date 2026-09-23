package middleware

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminRequired checks the user's current role in the database, so a
// suspended or demoted account cannot retain admin access via an old JWT.
func AdminRequired(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	var isAdmin, isSuspended bool
	err := db.QueryRowContext(c.Request.Context(),
		"SELECT is_admin, is_suspended FROM users WHERE id = $1", c.GetInt64("user_id"),
	).Scan(&isAdmin, &isSuspended)
	if err == sql.ErrNoRows || err == nil && (!isAdmin || isSuspended) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin access"})
		return
	}
	c.Next()
}
