package email

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
)

type ResendRequest struct {
    From    string `json:"from"`
    To      string `json:"to"`
    Subject string `json:"subject"`
    Html    string `json:"html"`
}

var apiKey = os.Getenv("RESEND_API_KEY")

func SendEmail(to, subject, htmlContent string) error {
    if apiKey == "" {
        return fmt.Errorf("RESEND_API_KEY not set")
    }

    request := ResendRequest{
        From:    "StudyLink <onboarding@resend.dev>",
        To:      to,
        Subject: subject,
        Html:    htmlContent,
    }

    jsonData, err := json.Marshal(request)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("Resend API error: %d", resp.StatusCode)
    }

    return nil
}

func SendBookingNotification(to, name, tutorName, studentName, date, time, topic string) error {
    html := fmt.Sprintf(`
        <h2>New Booking Created!</h2>
        <p>Hello %s,</p>
        <p>A new study session has been booked:</p>
        <ul>
            <li><strong>Tutor:</strong> %s</li>
            <li><strong>Student:</strong> %s</li>
            <li><strong>Date:</strong> %s</li>
            <li><strong>Time:</strong> %s</li>
            <li><strong>Topic:</strong> %s</li>
        </ul>
        <p>Log in to your dashboard to view and manage this booking.</p>
        <p>StudyLink Team</p>
    `, name, tutorName, studentName, date, time, topic)

    return SendEmail(to, "New Study Session Booking", html)
}

func SendBookingAcceptedNotification(to, name, tutorName, studentName, date, time string) error {
    html := fmt.Sprintf(`
        <h2>Booking Accepted! ✅</h2>
        <p>Hello %s,</p>
        <p>Your study session has been accepted:</p>
        <ul>
            <li><strong>Tutor:</strong> %s</li>
            <li><strong>Student:</strong> %s</li>
            <li><strong>Date:</strong> %s</li>
            <li><strong>Time:</strong> %s</li>
        </ul>
        <p>Please show up on time!</p>
        <p>StudyLink Team</p>
    `, name, tutorName, studentName, date, time)

    return SendEmail(to, "Booking Accepted - Study Session Confirmed", html)
}

func SendBookingCancelledNotification(to, name, tutorName, studentName, date, time string) error {
    html := fmt.Sprintf(`
        <h2>Booking Cancelled ❌</h2>
        <p>Hello %s,</p>
        <p>A study session has been cancelled:</p>
        <ul>
            <li><strong>Tutor:</strong> %s</li>
            <li><strong>Student:</strong> %s</li>
            <li><strong>Date:</strong> %s</li>
            <li><strong>Time:</strong> %s</li>
        </ul>
        <p>StudyLink Team</p>
    `, name, tutorName, studentName, date, time)

    return SendEmail(to, "Booking Cancelled", html)
}
