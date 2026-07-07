package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"L2EStudyLink/email"
)

// generateResetToken returns a random 32-byte hex string (64 characters),
// unguessable and safe to embed directly in a URL.
func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ForgotPassword looks up the user by email, generates a reset token valid
// for 1 hour, stores it, and emails a reset link. Always returns the same
// success response whether or not the email exists, so this endpoint can't
// be used to check which emails are registered.
func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid email required"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	const genericResponse = "If that email is registered, a reset link has been sent."

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", input.Email).Scan(&userID)

	if err == sql.ErrNoRows {
		// Don't reveal whether the email exists.
		c.JSON(http.StatusOK, gin.H{"message": genericResponse})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}

	token, err := generateResetToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}

	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = db.Exec(
		"INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
		userID, token, expiresAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	resetLink := fmt.Sprintf("%s/page/reset-password?token=%s", baseURL, token)

	htmlBody := fmt.Sprintf(`
		<p>You requested a password reset for L2EStudyLink.</p>
		<p><a href="%s">Click here to reset your password</a></p>
		<p>This link expires in 1 hour. If you didn't request this, you can ignore this email.</p>
	`, resetLink)

// Email failures shouldn't reveal internal state to the client, but we
	// still want a real error if sending genuinely fails.
	if err := email.SendEmail(input.Email, "Reset your L2EStudyLink password", htmlBody); err != nil {
		log.Println("ForgotPassword: failed to send email:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": genericResponse})
}

// ResetPassword validates the token, checks it hasn't expired or been used,
// then updates the user's password and marks the token used.
func ResetPassword(c *gin.Context) {
	var input struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and a password of at least 6 characters are required"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	var userID int
	var expiresAt time.Time
	var used bool

	err := db.QueryRow(
		"SELECT user_id, expires_at, used FROM password_reset_tokens WHERE token = $1",
		input.Token,
	).Scan(&userID, &expiresAt, &used)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset link"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}

	if used || time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset link"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}

	_, err = db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", string(hashedPassword), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}

	_, err = db.Exec("UPDATE password_reset_tokens SET used = TRUE WHERE token = $1", input.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully. You can now log in."})
}