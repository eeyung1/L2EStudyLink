package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleVoiceUpload_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/voice/upload", HandleVoiceUpload)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio", "sample.webm")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	sampleData := []byte("fake-audio-bytes-for-testing")
	if _, err := io.Copy(part, bytes.NewReader(sampleData)); err != nil {
		t.Fatalf("Failed to write sample audio: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp VoiceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if !strings.HasPrefix(resp.AudioURL, "/uploads/") || !strings.HasSuffix(resp.AudioURL, ".webm") {
		t.Errorf("Unexpected audio URL format: %s", resp.AudioURL)
	}
	if resp.Text == "" {
		t.Errorf("Expected non-empty placeholder text")
	}

	// Verify file was saved on disk
	localPath := strings.TrimPrefix(resp.AudioURL, "/")
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Errorf("Saved file does not exist at %s", localPath)
	} else {
		// Clean up uploaded test file
		_ = os.Remove(localPath)
	}
}

func TestHandleVoiceUpload_MissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/voice/upload", HandleVoiceUpload)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("other", "field")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for missing file, got %d", w.Code)
	}
}

func TestHandleVoiceUpload_TooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/voice/upload", HandleVoiceUpload)

	// Create oversized payload (>5MB)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio", "large.webm")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	// 5.5MB of data
	largeData := make([]byte, (5<<20)+512*1024)
	if _, err := part.Write(largeData); err != nil {
		t.Fatalf("Failed to write large data: %v", err)
	}
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for file exceeding limit, got %d", w.Code)
	}
}
