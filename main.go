package main

import (
    "log"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"

    "L2EStudyLink/db"
)

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("Warning: .env file not found, using system environment")
    }

    // Initialize database
    if err := db.InitDB(); err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.CloseDB()

    // Create Gin router
    router := gin.Default()

    // Health check endpoint
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "message": "L2EStudyLink is running",
        })
    })

    // API routes
    api := router.Group("/api/v1")
    {
        api.GET("/ping", func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "pong"})
        })
    }

    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server starting on :%s", port)
    if err := router.Run(":" + port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}