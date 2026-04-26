package handlers

import (
    "database/sql"
    "net/http"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"
)

func ResetAdminPassword(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    
    // Hash new password "admin123"
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
    
    result, err := db.Exec(`
        UPDATE users SET password_hash = $1 WHERE email = 'eyungemmanuel@gmail.com'
    `, string(hashedPassword))
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    c.JSON(http.StatusOK, gin.H{
        "message": "Password reset for eyungemmanuel@gmail.com",
        "rows_affected": rowsAffected,
        "new_password": "admin123",
    })
}

func ResetAdminPassword(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    
    // Hash password "admin123"
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
    
    result, err := db.Exec(`
        UPDATE users SET password_hash = $1 WHERE email = 'eyungemmanuel@gmail.com'
    `, string(hashedPassword))
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    c.JSON(http.StatusOK, gin.H{
        "message": "Password reset for eyungemmanuel@gmail.com",
        "rows_affected": rowsAffected,
        "new_password": "admin123",
    })
}
