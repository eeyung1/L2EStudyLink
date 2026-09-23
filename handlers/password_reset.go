package handlers

import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "crypto/subtle"
    "database/sql"
    "encoding/hex"
    "fmt"
    "log"
    "math/big"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"

    "L2EStudyLink/config"
    "L2EStudyLink/email"
)

const resetLifetime = 10 * time.Minute
const resetCooldown = time.Minute
const maxResetAttempts = 5
const resetReply = "If that email is registered, a verification code has been sent. Check your inbox."

func resetCode() (string, error) {
    n, err := rand.Int(rand.Reader, big.NewInt(1000000))
    if err != nil { return "", err }
    return fmt.Sprintf("%06d", n.Int64()), nil
}

// The database stores an HMAC, never the short code itself. The attempt count
// is encoded in the existing token column, so existing installations need no
// schema migration. JWT_SECRET is required at application startup.
func resetCodeHash(userID int64, nonce, code string) string {
    mac := hmac.New(sha256.New, config.JWTSecret())
    fmt.Fprintf(mac, "password-reset-otp:%d:%s:%s", userID, nonce, code)
    return fmt.Sprintf("%x", mac.Sum(nil))
}

func ForgotPassword(c *gin.Context) {
    var input struct { Email string `json:"email" binding:"required,email"` }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Valid email required"})
        return
    }
    // This check applies equally to known and unknown addresses.
    if os.Getenv("BREVO_API_KEY") == "" || os.Getenv("BREVO_FROM_EMAIL") == "" {
        log.Print("ForgotPassword: BREVO_API_KEY or BREVO_FROM_EMAIL is missing")
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Password recovery is temporarily unavailable"})
        return
    }
    address := strings.ToLower(strings.TrimSpace(input.Email))
    db := c.MustGet("db").(*sql.DB)
    var userID int64
    if err := db.QueryRowContext(c.Request.Context(), "SELECT id FROM users WHERE LOWER(email) = $1", address).Scan(&userID); err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusOK, gin.H{"message": resetReply})
            return
        }
        log.Printf("ForgotPassword lookup: %v", err)
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Password recovery is temporarily unavailable"})
        return
    }

    var lastSent time.Time
    err := db.QueryRowContext(c.Request.Context(),
        "SELECT created_at FROM password_reset_tokens WHERE user_id = $1 AND token LIKE 'otp:%' ORDER BY created_at DESC, id DESC LIMIT 1", userID,
    ).Scan(&lastSent)
    if err != nil && err != sql.ErrNoRows {
        log.Printf("ForgotPassword cooldown: %v", err)
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Password recovery is temporarily unavailable"})
        return
    }
    if err == nil && time.Since(lastSent) < resetCooldown {
        c.JSON(http.StatusOK, gin.H{"message": resetReply})
        return
    }

    code, err := resetCode()
    if err != nil {
        log.Printf("ForgotPassword code generation: %v", err)
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Password recovery is temporarily unavailable"})
        return
    }
    randomNonce := make([]byte, 16)
    if _, err := rand.Read(randomNonce); err != nil {
        log.Printf("ForgotPassword nonce generation: %v", err)
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Password recovery is temporarily unavailable"})
        return
    }
    nonce := hex.EncodeToString(randomNonce)
    token := "otp:0:" + nonce + ":" + resetCodeHash(userID, nonce, code)
    _, err = db.ExecContext(c.Request.Context(),
        "INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
        userID, token, time.Now().Add(resetLifetime),
    )
    if err != nil {
        log.Printf("ForgotPassword insert: %v", err)
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Password recovery is temporarily unavailable"})
        return
    }
    body := fmt.Sprintf(`<p>Your L2EStudyLink password reset code is <strong>%s</strong>.</p>
<p>It expires in 10 minutes. If you did not request it, ignore this email.</p>`, code)
    if err := email.SendEmail(address, "Your L2EStudyLink password reset code", body); err != nil {
        // Do not reveal whether this email address has an account.
        log.Printf("ForgotPassword delivery: %v", err)
        if _, revokeErr := db.ExecContext(c.Request.Context(), "DELETE FROM password_reset_tokens WHERE token = $1", token); revokeErr != nil {
            log.Printf("ForgotPassword revoke: %v", revokeErr)
        }
    }
    c.JSON(http.StatusOK, gin.H{"message": resetReply})
}

func ResetPassword(c *gin.Context) {
    var input struct {
        Email string `json:"email" binding:"required,email"`
        Code string `json:"code" binding:"required,len=6,numeric"`
        NewPassword string `json:"new_password" binding:"required,min=8"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Enter your email, six-digit code, and a password of at least 8 characters"})
        return
    }
    db := c.MustGet("db").(*sql.DB)
    invalid := gin.H{"error": "Invalid or expired code. Request a new code if needed."}
    var userID int64
    var name string
    address := strings.ToLower(strings.TrimSpace(input.Email))
    err := db.QueryRowContext(c.Request.Context(), "SELECT id, name FROM users WHERE LOWER(email) = $1", address).Scan(&userID, &name)
    if err == sql.ErrNoRows { c.JSON(http.StatusBadRequest, invalid); return }
    if err != nil {
        log.Printf("ResetPassword lookup: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"})
        return
    }

    tx, err := db.BeginTx(c.Request.Context(), nil)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"}); return }
    defer tx.Rollback()
    var token string
    var expiresAt time.Time
    err = tx.QueryRowContext(c.Request.Context(), `
        SELECT token, expires_at FROM password_reset_tokens
        WHERE user_id = $1 AND used = FALSE AND token LIKE 'otp:%'
        ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE
    `, userID).Scan(&token, &expiresAt)
    if err == sql.ErrNoRows { c.JSON(http.StatusBadRequest, invalid); return }
    if err != nil {
        log.Printf("ResetPassword token lookup: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"})
        return
    }
    parts := strings.Split(token, ":")
    if len(parts) != 4 || parts[0] != "otp" || time.Now().After(expiresAt) {
        c.JSON(http.StatusBadRequest, invalid)
        return
    }
    attempts, err := strconv.Atoi(parts[1])
    if err != nil || attempts >= maxResetAttempts {
        c.JSON(http.StatusBadRequest, invalid)
        return
    }
    expected := resetCodeHash(userID, parts[2], input.Code)
    if subtle.ConstantTimeCompare([]byte(parts[3]), []byte(expected)) != 1 {
        attempts++
        nextToken := fmt.Sprintf("otp:%d:%s:%s", attempts, parts[2], parts[3])
        _, err = tx.ExecContext(c.Request.Context(),
            "UPDATE password_reset_tokens SET token = $1, used = $2 WHERE token = $3",
            nextToken, attempts >= maxResetAttempts, token,
        )
        if err != nil {
            log.Printf("ResetPassword attempt update: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"})
            return
        }
        if err := tx.Commit(); err != nil {
            log.Printf("ResetPassword attempt commit: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"})
            return
        }
        c.JSON(http.StatusBadRequest, invalid)
        return
    }
    hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"}); return }
    if _, err = tx.ExecContext(c.Request.Context(), "UPDATE users SET password_hash = $1 WHERE id = $2", string(hashed), userID); err != nil {
        log.Printf("ResetPassword update: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"})
        return
    }
    if _, err = tx.ExecContext(c.Request.Context(), "UPDATE password_reset_tokens SET used = TRUE WHERE user_id = $1 AND used = FALSE", userID); err != nil {
        log.Printf("ResetPassword consume: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"})
        return
    }
    if err = tx.Commit(); err != nil {
        log.Printf("ResetPassword commit: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to reset password"})
        return
    }

    session := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID,
        "exp": time.Now().Add(24 * time.Hour).Unix(),
    })
    signed, err := session.SignedString(config.JWTSecret())
    if err != nil {
        log.Printf("ResetPassword sign in: %v", err)
        c.JSON(http.StatusOK, gin.H{"message": "Password updated. Please sign in."})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Password updated", "token": signed, "user": gin.H{"id": userID, "name": name}})
}
