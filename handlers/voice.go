package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type VoiceResponse struct {
	AudioURL string `json:"audio_url"`
	Text     string `json:"text"`
}

func HandleVoiceUpload(w http.ResponseWriter, r *http.Request) {
	// 1. Limit upload size (5MB for voice notes is plenty)
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	
	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	// 2. Get the file from the form
	file, _, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "Could not get audio file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 3. Create a unique filename
	fileName := fmt.Sprintf("%s.webm", uuid.New().String())
	filePath := filepath.Join("uploads", fileName)

	// 4. Save to local disk
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file locally", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	// 5. Respond with the path (Transcription step comes next)
	resp := VoiceResponse{
		AudioURL: "/" + filePath,
		Text:     "Audio received. Transcription pending...",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
