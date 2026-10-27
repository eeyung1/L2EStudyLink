package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func plannerInstant(value string) time.Time {
	date, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return date
}
func plannerBlock(id int64, day int, start, end string) PlannerBlock {
	return PlannerBlock{ID: id, UserID: 1, DayOfWeek: day, StartTime: start, EndTime: end}
}
func plannerLog(id, block int64, date string) PlannerReflection {
	var parent *int64
	if block != 0 {
		parent = &block
	}
	return PlannerReflection{ID: id, UserID: 1, TimetableID: parent, LogDate: date, Completed: true}
}
func calculatePlanner(t *testing.T, week, asOf string, blocks []PlannerBlock, logs []PlannerReflection) WeeklyPlannerMetrics {
	t.Helper()
	m, err := CalculateWeeklyPlannerMetrics(1, plannerInstant(week), plannerInstant(asOf), time.UTC, blocks, logs)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPlannerWeeklyCoverage(t *testing.T) {
	blocks := []PlannerBlock{plannerBlock(1, 1, "09:00", "10:00"), plannerBlock(2, 2, "09:00", "10:30"), plannerBlock(3, 3, "14:00", "15:00")}
	logs := []PlannerReflection{plannerLog(1, 1, "2026-10-26"), plannerLog(1, 1, "2026-10-26"), plannerLog(2, 1, "2026-10-26"), plannerLog(3, 0, "2026-10-27"), plannerLog(4, 2, "2026-10-26"), plannerLog(5, 3, "2026-10-28")}
	logs[5].Completed = false
	m := calculatePlanner(t, "2026-10-28T12:00:00Z", "2026-11-02T12:00:00Z", blocks, logs)
	if m.WeekStart != "2026-10-26" || m.WeekEnd != "2026-11-02" || len(m.Days) != 7 {
		t.Fatalf("week bounds: %+v", m)
	}
	if m.ScheduledBlocks != 3 || m.ScheduledMinutes != 210 || m.ReflectedBlocks != 1 || m.ReflectedMinutes != 60 || m.DueBlocks != 3 || m.MissingReflections != 2 || m.PendingReflections != 0 || m.AdherencePercent != 33.33 {
		t.Fatalf("coverage: %+v", m)
	}
	if m.UnmatchedReflections != 2 || m.ActiveDays != 2 {
		t.Fatalf("unmatched/days: %+v", m)
	}
	if m.Days[0].ReflectedMinutes != 60 || m.Days[1].MissingReflections != 1 || m.Days[2].MissingReflections != 1 {
		t.Fatalf("daily breakdown: %+v", m.Days)
	}
	// Reordering the input must not change the report or mutate the inputs.
	before, _ := json.Marshal(logs)
	reversed := append([]PlannerReflection(nil), logs...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	again := calculatePlanner(t, "2026-10-28T12:00:00Z", "2026-11-02T12:00:00Z", blocks, reversed)
	if !reflect.DeepEqual(m, again) {
		t.Fatalf("input order changed results: %+v / %+v", m, again)
	}
	after, _ := json.Marshal(logs)
	if string(before) != string(after) {
		t.Fatal("inputs mutated")
	}
}

func TestPlannerPartialAndFutureWeeks(t *testing.T) {
	blocks := []PlannerBlock{plannerBlock(1, 1, "09:00", "10:00"), plannerBlock(2, 2, "09:00", "10:00"), plannerBlock(3, 3, "09:00", "10:00"), plannerBlock(4, 4, "09:00", "10:00")}
	logs := []PlannerReflection{plannerLog(1, 1, "2026-10-26"), plannerLog(2, 4, "2026-10-29")}
	cases := []struct {
		asOf                             string
		due, missing, pending, reflected int
		adherence                        float64
	}{
		{"2026-10-25T23:59:59Z", 0, 0, 4, 0, 0},
		{"2026-10-28T09:59:59Z", 2, 1, 2, 1, 50},
		{"2026-10-28T10:00:00Z", 3, 2, 1, 1, 33.33},
		{"2026-10-30T00:00:00Z", 4, 2, 0, 2, 50},
	}
	for _, tc := range cases {
		t.Run(tc.asOf, func(t *testing.T) {
			m := calculatePlanner(t, "2026-10-26T00:00:00Z", tc.asOf, blocks, logs)
			if m.DueBlocks != tc.due || m.MissingReflections != tc.missing || m.PendingReflections != tc.pending || m.ReflectedBlocks != tc.reflected || m.AdherencePercent != tc.adherence {
				t.Fatalf("partial week: %+v", m)
			}
		})
	}
}

func TestPlannerStreakChangesAndBoundaries(t *testing.T) {
	prior := []PlannerReflection{plannerLog(1, 0, "2026-10-23"), plannerLog(2, 0, "2026-10-24"), plannerLog(3, 0, "2026-10-25")}
	t.Run("gain", func(t *testing.T) {
		logs := append(append([]PlannerReflection(nil), prior...), plannerLog(4, 0, "2026-10-26"), plannerLog(5, 0, "2026-10-27"), plannerLog(6, 0, "2026-11-02"))
		m := calculatePlanner(t, "2026-10-26T00:00:00Z", "2026-10-28T12:00:00Z", nil, logs)
		if m.StreakAtStart != 3 || m.StreakAtEnd != 5 || m.StreakChange != 2 || m.ActiveDays != 2 || m.UnmatchedReflections != 2 {
			t.Fatalf("streak gain: %+v", m)
		}
	})
	t.Run("loss", func(t *testing.T) {
		m := calculatePlanner(t, "2026-10-26T00:00:00Z", "2026-10-28T12:00:00Z", nil, prior)
		if m.StreakAtStart != 3 || m.StreakAtEnd != 0 || m.StreakChange != -3 {
			t.Fatalf("streak loss: %+v", m)
		}
	})
	t.Run("historical cutoff", func(t *testing.T) {
		logs := []PlannerReflection{plannerLog(1, 0, "2026-11-01"), plannerLog(2, 0, "2026-11-02"), plannerLog(3, 0, "2026-11-03")}
		m := calculatePlanner(t, "2026-10-26T00:00:00Z", "2026-12-01T00:00:00Z", nil, logs)
		if m.StreakAtEnd != 1 || m.ActiveDays != 1 || m.UnmatchedReflections != 1 {
			t.Fatalf("future week leaked: %+v", m)
		}
	})
	t.Run("new year", func(t *testing.T) {
		m := calculatePlanner(t, "2027-01-01T12:00:00Z", "2027-01-04T00:00:00Z", []PlannerBlock{plannerBlock(1, 0, "09:00", "10:00")}, []PlannerReflection{plannerLog(1, 1, "2027-01-03")})
		if m.WeekStart != "2026-12-28" || m.WeekEnd != "2027-01-04" || m.ReflectedBlocks != 1 || m.Days[6].Date != "2027-01-03" {
			t.Fatalf("year boundary: %+v", m)
		}
	})
	t.Run("leap day", func(t *testing.T) {
		m := calculatePlanner(t, "2028-02-29T12:00:00Z", "2028-03-06T00:00:00Z", nil, []PlannerReflection{plannerLog(1, 0, "2028-02-29"), plannerLog(2, 0, "2028-03-01")})
		if m.WeekStart != "2028-02-28" || m.ActiveDays != 2 || m.Days[1].Date != "2028-02-29" {
			t.Fatalf("leap day: %+v", m)
		}
	})
}

func TestPlannerTimezonesAndOvernight(t *testing.T) {
	t.Run("explicit local dates", func(t *testing.T) {
		zone := time.FixedZone("UTC+14", 14*60*60)
		m, err := CalculateWeeklyPlannerMetrics(1, plannerInstant("2026-10-25T11:30:00Z"), plannerInstant("2026-10-26T10:00:00Z"), zone, []PlannerBlock{plannerBlock(1, 1, "00:00", "01:00")}, []PlannerReflection{plannerLog(1, 1, "2026-10-26")})
		if err != nil {
			t.Fatal(err)
		}
		if m.WeekStart != "2026-10-26" || m.Timezone != "UTC+14" || m.ReflectedMinutes != 60 {
			t.Fatalf("local calendar: %+v", m)
		}
	})
	t.Run("DST wall clock duration", func(t *testing.T) {
		zone, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Fatal(err)
		}
		m, err := CalculateWeeklyPlannerMetrics(1, plannerInstant("2026-03-08T15:00:00Z"), plannerInstant("2026-03-09T15:00:00Z"), zone, []PlannerBlock{plannerBlock(1, 0, "01:30", "03:30")}, []PlannerReflection{plannerLog(1, 1, "2026-03-08")})
		if err != nil {
			t.Fatal(err)
		}
		if m.WeekStart != "2026-03-02" || m.ReflectedMinutes != 120 || m.StreakAtEnd != 1 {
			t.Fatalf("DST: %+v", m)
		}
	})
	t.Run("overnight due after midnight", func(t *testing.T) {
		blocks := []PlannerBlock{plannerBlock(1, 1, "23:30", "01:00")}
		m := calculatePlanner(t, "2026-10-26T00:00:00Z", "2026-10-27T00:59:59Z", blocks, nil)
		if m.ScheduledMinutes != 90 || m.PendingReflections != 1 || m.MissingReflections != 0 {
			t.Fatalf("overnight pending: %+v", m)
		}
		m = calculatePlanner(t, "2026-10-26T00:00:00Z", "2026-10-27T01:00:00Z", blocks, nil)
		if m.MissingReflections != 1 || m.PendingReflections != 0 {
			t.Fatalf("overnight due: %+v", m)
		}
	})
}

func TestPlannerEmptyIsolationAndValidation(t *testing.T) {
	week, now := plannerInstant("2026-10-26T00:00:00Z"), plannerInstant("2026-11-02T00:00:00Z")
	m := calculatePlanner(t, "2026-10-26T00:00:00Z", "2026-11-02T00:00:00Z", nil, nil)
	if m.AdherencePercent != 0 || m.ScheduledMinutes != 0 || m.StreakChange != 0 || len(m.Days) != 7 {
		t.Fatalf("empty: %+v", m)
	}
	foreignBlock := PlannerBlock{ID: 1, UserID: 2, DayOfWeek: 99, StartTime: "invalid"}
	foreignLog := PlannerReflection{ID: 1, UserID: 2, Completed: true, LogDate: "invalid"}
	isolated, err := CalculateWeeklyPlannerMetrics(1, week, now, time.UTC, []PlannerBlock{foreignBlock}, []PlannerReflection{foreignLog})
	if err != nil || !reflect.DeepEqual(m, isolated) {
		t.Fatalf("foreign data leaked: %+v %v", isolated, err)
	}
	tests := []struct {
		name      string
		user      int64
		week, now time.Time
		zone      *time.Location
		blocks    []PlannerBlock
		logs      []PlannerReflection
	}{
		{"missing user", 0, week, now, time.UTC, nil, nil},
		{"missing timezone", 1, week, now, nil, nil, nil},
		{"missing week", 1, time.Time{}, now, time.UTC, nil, nil},
		{"missing asof", 1, week, time.Time{}, time.UTC, nil, nil},
		{"bad weekday", 1, week, now, time.UTC, []PlannerBlock{plannerBlock(1, 7, "09:00", "10:00")}, nil},
		{"bad time", 1, week, now, time.UTC, []PlannerBlock{plannerBlock(1, 1, "25:00", "10:00")}, nil},
		{"zero duration", 1, week, now, time.UTC, []PlannerBlock{plannerBlock(1, 1, "09:00", "09:00")}, nil},
		{"duplicate block", 1, week, now, time.UTC, []PlannerBlock{plannerBlock(1, 1, "09:00", "10:00"), plannerBlock(1, 2, "09:00", "10:00")}, nil},
		{"bad reflection date", 1, week, now, time.UTC, nil, []PlannerReflection{plannerLog(1, 0, "invalid")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := CalculateWeeklyPlannerMetrics(tc.user, tc.week, tc.now, tc.zone, tc.blocks, tc.logs); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
	if _, err := LoadWeeklyPlannerMetrics(context.Background(), nil, 1, week, now, time.UTC); err == nil {
		t.Fatal("nil database accepted")
	}
}

func TestPlannerLoadWeeklyMetrics(t *testing.T) {
	database := timetableTestDB(t, false)
	owner, block := timetableFixture(t, database)
	var other int64
	if err := database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Other','planner-other@example.com','test') RETURNING id`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO timetable(user_id,day_of_week,start_time,end_time,activity) VALUES($1,1,'00:00','23:59','Other user secret')`, other); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO activity_logs(user_id,timetable_id,log_date,completed,activity_name) VALUES($1,$2,'2026-10-27',true,'Learn Go'),($1,NULL,'2026-10-28',true,'Deleted block'),($3,NULL,'2026-10-26',true,'Other user secret')`, owner, block, other); err != nil {
		t.Fatal(err)
	}
	week, asOf := plannerInstant("2026-10-26T00:00:00Z"), plannerInstant("2026-11-02T00:00:00Z")
	m, err := LoadWeeklyPlannerMetrics(context.Background(), database, owner, week, asOf, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if m.ScheduledMinutes != 60 || m.ReflectedMinutes != 60 || m.ReflectedBlocks != 1 || m.UnmatchedReflections != 1 || m.ActiveDays != 2 {
		t.Fatalf("loader coverage/isolation: %+v", m)
	}
	encoded, _ := json.Marshal(m)
	if strings.Contains(string(encoded), "secret") {
		t.Fatal("foreign reflection text leaked")
	}
	if _, err := database.Exec(`DELETE FROM timetable WHERE id=$1`, block); err != nil {
		t.Fatal(err)
	}
	m, err = LoadWeeklyPlannerMetrics(context.Background(), database, owner, week, asOf, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if m.ScheduledMinutes != 0 || m.ReflectedMinutes != 0 || m.UnmatchedReflections != 2 || m.ActiveDays != 2 {
		t.Fatalf("detached history: %+v", m)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadWeeklyPlannerMetrics(ctx, database, owner, week, asOf, time.UTC); err == nil {
		t.Fatal("canceled query succeeded")
	}
	// An absent timetable is an error, rather than a plausible all-zero report.
	if _, err := database.Exec(`DROP TABLE timetable CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWeeklyPlannerMetrics(context.Background(), database, owner, week, asOf, time.UTC); err == nil || err == sql.ErrNoRows {
		t.Fatalf("query failure hidden: %v", err)
	}
}
