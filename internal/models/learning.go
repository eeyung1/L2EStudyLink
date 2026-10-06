package models

import (
	"time"
)

type LearningEntry struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id"`
	Title      string    `json:"title" db:"title"`
	Content    string    `json:"content" db:"content"`
	Category   string    `json:"category" db:"category"`
	HoursSpent float64   `json:"hours_spent" db:"hours_spent"`
	EntryDate  time.Time `json:"entry_date" db:"entry_date"`
	AudioURL   *string   `json:"audio_url" db:"audio_url"` // Pointer handles NULL values
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
