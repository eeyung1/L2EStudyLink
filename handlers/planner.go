package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

// PlannerBlock is a recurring timetable entry. UserID is checked even when the
// caller supplies data directly rather than using LoadWeeklyPlannerMetrics.
type PlannerBlock struct {
	ID, UserID         int64
	DayOfWeek          int
	StartTime, EndTime string
}

type PlannerReflection struct {
	ID, UserID  int64
	TimetableID *int64
	LogDate     string
	Completed   bool
}

type PlannerDayMetrics struct {
	Date                 string `json:"date"`
	ScheduledBlocks      int    `json:"scheduled_blocks"`
	ScheduledMinutes     int    `json:"scheduled_minutes"`
	ReflectedBlocks      int    `json:"reflected_blocks"`
	ReflectedMinutes     int    `json:"reflected_minutes"`
	MissingReflections   int    `json:"missing_reflections"`
	PendingReflections   int    `json:"pending_reflections"`
	UnmatchedReflections int    `json:"unmatched_reflections"`
}

// WeeklyPlannerMetrics describes reflection coverage, not measured work or
// learning quality. Minutes come from the CURRENT recurring timetable; deleted
// blocks have no duration snapshot and historical schedule edits are unknowable.
// WeekEnd is exclusive. Adherence uses due blocks (ended by AsOf, or already
// reflected), so future/unended blocks without a reflection are pending, not
// missing. Streaks count completed-reflection dates, including detached logs,
// using the existing today-or-yesterday grace rule from CalculateStreaks.
type WeeklyPlannerMetrics struct {
	Version              string              `json:"version"`
	Timezone             string              `json:"timezone"`
	WeekStart            string              `json:"week_start"`
	WeekEnd              string              `json:"week_end"`
	AsOf                 string              `json:"as_of"`
	ScheduleBasis        string              `json:"schedule_basis"`
	MinutesBasis         string              `json:"minutes_basis"`
	ScheduledBlocks      int                 `json:"scheduled_blocks"`
	ScheduledMinutes     int                 `json:"scheduled_minutes"`
	DueBlocks            int                 `json:"due_blocks"`
	ReflectedBlocks      int                 `json:"reflected_blocks"`
	ReflectedMinutes     int                 `json:"reflected_minutes"`
	AdherencePercent     float64             `json:"adherence_percent"`
	MissingReflections   int                 `json:"missing_reflections"`
	PendingReflections   int                 `json:"pending_reflections"`
	UnmatchedReflections int                 `json:"unmatched_reflections"`
	ActiveDays           int                 `json:"active_days"`
	StreakAtStart        int                 `json:"streak_at_start"`
	StreakAtEnd          int                 `json:"streak_at_end"`
	StreakChange         int                 `json:"streak_change"`
	Days                 []PlannerDayMetrics `json:"days"`
}

// CalculateWeeklyPlannerMetrics normalizes the supplied instant to its Monday
// in an explicit timezone. Date-only reflection dates are interpreted as local
// calendar dates without converting them through UTC. Callers must supply AsOf
// explicitly; identical inputs produce identical results, including past weeks.
// Block durations are nominal wall-clock minutes, including overnight blocks,
// rather than elapsed UTC time across a daylight-saving change.
func CalculateWeeklyPlannerMetrics(userID int64, week, asOf time.Time, location *time.Location, blocks []PlannerBlock, reflections []PlannerReflection) (WeeklyPlannerMetrics, error) {
	var result WeeklyPlannerMetrics
	if userID <= 0 || location == nil || week.IsZero() || asOf.IsZero() {
		return result, fmt.Errorf("planner requires user, week, as-of time and timezone")
	}
	localWeek := week.In(location)
	start := time.Date(localWeek.Year(), localWeek.Month(), localWeek.Day(), 0, 0, 0, 0, location)
	start = start.AddDate(0, 0, -(int(start.Weekday())+6)%7)
	end := start.AddDate(0, 0, 7)
	localNow := asOf.In(location)
	today := localNow.Format(time.DateOnly)
	calendarStart := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	result = WeeklyPlannerMetrics{
		Version: "1", Timezone: location.String(), WeekStart: start.Format(time.DateOnly), WeekEnd: end.Format(time.DateOnly), AsOf: localNow.Format(time.RFC3339),
		ScheduleBasis: "current_recurring_timetable", MinutesBasis: "scheduled_minutes_with_completed_reflection", Days: make([]PlannerDayMetrics, 7),
	}
	type occurrence struct {
		day, minutes int
		end          time.Time
	}
	schedule := make(map[int64]occurrence)
	for i := range result.Days {
		result.Days[i].Date = start.AddDate(0, 0, i).Format(time.DateOnly)
	}
	for _, block := range blocks {
		if block.UserID != userID {
			continue
		}
		if block.ID <= 0 || block.DayOfWeek < 0 || block.DayOfWeek > 6 {
			return WeeklyPlannerMetrics{}, fmt.Errorf("invalid planner block %d", block.ID)
		}
		if _, exists := schedule[block.ID]; exists {
			return WeeklyPlannerMetrics{}, fmt.Errorf("duplicate planner block %d", block.ID)
		}
		from, err := time.Parse("15:04", block.StartTime)
		if err != nil {
			return WeeklyPlannerMetrics{}, fmt.Errorf("block %d start time: %w", block.ID, err)
		}
		to, err := time.Parse("15:04", block.EndTime)
		if err != nil {
			return WeeklyPlannerMetrics{}, fmt.Errorf("block %d end time: %w", block.ID, err)
		}
		fromMinute, toMinute := from.Hour()*60+from.Minute(), to.Hour()*60+to.Minute()
		minutes := toMinute - fromMinute
		if minutes == 0 {
			return WeeklyPlannerMetrics{}, fmt.Errorf("block %d has zero duration", block.ID)
		}
		overnight := minutes < 0
		if overnight {
			minutes += 24 * 60
		}
		day := (block.DayOfWeek + 6) % 7
		date := start.AddDate(0, 0, day)
		if overnight {
			date = date.AddDate(0, 0, 1)
		}
		finish := time.Date(date.Year(), date.Month(), date.Day(), to.Hour(), to.Minute(), 0, 0, location)
		schedule[block.ID] = occurrence{day: day, minutes: minutes, end: finish}
		result.Days[day].ScheduledBlocks++
		result.Days[day].ScheduledMinutes += minutes
	}
	dailyCounts := make(map[string]int)
	matched := make(map[int64]bool)
	seen := make(map[int64]bool)
	for _, reflection := range reflections {
		if reflection.UserID != userID || !reflection.Completed {
			continue
		}
		if reflection.ID > 0 && seen[reflection.ID] {
			continue
		}
		if reflection.ID > 0 {
			seen[reflection.ID] = true
		}
		date, err := time.Parse(time.DateOnly, reflection.LogDate)
		if err != nil {
			return WeeklyPlannerMetrics{}, fmt.Errorf("reflection %d date: %w", reflection.ID, err)
		}
		if reflection.LogDate > today || reflection.LogDate >= result.WeekEnd {
			continue
		}
		dailyCounts[reflection.LogDate]++
		if reflection.LogDate < result.WeekStart {
			continue
		}
		day := int(date.Sub(calendarStart).Hours() / 24)
		var block occurrence
		found := false
		if reflection.TimetableID != nil {
			block, found = schedule[*reflection.TimetableID]
		}
		if !found || block.day != day {
			result.Days[day].UnmatchedReflections++
			continue
		}
		matched[*reflection.TimetableID] = true
	}
	for id, block := range schedule {
		day := &result.Days[block.day]
		if matched[id] {
			day.ReflectedBlocks++
			day.ReflectedMinutes += block.minutes
		} else if !block.end.After(asOf) {
			day.MissingReflections++
		} else {
			day.PendingReflections++
		}
	}
	for _, day := range result.Days {
		result.ScheduledBlocks += day.ScheduledBlocks
		result.ScheduledMinutes += day.ScheduledMinutes
		result.ReflectedBlocks += day.ReflectedBlocks
		result.ReflectedMinutes += day.ReflectedMinutes
		result.MissingReflections += day.MissingReflections
		result.PendingReflections += day.PendingReflections
		result.UnmatchedReflections += day.UnmatchedReflections
		if dailyCounts[day.Date] > 0 {
			result.ActiveDays++
		}
	}
	result.DueBlocks = result.ReflectedBlocks + result.MissingReflections
	if result.DueBlocks > 0 {
		result.AdherencePercent = math.Round(10000*float64(result.ReflectedBlocks)/float64(result.DueBlocks)) / 100
	}
	if !asOf.Before(start) {
		// Use UTC calendar labels with CalculateStreaks; the dates are already local.
		result.StreakAtStart, _ = CalculateStreaks(dailyCounts, calendarStart.AddDate(0, 0, -1))
		cutoff := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.UTC)
		last := calendarStart.AddDate(0, 0, 6)
		if cutoff.After(last) {
			cutoff = last
		}
		result.StreakAtEnd, _ = CalculateStreaks(dailyCounts, cutoff)
		result.StreakChange = result.StreakAtEnd - result.StreakAtStart
	}
	return result, nil
}

// LoadWeeklyPlannerMetrics reads only the requested user's rows in one stable,
// read-only snapshot. Reflection text and other users' data are never loaded.
// The caller must obtain userID from the authenticated session, not client JSON.
func LoadWeeklyPlannerMetrics(ctx context.Context, database *sql.DB, userID int64, week, asOf time.Time, location *time.Location) (WeeklyPlannerMetrics, error) {
	empty, err := CalculateWeeklyPlannerMetrics(userID, week, asOf, location, nil, nil)
	if err != nil {
		return empty, err
	}
	if database == nil {
		return empty, fmt.Errorf("planner database unavailable")
	}
	tx, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id, user_id, day_of_week, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI') FROM timetable WHERE user_id=$1 ORDER BY id`, userID)
	if err != nil {
		return empty, err
	}
	blocks := make([]PlannerBlock, 0)
	for rows.Next() {
		var block PlannerBlock
		if err := rows.Scan(&block.ID, &block.UserID, &block.DayOfWeek, &block.StartTime, &block.EndTime); err != nil {
			rows.Close()
			return empty, err
		}
		blocks = append(blocks, block)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return empty, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT id, user_id, timetable_id, to_char(log_date, 'YYYY-MM-DD'), completed FROM activity_logs WHERE user_id=$1 AND completed=true AND log_date<$2::date AND log_date<=$3::date ORDER BY log_date,id`, userID, empty.WeekEnd, asOf.In(location).Format(time.DateOnly))
	if err != nil {
		return empty, err
	}
	reflections := make([]PlannerReflection, 0)
	for rows.Next() {
		var reflection PlannerReflection
		if err := rows.Scan(&reflection.ID, &reflection.UserID, &reflection.TimetableID, &reflection.LogDate, &reflection.Completed); err != nil {
			rows.Close()
			return empty, err
		}
		reflections = append(reflections, reflection)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return empty, err
	}
	result, err := CalculateWeeklyPlannerMetrics(userID, week, asOf, location, blocks, reflections)
	if err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
