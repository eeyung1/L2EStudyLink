# Coordination

## Project Context
L2EStudyLink is a peer tutoring and study-session booking platform in the Learn2Earn ecosystem, built with Go (Gin), PostgreSQL (with SQLite parity for local dev), server-rendered HTML templates with vanilla JS and Tailwind CDN.

## Standing Permissions
- Autonomous development in isolated Git worktrees under `.agent-worktrees/<task-id>`.
- Read and run local unit tests with `/home/recoding/sdk/go/bin/go test ./...` and `node --test` / `node --check`.
- Do not introduce external frameworks (e.g. React/Vue/npm build step) or change the Gin handler inline SQL architecture.

## Owner Decisions
- **2026-10-27**: Add Reflection UI standard must match **OPay UI standard**:
  - Segmented pill switcher: `[ ✍️ Write Reflection ]` vs. `[ 🎙️ Speak Reflection ]`
  - High-contrast OPay emerald green accent (`#00B875` / `#059669`) with dark mode support.
  - Tactile record button with pulsing ripple/waveform animation, recording timer (`00:00 / 03:00 max`).
  - In-modal instant playback preview with scrubber, re-record, and delete controls before saving.
  - Automatic submission to `/api/v1/voice/upload` and link persistence to `activity_logs.audio_url`.
- **2026-10-27**: Reflections must be decoupled from timetable cascade deletions (`ON DELETE SET NULL`) with `activity_name` snapshotting.
- **2026-10-27**: Maintain strict single-task isolated worktrees (`ONE TASK -> ONE OWNER -> ONE GENERATION -> ONE CLAIM -> ONE BRANCH -> ONE ISOLATED WORKTREE -> ONE BOUNDED SCOPE`).

## Blockers and Proposed Decisions
- None currently active.

## Automated Actions
- Task assignment and claim locking managed via `multi-agent-orchestrator`.
