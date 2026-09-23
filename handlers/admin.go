package handlers

import (
    "database/sql"
    "net/http"

    "github.com/gin-gonic/gin"
)

func AdminStats(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    
    var totalUsers, totalBookings, totalSkills int
    
    // Count ALL users (no filter)
    db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
    
    // Count ALL bookings
    db.QueryRow("SELECT COUNT(*) FROM bookings").Scan(&totalBookings)
    
    // Count ALL skills
    db.QueryRow("SELECT COUNT(*) FROM skills").Scan(&totalSkills)
    
    c.JSON(http.StatusOK, gin.H{
        "total_users": totalUsers,
        "total_bookings": totalBookings,
        "total_skills": totalSkills,
    })
}

func AdminUsers(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    
    rows, err := db.Query(`
        SELECT id, name, email, rating, total_sessions, no_show_count, is_suspended, is_admin 
        FROM users ORDER BY id
    `)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
        return
    }
    defer rows.Close()
    
    var users []gin.H
    for rows.Next() {
        var id int64
        var name, email string
        var rating float64
        var totalSessions, noShowCount int
        var isSuspended, isAdmin bool
        
        if err := rows.Scan(&id, &name, &email, &rating, &totalSessions, &noShowCount, &isSuspended, &isAdmin); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read users"})
            return
        }
        
        users = append(users, gin.H{
            "id": id,
            "name": name,
            "email": email,
            "rating": rating,
            "total_sessions": totalSessions,
            "no_show_count": noShowCount,
            "is_suspended": isSuspended,
            "is_admin": isAdmin,
        })
    }
    
    c.JSON(http.StatusOK, users)
}

func AdminDeleteUser(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    userID := c.Param("id")
    
    _, err := db.Exec("DELETE FROM users WHERE id = $1", userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func AdminToggleSuspend(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    userID := c.Param("id")
    
    var currentStatus bool
    if err := db.QueryRow("SELECT is_suspended FROM users WHERE id = $1", userID).Scan(&currentStatus); err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
        }
        return
    }
    
    newStatus := !currentStatus
    _, err := db.Exec("UPDATE users SET is_suspended = $1 WHERE id = $2", newStatus, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "User suspension toggled"})
}

func AdminBookings(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    
    rows, err := db.Query(`
        SELECT b.id, t.name as tutor_name, s.name as student_name, 
               b.session_date, b.start_time, b.status
        FROM bookings b
        JOIN users t ON b.tutor_id = t.id
        JOIN users s ON b.student_id = s.id
        ORDER BY b.id DESC
    `)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
        return
    }
    defer rows.Close()
    
    var bookings []gin.H
    for rows.Next() {
        var id int64
        var tutorName, studentName, sessionDate, startTime, status string
        
        rows.Scan(&id, &tutorName, &studentName, &sessionDate, &startTime, &status)
        
        bookings = append(bookings, gin.H{
            "id": id,
            "tutor": tutorName,
            "student": studentName,
            "date": sessionDate,
            "time": startTime,
            "status": status,
        })
    }
    
    c.JSON(http.StatusOK, bookings)
}
