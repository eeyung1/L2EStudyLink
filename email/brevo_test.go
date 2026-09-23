package email

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendEmailUsesBrevoTransactionalAPI(t *testing.T) {
	t.Setenv("BREVO_API_KEY", "test-key")
	t.Setenv("BREVO_FROM_EMAIL", "verified@gmail.com")
	t.Setenv("BREVO_FROM_NAME", "L2EStudyLink")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v3/smtp/email" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("api-key") != "test-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("incorrect Brevo headers")
		}
		var payload struct {
			Sender struct{ Email, Name string } `json:"sender"`
			To []struct{ Email string } `json:"to"`
			Subject string `json:"subject"`
			HTML string `json:"htmlContent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.Sender.Email != "verified@gmail.com" || payload.Sender.Name != "L2EStudyLink" || len(payload.To) != 1 || payload.To[0].Email != "student@example.com" || payload.Subject != "Code" || payload.HTML != "<p>123456</p>" {
			t.Errorf("wrong email payload: %+v", payload)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"messageId":"test"}`))
	}))
	defer server.Close()

	if err := sendEmail(server.Client(), server.URL+"/v3/smtp/email", "student@example.com", "Code", "<p>123456</p>"); err != nil {
		t.Fatal(err)
	}
}

func TestSendEmailRejectsBrevoErrors(t *testing.T) {
	t.Setenv("BREVO_API_KEY", "test-key")
	t.Setenv("BREVO_FROM_EMAIL", "verified@gmail.com")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	if err := sendEmail(server.Client(), server.URL, "student@example.com", "Code", "<p>123456</p>"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected Brevo 401, got %v", err)
	}
}

func TestSendEmailRequiresConfiguration(t *testing.T) {
	t.Setenv("BREVO_API_KEY", "")
	t.Setenv("BREVO_FROM_EMAIL", "")
	if err := sendEmail(http.DefaultClient, "http://localhost", "student@example.com", "Code", "code"); err == nil {
		t.Fatal("missing Brevo credentials should fail before sending")
	}
}
