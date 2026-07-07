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

    // Static HTML pages
    router.LoadHTMLGlob("templates/*.html")
    
    router.GET("/", func(c *gin.Context) {
        c.Redirect(302, "/page/login")
    })
    
    router.GET("/dashboard", func(c *gin.Context) {
        c.HTML(200, "dashboard.html", nil)
    })
    
    router.GET("/search-page", func(c *gin.Context) {
        c.HTML(200, "search-page.html", nil)
    })
    
    router.GET("/my-bookings", func(c *gin.Context) {
        c.HTML(200, "my-bookings.html", nil)
    })
    
    router.GET("/admin", func(c *gin.Context) {
        c.HTML(200, "admin.html", nil)
    })

    router.GET("/page/forgot-password", func(c *gin.Context) {
    c.HTML(200, "forgot-password.html", nil)
    })
    
    router.GET("/page/reset-password", func(c *gin.Context) {
        c.HTML(200, "reset-password.html", nil)
    })
        
    router.GET("/timetable", func(c *gin.Context) {
        c.HTML(200, "timetable.html", nil)
    })
    
    router.GET("/reflections", func(c *gin.Context) {
        c.HTML(200, "reflections.html", nil)
    })
    
    router.GET("/page/:name", func(c *gin.Context) {
        name := c.Param("name")
        c.HTML(200, name+".html", nil)
    })

    // API routes
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
        api.POST("/forgot-password", handlers.ForgotPassword)
        api.POST("/reset-password", handlers.ResetPassword)
        api.GET("/ping", func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "pong"})
        })
        api.GET("/search", handlers.SearchTutors)
        api.GET("/search-with-availability", handlers.SearchTutorsWithAvailability)
        api.GET("/tutors/:id", handlers.GetTutorProfile)

        protected := api.Group("/")
        protected.Use(middleware.AuthRequired)
        {
            protected.GET("/me", handlers.GetProfile)
            protected.PUT("/profile", handlers.UpdateProfile)
            protected.POST("/skills", handlers.AddSkill)
            protected.DELETE("/skills/:skill", handlers.RemoveSkill)
            protected.GET("/availability", handlers.GetAvailability)
            protected.PUT("/availability", handlers.SetAvailability)
            protected.POST("/bookings", handlers.CreateBooking)
            protected.GET("/bookings", handlers.GetMyBookings)
            protected.DELETE("/bookings/:id", handlers.CancelBooking)
            protected.PUT("/bookings/:id/status", handlers.UpdateBookingStatus)
            protected.GET("/admin/stats", handlers.AdminStats)
            protected.GET("/admin/users", handlers.AdminUsers)
            protected.DELETE("/admin/users/:id", handlers.AdminDeleteUser)
            protected.PUT("/admin/users/:id/suspend", handlers.AdminToggleSuspend)
            protected.GET("/admin/bookings", handlers.AdminBookings)
            protected.POST("/admin/reset-password", handlers.ResetAdminPassword)
            
            // Timetable routes
            protected.GET("/timetable", handlers.GetTimetable)
            protected.POST("/timetable", handlers.AddTimeBlock)
            protected.DELETE("/timetable/:id", handlers.DeleteTimeBlock)
            protected.POST("/reflections", handlers.AddReflection)
            protected.GET("/reflections", handlers.GetReflections)
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