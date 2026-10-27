package focus

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var testEpoch = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

type testClock struct {
	mu      sync.Mutex
	instant time.Time
}

func (c *testClock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.instant }
func (c *testClock) advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instant = c.instant.Add(duration)
}
func participant(id int64) ParticipantInput {
	return ParticipantInput{UserID: id, DisplayName: fmt.Sprintf("Learner %d", id), Target: "Complete one Go exercise"}
}
func newTestStore(t *testing.T, config Config) (*Store, *testClock) {
	t.Helper()
	clock := &testClock{instant: testEpoch}
	config.Clock = clock.now
	store, err := NewStore(config)
	if err != nil {
		t.Fatal(err)
	}
	return store, clock
}

func TestTimerPhaseBoundaries(t *testing.T) {
	cases := []struct {
		name             string
		offset           time.Duration
		phase            Phase
		cycle, remaining int64
	}{
		{"before room starts", -time.Second, Waiting, 0, 1},
		{"start", 0, Focusing, 1, 1500},
		{"fractional countdown", 500 * time.Millisecond, Focusing, 1, 1500},
		{"last focus instant", FocusDuration - time.Nanosecond, Focusing, 1, 1},
		{"break boundary", FocusDuration, Break, 1, 300},
		{"last break instant", cycleDuration - time.Nanosecond, Break, 1, 1},
		{"next focus", cycleDuration, Focusing, 2, 1500},
		{"next break", cycleDuration + FocusDuration, Break, 2, 300},
		{"many cycles", 24 * time.Hour, Focusing, 49, 1500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now := testEpoch.Add(tc.offset)
			timer, err := TimerAt(testEpoch, now)
			if err != nil {
				t.Fatal(err)
			}
			if timer.Phase != tc.phase || timer.Cycle != tc.cycle || timer.SecondsRemaining != tc.remaining || !timer.ServerTime.Equal(now) || !timer.EndsAt.After(now) {
				t.Fatalf("timer: %+v", timer)
			}
		})
	}
	if _, err := TimerAt(time.Time{}, testEpoch); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("zero epoch accepted")
	}
	if _, err := TimerAt(testEpoch, time.Time{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("zero time accepted")
	}
	if _, err := TimerAt(time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC), testEpoch); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("saturated duration accepted")
	}
}

func TestTimerUsesElapsedTimeAcrossDST(t *testing.T) {
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 3, 8, 1, 50, 0, 0, zone)
	timer, err := TimerAt(start, start.Add(25*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if timer.Phase != Break || timer.SecondsRemaining != 300 || timer.ServerTime.Location() != time.UTC {
		t.Fatalf("DST shifted timer: %+v", timer)
	}
}

func TestRoomLateJoinAndRejoinShareClock(t *testing.T) {
	store, clock := newTestStore(t, Config{MaxParticipants: 2, PresenceTTL: time.Hour, IdleTTL: 2 * time.Hour})
	first, err := store.Create("  Deep work  ", participant(2))
	if err != nil {
		t.Fatal(err)
	}
	if first.Title != "Deep work" || len(first.ID) != 32 || first.Timer.Phase != Focusing {
		t.Fatalf("create: %+v", first)
	}
	clock.advance(26 * time.Minute)
	joined, err := store.Join(first.ID, participant(1))
	if err != nil {
		t.Fatal(err)
	}
	if joined.Timer.Phase != Break || joined.Timer.Cycle != 1 || joined.Timer.SecondsRemaining != 240 || !joined.Epoch.Equal(first.Epoch) {
		t.Fatalf("late join reset clock: %+v", joined)
	}
	if joined.Participants[0].UserID != 1 || joined.Participants[1].UserID != 2 {
		t.Fatalf("roster unsorted: %+v", joined.Participants)
	}
	joinedAt := joined.Participants[0].JoinedAt
	clock.advance(time.Second)
	input := participant(1)
	input.Target = "Write the parser test"
	repeated, err := store.Join(first.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeated.Participants) != 2 || !repeated.Participants[0].JoinedAt.Equal(joinedAt) || repeated.Participants[0].Target != input.Target {
		t.Fatalf("rejoin: %+v", repeated)
	}
	if _, err := store.Join(first.ID, participant(3)); !errors.Is(err, ErrRoomFull) {
		t.Fatalf("room overflow: %v", err)
	}
	repeated.Participants[0].Target = "tampered snapshot"
	copy, err := store.Get(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copy.Participants[0].Target != input.Target {
		t.Fatal("snapshot aliases store")
	}
	if err := store.Leave(first.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := store.Leave(first.ID, 1); !errors.Is(err, ErrNotMember) {
		t.Fatalf("unknown departure: %v", err)
	}
	if _, err := store.Join(first.ID, participant(3)); err != nil {
		t.Fatalf("capacity not released: %v", err)
	}
}

func TestPresenceExpiryHeartbeatAndIdleCleanup(t *testing.T) {
	store, clock := newTestStore(t, Config{MaxRooms: 1, PresenceTTL: 5 * time.Second, IdleTTL: 15 * time.Second})
	room, err := store.Create("Focus", participant(1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Join(room.ID, participant(2)); err != nil {
		t.Fatal(err)
	}
	clock.advance(3 * time.Second)
	if _, err := store.Heartbeat(room.ID, 1, "Finish the test"); err != nil {
		t.Fatal(err)
	}
	clock.advance(2 * time.Second)
	snapshot, err := store.Get(room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Participants) != 1 || snapshot.Participants[0].UserID != 1 {
		t.Fatalf("expiry boundary: %+v", snapshot)
	}
	if _, err := store.Heartbeat(room.ID, 2, "Late heartbeat"); !errors.Is(err, ErrNotMember) {
		t.Fatalf("expired heartbeat rejoined: %v", err)
	}
	if _, err := store.Create("Second", participant(3)); !errors.Is(err, ErrCapacity) {
		t.Fatalf("room capacity: %v", err)
	}
	clock.advance(3 * time.Second)
	snapshot, err = store.Get(room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Participants) != 0 {
		t.Fatalf("stale participant retained: %+v", snapshot)
	}
	clock.advance(9 * time.Second) // 17s since creation, 14s since last real activity.
	if _, err := store.Get(room.ID); err != nil {
		t.Fatalf("room expired too early: %v", err)
	}
	clock.advance(time.Second)
	if _, err := store.Get(room.ID); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("idle room retained: %v", err)
	}
	next, err := store.Create("Next session", participant(3))
	if err != nil {
		t.Fatalf("room capacity not recovered: %v", err)
	}
	if next.ID == room.ID || next.Timer.SecondsRemaining != 1500 {
		t.Fatalf("new room inherited old state: %+v", next)
	}
}

func TestRoomValidationAndDetachedList(t *testing.T) {
	for _, config := range []Config{{MaxRooms: -1}, {MaxParticipants: -1}, {PresenceTTL: -time.Second}, {PresenceTTL: time.Minute, IdleTTL: time.Second}} {
		if _, err := NewStore(config); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("invalid config accepted: %+v", config)
		}
	}
	store, _ := newTestStore(t, Config{})
	list, err := store.List()
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("empty list: %+v %v", list, err)
	}
	for _, input := range []ParticipantInput{{UserID: 0, DisplayName: "Learner", Target: "Go"}, {UserID: 1, DisplayName: "", Target: "Go"}, {UserID: 1, DisplayName: "Learner", Target: ""}, {UserID: 1, DisplayName: "Learner", Target: strings.Repeat("x", 161)}, {UserID: 1, DisplayName: string([]byte{255}), Target: "Go"}} {
		if _, err := store.Create("Focus", input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("invalid participant accepted: %+v", input)
		}
	}
	if _, err := store.Create(strings.Repeat("x", 81), participant(1)); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("long title accepted")
	}
	if _, err := store.Get("missing"); !errors.Is(err, ErrRoomNotFound) {
		t.Fatal("missing room found")
	}
	if _, err := store.Join("missing", participant(1)); !errors.Is(err, ErrRoomNotFound) {
		t.Fatal("joined missing room")
	}
	room, err := store.Create("Focus", participant(1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Heartbeat(room.ID, 99, "Go"); !errors.Is(err, ErrNotMember) {
		t.Fatal("nonmember heartbeat accepted")
	}
	if _, err := store.Heartbeat(room.ID, 1, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("empty target accepted")
	}
	if err := store.Leave(room.ID, 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("invalid departure accepted")
	}
	if _, err := store.Create("Second", participant(2)); err != nil {
		t.Fatal(err)
	}
	list, err = store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID > list[1].ID {
		t.Fatalf("list order: %+v", list)
	}
	list[0].Participants[0].Target = "changed"
	actual, err := store.Get(list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Participants[0].Target == "changed" {
		t.Fatal("list aliases roster")
	}
}

func TestConcurrentRoomCapacityAndRoster(t *testing.T) {
	store, _ := newTestStore(t, Config{MaxParticipants: 10})
	room, err := store.Create("Concurrent focus", participant(1))
	if err != nil {
		t.Fatal(err)
	}
	var accepted atomic.Int64
	var group sync.WaitGroup
	for userID := int64(2); userID <= 101; userID++ {
		group.Add(1)
		go func(id int64) {
			defer group.Done()
			_, err := store.Join(room.ID, participant(id))
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, ErrRoomFull) {
				t.Errorf("join: %v", err)
			}
		}(userID)
	}
	group.Wait()
	if accepted.Load() != 9 {
		t.Fatalf("accepted %d additional participants", accepted.Load())
	}
	snapshot, err := store.Get(room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Participants) != 10 {
		t.Fatalf("capacity violated: %d", len(snapshot.Participants))
	}
	for _, member := range snapshot.Participants {
		group.Add(1)
		go func(id int64) {
			defer group.Done()
			for i := 0; i < 25; i++ {
				if _, err := store.Heartbeat(room.ID, id, "Keep focusing"); err != nil {
					t.Errorf("heartbeat: %v", err)
				}
				if _, err := store.List(); err != nil {
					t.Errorf("list: %v", err)
				}
			}
			if err := store.Leave(room.ID, id); err != nil {
				t.Errorf("leave: %v", err)
			}
		}(member.UserID)
	}
	group.Wait()
	snapshot, err = store.Get(room.ID)
	if err != nil || len(snapshot.Participants) != 0 {
		t.Fatalf("roster after departures: %+v %v", snapshot, err)
	}
}
