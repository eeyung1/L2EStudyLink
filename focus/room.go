// Package focus supplies the process-local domain engine for shared Pomodoro
// rooms. It has no HTTP, authentication, persistence, or notification side effects.
package focus

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const FocusDuration = 25 * time.Minute
const BreakDuration = 5 * time.Minute
const cycleDuration = FocusDuration + BreakDuration

var (
	ErrInvalidInput = errors.New("invalid focus room input")
	ErrRoomNotFound = errors.New("focus room not found")
	ErrRoomFull     = errors.New("focus room is full")
	ErrCapacity     = errors.New("focus room capacity reached")
	ErrNotMember    = errors.New("participant is not in this focus room")
)

type Phase string

const (
	Waiting  Phase = "waiting"
	Focusing Phase = "focus"
	Break    Phase = "break"
)

type Timer struct {
	Phase            Phase     `json:"phase"`
	Cycle            int64     `json:"cycle"`
	StartedAt        time.Time `json:"started_at"`
	EndsAt           time.Time `json:"ends_at"`
	ServerTime       time.Time `json:"server_time"`
	SecondsRemaining int64     `json:"seconds_remaining"`
}

// TimerAt uses half-open phases: focus [0,25m), break [25m,30m). A
// participant joining late observes the room's original epoch; joining does
// not start a new cycle. Durations are elapsed time, unaffected by DST.
func TimerAt(epoch, now time.Time) (Timer, error) {
	if epoch.IsZero() || now.IsZero() {
		return Timer{}, ErrInvalidInput
	}
	epoch, now = epoch.UTC(), now.UTC()
	elapsed := now.Sub(epoch)
	if !epoch.Add(elapsed).Equal(now) {
		return Timer{}, ErrInvalidInput
	}
	timer := Timer{ServerTime: now}
	if now.Before(epoch) {
		timer.Phase = Waiting
		timer.StartedAt = epoch
		timer.EndsAt = epoch
	} else {
		timer.Cycle = int64(elapsed/cycleDuration) + 1
		timer.StartedAt = epoch.Add(elapsed / cycleDuration * cycleDuration)
		if elapsed%cycleDuration < FocusDuration {
			timer.Phase = Focusing
			timer.EndsAt = timer.StartedAt.Add(FocusDuration)
		} else {
			timer.Phase = Break
			timer.StartedAt = timer.StartedAt.Add(FocusDuration)
			timer.EndsAt = timer.StartedAt.Add(BreakDuration)
		}
	}
	remaining := timer.EndsAt.Sub(now)
	timer.SecondsRemaining = int64(remaining / time.Second)
	if remaining%time.Second > 0 {
		timer.SecondsRemaining++
	}
	return timer, nil
}

type ParticipantInput struct {
	UserID              int64
	DisplayName, Target string
}
type Participant struct {
	UserID      int64     `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Target      string    `json:"target"`
	JoinedAt    time.Time `json:"joined_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type Snapshot struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Epoch        time.Time     `json:"epoch"`
	Timer        Timer         `json:"timer"`
	Participants []Participant `json:"participants"`
}

// Config is trusted server configuration, not client input. Zero values select
// bounded defaults. Clock is called while the store is locked; it must return a
// nonzero server instant and must not call back into the store.
type Config struct {
	MaxRooms, MaxParticipants int
	PresenceTTL, IdleTTL      time.Duration
	Clock                     func() time.Time
}

type room struct {
	id, title           string
	epoch, lastActivity time.Time
	participants        map[int64]Participant
}
type Store struct {
	mu     sync.Mutex
	config Config
	rooms  map[string]*room
}

func NewStore(config Config) (*Store, error) {
	if config.MaxRooms == 0 {
		config.MaxRooms = 100
	}
	if config.MaxParticipants == 0 {
		config.MaxParticipants = 50
	}
	if config.PresenceTTL == 0 {
		config.PresenceTTL = 90 * time.Second
	}
	if config.IdleTTL == 0 {
		config.IdleTTL = 30 * time.Minute
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}
	if config.MaxRooms < 1 || config.MaxParticipants < 1 || config.PresenceTTL < time.Second || config.IdleTTL < config.PresenceTTL {
		return nil, ErrInvalidInput
	}
	return &Store{config: config, rooms: make(map[string]*room)}, nil
}

func cleanText(value string, max int) (string, bool) {
	value = strings.TrimSpace(value)
	return value, value != "" && utf8.ValidString(value) && utf8.RuneCountInString(value) <= max
}
func cleanParticipant(input ParticipantInput) (ParticipantInput, error) {
	var ok bool
	input.DisplayName, ok = cleanText(input.DisplayName, 80)
	if !ok {
		return input, ErrInvalidInput
	}
	input.Target, ok = cleanText(input.Target, 160)
	if !ok || input.UserID <= 0 {
		return input, ErrInvalidInput
	}
	return input, nil
}

// Create starts the room's one shared epoch and joins its creator. Callers must
// derive the participant identity from an authenticated session/profile.
func (s *Store) Create(title string, creator ParticipantInput) (Snapshot, error) {
	title, ok := cleanText(title, 80)
	if !ok {
		return Snapshot{}, ErrInvalidInput
	}
	creator, err := cleanParticipant(creator)
	if err != nil {
		return Snapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.config.Clock().UTC()
	if now.IsZero() {
		return Snapshot{}, ErrInvalidInput
	}
	s.prune(now)
	if len(s.rooms) >= s.config.MaxRooms {
		return Snapshot{}, ErrCapacity
	}
	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		return Snapshot{}, err
	}
	id := hex.EncodeToString(randomID[:])
	r := &room{id: id, title: title, epoch: now, lastActivity: now, participants: make(map[int64]Participant)}
	r.participants[creator.UserID] = Participant{UserID: creator.UserID, DisplayName: creator.DisplayName, Target: creator.Target, JoinedAt: now, LastSeenAt: now}
	s.rooms[id] = r
	return snapshot(r, now)
}

// Join is idempotent per room/user. Rejoining updates profile/target and presence
// without resetting JoinedAt or the room clock, even when the room is full.
func (s *Store) Join(id string, input ParticipantInput) (Snapshot, error) {
	input, err := cleanParticipant(input)
	if err != nil {
		return Snapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.config.Clock().UTC()
	s.prune(now)
	r, ok := s.rooms[id]
	if !ok {
		return Snapshot{}, ErrRoomNotFound
	}
	participant, exists := r.participants[input.UserID]
	if !exists && len(r.participants) >= s.config.MaxParticipants {
		return Snapshot{}, ErrRoomFull
	}
	if !exists {
		participant.JoinedAt = now
	}
	participant.UserID = input.UserID
	participant.DisplayName = input.DisplayName
	participant.Target = input.Target
	participant.LastSeenAt = now
	r.participants[input.UserID] = participant
	r.lastActivity = now
	return snapshot(r, now)
}

// Heartbeat refreshes a current member's presence and target. It cannot join an
// expired participant implicitly; the caller must explicitly Join again.
func (s *Store) Heartbeat(id string, userID int64, target string) (Snapshot, error) {
	target, ok := cleanText(target, 160)
	if !ok || userID <= 0 {
		return Snapshot{}, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.config.Clock().UTC()
	s.prune(now)
	r, ok := s.rooms[id]
	if !ok {
		return Snapshot{}, ErrRoomNotFound
	}
	participant, ok := r.participants[userID]
	if !ok {
		return Snapshot{}, ErrNotMember
	}
	participant.Target = target
	participant.LastSeenAt = now
	r.participants[userID] = participant
	r.lastActivity = now
	return snapshot(r, now)
}

func (s *Store) Leave(id string, userID int64) error {
	if userID <= 0 {
		return ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.config.Clock().UTC()
	s.prune(now)
	r, ok := s.rooms[id]
	if !ok {
		return ErrRoomNotFound
	}
	if _, ok := r.participants[userID]; !ok {
		return ErrNotMember
	}
	delete(r.participants, userID)
	r.lastActivity = now
	return nil
}

// Get and List return detached, deterministically ordered snapshots. Mutating a
// returned slice cannot modify roster state. Membership is not an authorization
// policy: a future transport must authorize access before calling these methods.
func (s *Store) Get(id string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.config.Clock().UTC()
	s.prune(now)
	r, ok := s.rooms[id]
	if !ok {
		return Snapshot{}, ErrRoomNotFound
	}
	return snapshot(r, now)
}
func (s *Store) List() ([]Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.config.Clock().UTC()
	s.prune(now)
	result := make([]Snapshot, 0, len(s.rooms))
	for _, r := range s.rooms {
		item, err := snapshot(r, now)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Expiration is lazy on every operation; no goroutines or timers are left behind.
// Pruning stale membership never resets the shared Pomodoro epoch.
func (s *Store) prune(now time.Time) {
	for id, r := range s.rooms {
		for userID, participant := range r.participants {
			if now.Sub(participant.LastSeenAt) >= s.config.PresenceTTL {
				delete(r.participants, userID)
			}
		}
		if len(r.participants) == 0 && now.Sub(r.lastActivity) >= s.config.IdleTTL {
			delete(s.rooms, id)
		}
	}
}
func snapshot(r *room, now time.Time) (Snapshot, error) {
	timer, err := TimerAt(r.epoch, now)
	if err != nil {
		return Snapshot{}, err
	}
	result := Snapshot{ID: r.id, Title: r.title, Epoch: r.epoch, Timer: timer, Participants: make([]Participant, 0, len(r.participants))}
	for _, participant := range r.participants {
		result.Participants = append(result.Participants, participant)
	}
	sort.Slice(result.Participants, func(i, j int) bool { return result.Participants[i].UserID < result.Participants[j].UserID })
	return result, nil
}
