package main

import (
    "log"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"

    "L2EStudyLink/db"
    "L2EStudyLink/handlers"
    "L2EStudyLink/middleware"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Println("Warning: .env file not found, using system environment")
    }

    if err := db.InitDB(); err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.CloseDB()

    router := gin.Default()

    router.Use(func(c *gin.Context) {
        c.Set("db", db.DB)
        c.Next()
    })

    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "message": "L2EStudyLink is running",
        })
    })

    api := router.Group("/api/v1")
    {
        api.POST("/signup", handlers.Signup)
        api.POST("/login", handlers.Login)
        api.GET("/ping", func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "pong"})
        })

        protected := api.Group("/")
        protected.Use(middleware.AuthRequired)
        {
            protected.GET("/me", func(c *gin.Context) {
                userID := c.GetInt64("user_id")
                c.JSON(200, gin.H{
                    "message": "You are authenticated!",
                    "user_id": userID,
                })
            })
        }
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server starting on :%s", port)
    if err := router.Run(":" + port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}
