package email

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
)

type brevoAddress struct {
    Email string `json:"email"`
    Name  string `json:"name,omitempty"`
}

type brevoRequest struct {
    Sender brevoAddress   `json:"sender"`
    To     []brevoAddress `json:"to"`
    Subject string        `json:"subject"`
    HTML    string        `json:"htmlContent"`
}

func SendEmail(to, subject, htmlContent string) error {
    return sendEmail(&http.Client{Timeout: 10 * time.Second}, "https://api.brevo.com/v3/smtp/email", to, subject, htmlContent)
}

func sendEmail(client *http.Client, endpoint, to, subject, htmlContent string) error {
    apiKey := os.Getenv("BREVO_API_KEY")
    if apiKey == "" {
        return fmt.Errorf("BREVO_API_KEY not set")
    }
    sender := os.Getenv("BREVO_FROM_EMAIL")
    if sender == "" {
        return fmt.Errorf("BREVO_FROM_EMAIL not set")
    }

    request := brevoRequest{
        Sender: brevoAddress{Email: sender, Name: os.Getenv("BREVO_FROM_NAME")},
        To: []brevoAddress{{Email: to}},
        Subject: subject,
        HTML: htmlContent,
    }

    jsonData, err := json.Marshal(request)
    if err != nil {
        return err
    }

    req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonData))
    if err != nil {
        return err
    }

    req.Header.Set("api-key", apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("Brevo API error: %d", resp.StatusCode)
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
