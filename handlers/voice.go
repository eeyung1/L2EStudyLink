package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VoiceResponse struct {
	AudioURL string `json:"audio_url"`
	Text     string `json:"text"`
}

func HandleVoiceUpload(c *gin.Context) {
	// 1. Limit upload size (5MB for voice notes is plenty)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)

	// 2. Get the file from multipart form
	file, err := c.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not get audio file or file exceeds 5MB limit"})
		return
	}

	// 3. Ensure uploads directory exists
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare storage"})
		return
	}

	// 4. Safely determine file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".webm" && ext != ".ogg" && ext != ".mp4" && ext != ".wav" && ext != ".m4a" && ext != ".mp3" {
		ext = ".webm"
	}

	// 5. Create a unique filename and save
	fileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// 6. Respond with the path (Transcription step comes next)
	resp := VoiceResponse{
		AudioURL: "/" + filepath.ToSlash(filePath),
		Text:     "Audio received. Transcription pending...",
	}

	c.JSON(http.StatusOK, resp)
}
