# Shared focus-room engine

This package is the foundation for the README's co-working room roadmap item.
It does not register `/focus` or add an API, page, database table, audio cue, or
reflection submission. Those are separate integration tasks.

Each room gets one server timestamp when created. Everyone in the room observes
that epoch: 25 minutes focusing, then 5 minutes on break, repeating continuously.
Joining or refreshing presence never restarts the clock. The exact phase boundary
belongs to the next phase; remaining seconds round up, including subsecond input.
Timers use elapsed time in UTC, including across daylight-saving changes.

```go
rooms, err := focus.NewStore(focus.Config{})
if err != nil { /* handle invalid server configuration */ }
room, err := rooms.Create("Quiet Go practice", focus.ParticipantInput{
    UserID: authenticatedUserID,
    DisplayName: profileName,
    Target: "Write one parser test",
})
// Handle err before using room.
// rooms.Join(room.ID, participant), rooms.Heartbeat(room.ID, userID, target),
// rooms.Get(room.ID), rooms.List(), and rooms.Leave(room.ID, userID)
// provide the transport-independent lifecycle.
```

The default limits are 100 rooms and 50 participants per room. Presence expires
at 90 seconds without a join/heartbeat. A transport should heartbeat around every
30 seconds while the user is present. Empty rooms expire after 30 minutes without
real participant activity. Expiration runs on normal operations; there are no
background goroutines. Reading a room does not prolong participant presence or
room life. Rejoining is idempotent per room/user and preserves the original join
time until that presence expires. A heartbeat cannot rejoin an expired user.

Room snapshots and participant slices are detached from mutable storage. Rooms
are ordered by ID, participants by user ID. A mutex protects lifecycle operations
and capacity checks together; the race tests exercise concurrent users.

Future HTTP/UI integration must:

- Obtain user ID and display name from the authenticated session/profile, never
  trust a caller-supplied identity, and authorize room/roster access.
- Render user-controlled room titles, names, and targets as text.
- Use `timer.server_time` and `timer.ends_at` to compensate for client clock skew,
  then refresh periodically rather than treating browser timer ticks as authority.
- Require an explicit reflection after a session; membership and elapsed room time
  do not prove that studying occurred and must not create verified study hours.
- Treat these rooms as process-local and ephemeral. Restart clears all rooms;
  multiple application instances require shared state or room affinity before
  calling the experience synchronized across instances. There is no durable
  participation history, cross-process quota, invitation/privacy policy, SSE
  transport, or scheduler here.

Run `go test -race ./focus` to validate timer and roster behavior without a database
or network service. `Config.Clock` is injectable for deterministic tests and must
supply a nonzero server time without calling back into the store.
