package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCalculateStreaks_Empty(t *testing.T) {
	today, _ := time.Parse("2006-01-02", "2026-10-27")
	counts := map[string]int{}
	curr, longest := CalculateStreaks(counts, today)
	if curr != 0 || longest != 0 {
		t.Fatalf("Expected (0, 0), got (%d, %d)", curr, longest)
	}
}

func TestCalculateStreaks_ActiveToday(t *testing.T) {
	today, _ := time.Parse("2006-01-02", "2026-10-27")
	counts := map[string]int{
		"2026-10-25": 1,
		"2026-10-26": 2,
		"2026-10-27": 1,
	}
	curr, longest := CalculateStreaks(counts, today)
	if curr != 3 {
		t.Errorf("Expected current streak 3, got %d", curr)
	}
	if longest != 3 {
		t.Errorf("Expected longest streak 3, got %d", longest)
	}
}

func TestCalculateStreaks_ActiveYesterdayNotToday(t *testing.T) {
	// Learner logged yesterday, but hasn't logged today yet -> streak is still active!
	today, _ := time.Parse("2006-01-02", "2026-10-27")
	counts := map[string]int{
		"2026-10-24": 1,
		"2026-10-25": 1,
		"2026-10-26": 3,
	}
	curr, longest := CalculateStreaks(counts, today)
	if curr != 3 {
		t.Errorf("Expected current streak 3, got %d", curr)
	}
	if longest != 3 {
		t.Errorf("Expected longest streak 3, got %d", longest)
	}
}

func TestCalculateStreaks_BrokenStreak(t *testing.T) {
	// Learner logged 2 days ago, missed yesterday and today -> streak is 0
	today, _ := time.Parse("2006-01-02", "2026-10-27")
	counts := map[string]int{
		"2026-10-24": 1,
		"2026-10-25": 2,
	}
	curr, longest := CalculateStreaks(counts, today)
	if curr != 0 {
		t.Errorf("Expected current streak 0, got %d", curr)
	}
	if longest != 2 {
		t.Errorf("Expected longest streak 2, got %d", longest)
	}
}

func TestCalculateStreaks_MultipleStreaks(t *testing.T) {
	// Past streak of 5, current streak of 2
	today, _ := time.Parse("2006-01-02", "2026-10-27")
	counts := map[string]int{
		"2026-09-01": 1,
		"2026-09-02": 1,
		"2026-09-03": 1,
		"2026-09-04": 1,
		"2026-09-05": 1,
		"2026-10-26": 2,
		"2026-10-27": 1,
	}
	curr, longest := CalculateStreaks(counts, today)
	if curr != 2 {
		t.Errorf("Expected current streak 2, got %d", curr)
	}
	if longest != 5 {
		t.Errorf("Expected longest streak 5, got %d", longest)
	}
}

func TestGetUserStreaks_NoDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/me/streaks", GetUserStreaks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/streaks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Since "db" is not set on gin.Context, it returns 500 cleanly
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when DB is missing, got %d", w.Code)
	}
}
