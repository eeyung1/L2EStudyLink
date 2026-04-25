package notifications

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

var webhookURL string

func Init(webhook string) {
    webhookURL = webhook
}

type DiscordWebhook struct {
    Content   string `json:"content"`
    Username  string `json:"username"`
}

func SendNotification(message string) {
    if webhookURL == "" {
        fmt.Println("No webhook URL set - notification not sent")
        return
    }
    
    webhook := DiscordWebhook{
        Content:  message,
        Username: "StudyLink Bot",
    }
    
    jsonData, err := json.Marshal(webhook)
    if err != nil {
        fmt.Println("Failed to marshal JSON:", err)
        return
    }
    
    resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Println("Failed to send webhook:", err)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 204 {
        fmt.Println("✅ Notification sent successfully!")
    } else {
        fmt.Println("Webhook response:", resp.Status)
    }
}

// Send notification with @mentions using Discord User IDs
func SendBookingNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, date, time, topic string) {
    message := ""
    
    // Add @mentions if Discord IDs are provided
    if tutorDiscord != "" {
        message += fmt.Sprintf("<@%s> ", tutorDiscord)
    }
    if studentDiscord != "" {
        message += fmt.Sprintf("<@%s> ", studentDiscord)
    }
    
    message += fmt.Sprintf("\n📚 **New Booking!**\n\n**Tutor:** %s\n**Student:** %s\n**When:** %s at %s\n**Topic:** %s\n\nCheck your dashboard to accept or decline!",
        tutorName, studentName, date, time, topic)
    
    SendNotification(message)
}

func SendBookingAcceptedNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, date, time string) {
    message := ""
    
    if tutorDiscord != "" {
        message += fmt.Sprintf("<@%s> ", tutorDiscord)
    }
    if studentDiscord != "" {
        message += fmt.Sprintf("<@%s> ", studentDiscord)
    }
    
    message += fmt.Sprintf("\n✅ **Booking Accepted!**\n\n**Tutor:** %s\n**Student:** %s\n**When:** %s at %s\n\nPlease show up on time!",
        tutorName, studentName, date, time)
    
    SendNotification(message)
}

func SendBookingCancelledNotificationWithMentions(tutorName, tutorDiscord, studentName, studentDiscord, date, time string) {
    message := ""
    
    if tutorDiscord != "" {
        message += fmt.Sprintf("<@%s> ", tutorDiscord)
    }
    if studentDiscord != "" {
        message += fmt.Sprintf("<@%s> ", studentDiscord)
    }
    
    message += fmt.Sprintf("\n❌ **Booking Cancelled!**\n\n**Tutor:** %s\n**Student:** %s\n**When:** %s at %s",
        tutorName, studentName, date, time)
    
    SendNotification(message)
}

// Legacy functions without mentions (for backward compatibility)
func SendBookingNotification(tutorName, studentName, date, time, topic string) {
    message := fmt.Sprintf("📚 **New Booking!**\n**Tutor:** %s\n**Student:** %s\n**When:** %s at %s\n**Topic:** %s", 
        tutorName, studentName, date, time, topic)
    SendNotification(message)
}

func SendBookingAcceptedNotification(tutorName, studentName, date, time string) {
    message := fmt.Sprintf("✅ **Booking Accepted!**\n**Tutor:** %s\n**Student:** %s\n**When:** %s at %s\n\nPlease show up on time!", 
        tutorName, studentName, date, time)
    SendNotification(message)
}

func SendBookingCancelledNotification(tutorName, studentName, date, time string) {
    message := fmt.Sprintf("❌ **Booking Cancelled!**\n**Tutor:** %s\n**Student:** %s\n**When:** %s at %s", 
        tutorName, studentName, date, time)
    SendNotification(message)
}
