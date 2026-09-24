package handlers

import (
    "crypto/hmac"
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "errors"
    "net/http"
    "strings"
    "time"

    "github.com/jackc/pgx/v5/pgconn"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    "L2EStudyLink/config"
)

func Signup(c *gin.Context) {
    var input struct {
        Name     string `json:"name" binding:"required"`
        Email    string `json:"email" binding:"required,email"`
        Password string `json:"password" binding:"required,min=6"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), 10)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
        return
    }

    db := c.MustGet("db").(*sql.DB)
    
    var userID int64
    err = db.QueryRow(
        "INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
        input.Name, input.Email, string(hashedPassword),
    ).Scan(&userID)

    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account. Please try again."})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "User created successfully",
        "user_id": userID,
    })
}

func Login(c *gin.Context) {
    var input struct {
        Email    string `json:"email" binding:"required"`
        Password string `json:"password" binding:"required"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    db := c.MustGet("db").(*sql.DB)
    email := strings.ToLower(strings.TrimSpace(input.Email))
    mac := hmac.New(sha256.New, config.JWTSecret())
    mac.Write([]byte(email))
    key := hex.EncodeToString(mac.Sum(nil))
    var locked bool
    err := db.QueryRowContext(c.Request.Context(),`SELECT COALESCE(locked_until > NOW(),FALSE) FROM login_attempts WHERE subject_key=$1`,key).Scan(&locked)
    if err!=nil && !errors.Is(err,sql.ErrNoRows) {c.JSON(500,gin.H{"error":"Something went wrong. Please try again."});return}
    if locked {c.Header("Retry-After","900");c.JSON(http.StatusTooManyRequests,gin.H{"error":"Too many attempts. Try again in 15 minutes."});return}
    var user struct {
        ID           int64
        Name         string
        PasswordHash string
    }

    err = db.QueryRowContext(c.Request.Context(),
        "SELECT id, name, password_hash FROM users WHERE LOWER(email) = $1",
        email,
    ).Scan(&user.ID, &user.Name, &user.PasswordHash)

    if err == sql.ErrNoRows {
        if err:=recordFailedLogin(c,db,key);err!=nil {c.JSON(500,gin.H{"error":"Something went wrong. Please try again."});return}
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
        if err:=recordFailedLogin(c,db,key);err!=nil {c.JSON(500,gin.H{"error":"Something went wrong. Please try again."});return}
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
        return
    }

    if _,err=db.ExecContext(c.Request.Context(),`DELETE FROM login_attempts WHERE subject_key=$1`,key);err!=nil {c.JSON(500,gin.H{"error":"Something went wrong. Please try again."});return}

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": user.ID,
        "exp":     time.Now().Add(time.Hour * 24).Unix(),
    })

    tokenString, err := token.SignedString(config.JWTSecret())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "token": tokenString,
        "user": gin.H{
            "id":   user.ID,
            "name": user.Name,
        },
    })
}

func recordFailedLogin(c *gin.Context, db *sql.DB, key string) error {
    _,err:=db.ExecContext(c.Request.Context(),`INSERT INTO login_attempts(subject_key,attempts,window_started_at,locked_until) VALUES($1,1,NOW(),NULL)
        ON CONFLICT(subject_key) DO UPDATE SET
        attempts=CASE WHEN login_attempts.window_started_at < NOW()-INTERVAL '15 minutes' THEN 1 ELSE login_attempts.attempts+1 END,
        window_started_at=CASE WHEN login_attempts.window_started_at < NOW()-INTERVAL '15 minutes' THEN NOW() ELSE login_attempts.window_started_at END,
        locked_until=CASE WHEN login_attempts.window_started_at < NOW()-INTERVAL '15 minutes' THEN NULL WHEN login_attempts.attempts+1>=5 THEN NOW()+INTERVAL '15 minutes' ELSE login_attempts.locked_until END`,key)
    return err
}
