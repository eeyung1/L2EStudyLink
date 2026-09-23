# L2EStudyLink

A peer tutoring / study-session booking platform (part of the Learn2Earn ecosystem) where students and tutors list skills and availability, search each other, book sessions, and track personal study timetables and reflections.

## Tech Stack

- **Language**: Go, using the **Gin** web framework (`github.com/gin-gonic/gin`)
- **Pattern**: hybrid app — server-rendered HTML pages via `router.LoadHTMLGlob("templates/*.html")` for the UI shell, plus a JSON REST API under `/api/v1/*` that page-level JavaScript calls via `fetch()`. Pages are "dumb" wrappers; real logic lives behind the API.
- **Auth**: JWT (`github.com/golang-jwt/jwt/v5`), issued on login, sent as `Authorization: Bearer <token>`, verified by `middleware.AuthRequired`. Token is stored in the browser via `localStorage`.
- **Database**: PostgreSQL in production, SQLite schema is maintained for local use (`schema.sql` / `schema.sqlite`); the current `db/postgres.go` connection code uses PostgreSQL.
- **Frontend**: plain HTML + Tailwind CDN + vanilla JavaScript `fetch()` calls. No build step, no npm, no React/Vue. Dark mode via a `localStorage` flag toggling a `dark` class on `<html>`, with manual per-utility-class overrides (not Tailwind's built-in `dark:` variants).
- **Hosting**: Render (`render.yaml`, `Procfile`, `start.sh`), also has a `Dockerfile`.
- **Email**: Brevo transactional API (`email/brevo.go`). **Notifications**: Discord webhook (`notifications/discord.go`).

## Architecture

- `main.go` — registers every route directly (page routes and API routes both live here, no separate routes file).
- `handlers/` — one file per feature area (`auth.go`, `booking.go`, `search.go`, `profile.go`, `availability.go`, `timetable.go`, `admin.go`, `password_reset.go`). Handlers read `db := c.MustGet("db").(*sql.DB)` and write raw SQL inline — no repository layer.
- `middleware/` — `AuthRequired` (JWT verification) and `AdminRequired` (checks the current database role and suspension status).
- `config/` — `JWTSecret()`, read from the `JWT_SECRET` env var; the app refuses to start if it's unset or still the old placeholder value.
- `templates/` — one `.html` file per page, Tailwind CDN + inline `<script>` blocks calling the JSON API. `static/js/read-api.js` handles safe retries for timetable and reflections GET requests.
- `db/postgres.go` — connection setup only.
- `email/`, `notifications/` — Brevo email and Discord webhook integrations.
- `schema.sql` / `schema.sqlite` — hand-written schema, no migration tool; Postgres and SQLite dialects kept in sync. Fresh databases created from either file are complete; existing databases need manual migration (see Known Issues).

## Features Implemented So Far

### Auth
- Signup (`POST /api/v1/signup`) — bcrypt-hashed passwords, unique email enforced.
- Login (`POST /api/v1/login`) — issues a 24-hour JWT.
- Self-service password reset (`POST /api/v1/forgot-password`, `POST /api/v1/reset-password`) — six-digit email code, 10-minute expiry, five attempts, one-minute resend cooldown and single-use reset. The code is stored as a keyed digest; the request endpoint gives the same response for known and unknown accounts. Successful reset signs the user in and opens their dashboard.
- JWT secret is loaded from an environment variable and validated at startup (fails fast if missing or left as the old placeholder).

### Profile
- View own profile (`GET /api/v1/me`) — includes skills list, bio, Discord username, rating, review/session counts.
- Update profile (`PUT /api/v1/profile`) — bio, Discord username.
- Add/remove skills with proficiency level (`POST /api/v1/skills`, `DELETE /api/v1/skills/:skill`).

### Availability
- Get/set weekly availability slots (`GET`/`PUT /api/v1/availability`), keyed by day-of-week + time range.

### Search
- Search tutors by skill (`GET /api/v1/search`).
- Search tutors by skill with availability embedded in one query, deduplicated and grouped in Go to avoid N+1 queries (`GET /api/v1/search-with-availability`).
- View a specific tutor's public profile, skills, and availability (`GET /api/v1/tutors/:id`).

### Bookings
- Create a booking (`POST /api/v1/bookings`) — blocks self-booking and double-booking the same tutor slot; triggers a Discord notification (with @mentions if Discord usernames are set) and emails to both parties.
- List my bookings, as either tutor or student (`GET /api/v1/bookings`).
- Cancel a booking (`DELETE /api/v1/bookings/:id`) — enforces a 2-hour cancellation cutoff before the session; notifies both parties.
- Tutor confirms or cancels a booking (`PUT /api/v1/bookings/:id/status`) — only the tutor can change status; notifies both parties.

### Timetable & Reflections
- Personal weekly timetable: add/list/delete time blocks (`GET`/`POST /api/v1/timetable`, `DELETE /api/v1/timetable/:id`). The read handler returns `HH:MM` times and an empty JSON array when there are no blocks.
- Reflections tied to a timetable block and date, upsert-style (creating a reflection for an existing block+date updates it) (`POST`/`GET /api/v1/reflections`). The read handler returns `YYYY-MM-DD` dates, `HH:MM` times, and an empty JSON array when there are no reflections.
- Both pages retry transient network or server failures on GET requests up to three attempts, redirect to login on an expired session, and show a **Try again** control for errors. The timetable remains viewable when its separate reflections request fails. Database query and scan failures are logged on the server rather than silently dropping rows.

### Admin
- Every `/api/v1/admin/*` API route requires authentication and a current database check that `is_admin` is true and `is_suspended` is false. An old JWT does not preserve admin access after a role change or suspension.
- Stats: total users, bookings, skills (`GET /api/v1/admin/stats`).
- List all users with session/rating/suspension info (`GET /api/v1/admin/users`).
- Delete a user (`DELETE /api/v1/admin/users/:id`).
- Suspend/unsuspend a user (`PUT /api/v1/admin/users/:id/suspend`).
- List all bookings across the platform (`GET /api/v1/admin/bookings`).
- The former hardcoded admin password reset route and handler have been removed. `GET /api/v1/admin/users` scans PostgreSQL boolean fields as booleans.

### Pages
Login, signup, forgot/reset password, dashboard, search, my-bookings, timetable, reflections, admin — all server-rendered Tailwind + vanilla JS pages, generic `/page/:name` route for the simpler static ones.

---

## AI Planner Integration — Current Work

**Goal:** give every L2EStudyLink user an in-app AI planner that analyzes their own timetable + reflections (daily or weekly) and produces the kind of structured evaluation described in `L2E_PLANNER_PROMPT_v2.md` — currently a manual workflow (paste the prompt + reflections into an AI model by hand) — natively inside the platform, for all fellows, not just one user.

### What the reference prompt (`L2E_PLANNER_PROMPT_v2.md`) defines

- A planner persona ("30 years of experience," direct/no-flattery tone) that evaluates a user's week against their own history.
- A fixed five-step evaluation format: acknowledge context → visual dashboard (metric cards, color-coded bar chart, week-over-week comparison table, milestones, alert boxes) → block-by-block analysis → five specific priorities for the coming week → one direct clarifying question.
- Continuity across sessions — the planner is expected to "remember" prior weeks' metrics and flags (a running metrics history table, recurring-pattern flags to watch).
- Currently hardcoded to one person (Eeyung) — his timetable blocks, project list, and metrics history are baked directly into the prompt text.

### Design implications for productizing this in L2EStudyLink

1. **Genericize the persona template.** The prompt as written is a personal document, not a reusable system prompt. It needs splitting into (a) a fixed planner-persona/format template (tone, five-step structure, dashboard spec) that's the same for every user, and (b) per-user context (name, program stage, tech stack, project list, metrics history) pulled from the database at request time instead of hardcoded.
2. **Data the planner needs, per user, per period (day or week):**
   - Timetable blocks for the period (`timetable` table — now defined in both schema files, see "Recently resolved" in Known Issues).
   - Reflections logged against those blocks (`activity_logs` — also now defined in both schema files).
   - Prior periods' metrics for the comparison table and streak tracking — this doesn't exist as stored data yet; hours/adherence per week would need to be computed from timetable + reflections, or stored as a rollup.
3. **New persistence needed:**
   - A `planner_sessions` (or similar) table to store each generated analysis — the text/dashboard payload, the period it covers, and enough of the computed metrics (total hours, daily average, adherence %) to feed next period's comparison table and streak checks, without re-deriving history from scratch every time.
   - Optionally a `planner_flags` table or JSON column for the recurring-pattern watchlist (e.g. "evening session floor," "protected rest window") so flags persist and get checked automatically rather than re-derived by the model each time from scratch.
4. **New handler + route(s):**
   - `handlers/planner.go` — e.g. `POST /api/v1/planner/analyze` with a `period: "daily" | "weekly"` and optional date range; reads the user's timetable + reflections for that period plus their stored planner history, calls the LLM, stores the result, returns it.
   - `GET /api/v1/planner/history` — past analyses, for the dashboard/timeline view.
5. **LLM integration:**
   - A small internal client package (e.g. `ai/client.go`) wrapping whichever provider is chosen (the fellow's own tooling already uses Groq elsewhere in the Learn2Earn ecosystem, per prior project notes — worth reusing if consistent) — needs its own API key env var (e.g. `GROQ_API_KEY` or `LLM_API_KEY`) following the same fail-fast-if-unset pattern as `JWT_SECRET`.
   - System prompt = the genericized persona template with per-user context interpolated in; user message = the timetable + reflections payload for the requested period.
6. **Frontend:** a planner page/tab (new template, e.g. `planner.html`) rendering the dashboard the prompt spec describes — metric cards, color-coded bar chart, comparison table, alert boxes — client-side from the structured JSON the API returns, rather than the LLM returning raw HTML directly (keeps rendering consistent and avoids trusting model-generated markup).

### Open questions to resolve before implementation starts

- Should the analysis be triggered on-demand by the user, or run automatically on a schedule (e.g. every Saturday, per the reference prompt's rhythm) via a background job?
- Does every user get the same fixed weekly/daily target and block structure, or is the timetable (and therefore what "adherence" means) fully user-defined, given `timetable` is already a per-user, freeform table? The reference prompt assumes a fixed personal schedule (07:00–08:00 Daily Study, etc.) — that won't generalize as-is to 770 fellows with different schedules.
- How much of the "metrics history" and "recurring flags" logic should be computed deterministically in Go before the prompt is built (hours logged, adherence %, streaks) versus left for the model to infer from raw reflection text? Deterministic computation will be more reliable for the dashboard numbers; the model is better suited to the qualitative block-by-block analysis and priorities.
- Cost/rate-limiting: this is now the first paid third-party AI dependency in the app (alongside Brevo and Discord) — needs its own error handling and probably a per-user request cap.

**Reference document:** `L2E_PLANNER_PROMPT_v2.md` is described here as the planner reference, but it is not currently committed in this repository. Obtain it before implementing the planner; do not treat the description above as the full specification.

---

## Known Issues / Gaps

Ranked roughly by how much they'd block real usage:

> **Recently resolved:** `timetable`, `activity_logs`, and `users.is_admin` are now defined in both `schema.sql` and `schema.sqlite` (kept in sync, dialect-correct). Fresh databases created from either file fully support `/api/v1/timetable`, `/api/v1/reflections`, `GetProfile`, and `AdminUsers` — this also unblocks the AI Planner work above. ⚠️ **Existing databases are not changed by these files:** `CREATE TABLE IF NOT EXISTS` adds the new tables but won't add `is_admin` to an existing `users` table. Apply it manually — Postgres: `ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT FALSE;` plus the two tables from `schema.sql`; SQLite: `ALTER TABLE users ADD COLUMN is_admin INTEGER DEFAULT 0;` plus the tables from `schema.sqlite`.

1. **Intermittent timetable/reflections API failures remain under investigation.** Render logs showed `GET /api/v1/reflections` returning 500 while the HTML page returned 200. The merged read retries improve recovery and the handlers now log database errors, but the underlying production database error has not yet been confirmed. Inspect Render application logs if it recurs.
2. **Reviews table exists in schema but has no handler or route.** `rating` and `total_reviews` are displayed, but users cannot submit a review.
3. **JWT is stored in `localStorage`**, not an httpOnly cookie — vulnerable to token theft via XSS.
4. **No rate limiting or lockout on `/api/v1/login`** — unlimited password guesses are possible.
5. **No booking completion / no-show flow.** The schema supports `completed` and `no_show`, but no handler transitions bookings to those states.

## Suggested Next Steps (in priority order)

1. Verify the merged timetable/reflections changes against the live app with an authenticated account; if a 500 recurs, read the new `GetReflections query` / `GetTimetable query` log entry and fix the specific database error.
2. Add tests for the timetable/reflections handlers and run `go test ./...` (the development workspace used for PR #4 did not have Go installed).
3. Obtain `L2E_PLANNER_PROMPT_v2.md`, then genericize it into a persona/format template plus per-user context.
4. Build the planner API, persistence, and frontend described above.
5. Build a booking completion flow and reviews after completed bookings.
6. Move the JWT out of `localStorage` into an httpOnly cookie.
7. Add basic rate limiting to `/api/v1/login`.

## Running Locally

```bash
go mod download
go run main.go   # or: go run .
```

Then visit `http://localhost:8080/`.

### Database

```bash
sudo service postgresql start
sudo service postgresql status   # confirm it's running
```

Required environment variables (see `.gitignore` — `.env` is not committed):
- `DATABASE_URL` — falls back to a local Postgres connection string if unset.
- `JWT_SECRET` — required; the app will not start without it.
- `BREVO_API_KEY` — Brevo API key; required for password-reset and booking emails.
- `BREVO_FROM_EMAIL` — a sender address you have added and verified in Brevo, such as an email account you control at Gmail. Verify it with the code Brevo emails to that address. Use only the email address here, without a display name.
- `BREVO_FROM_NAME` — optional sender display name, such as `L2EStudyLink`.

On Render, add the Brevo variables to the web service and confirm transactional email sending is activated in your Brevo account. Brevo's free tier currently includes up to 300 sends per day. If you use a free sender address, Brevo may replace the displayed sender with one of its own technical domains (transactional mail commonly uses `t-sender-sib.com`); `brevosend.com` is a Brevo-managed replacement domain, not an address you can choose yourself. Check delivery in Brevo's transactional logs and in the recipient's inbox or spam folder. A 201 API response means Brevo accepted the request, not that the recipient has received it. Existing reset codes issued before switching providers remain usable until they expire.
- `DISCORD_WEBHOOK_URL` — optional; notifications are skipped (logged, not sent) if unset.
- `LLM_API_KEY` *(planned — not yet used in code)* — will be required once the AI Planner integration lands.

### Useful commands

```bash
go mod download
go mod get <package_url>
```
