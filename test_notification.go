package main

import (
    "log"
    "os"
    "L2EStudyLink/notifications"
    "github.com/joho/godotenv"
)

func main() {
    godotenv.Load()
    
    webhook := os.Getenv("DISCORD_WEBHOOK_URL")
    notifications.Init(webhook)
    
    log.Println("Testing Discord notification...")
    notifications.SendNotification("🎉 StudyLink Discord notifications are WORKING! 🎉")
    log.Println("Check your Discord #notifications channel!")
}
