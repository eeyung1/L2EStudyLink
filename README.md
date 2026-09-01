# L2EStudyLink

A peer tutoring / study-session booking platform (part of the Learn2Earn ecosystem) where students and tutors list skills and availability, search each other, book sessions, and track personal study timetables and reflections.

## Tech Stack

- **Language**: Go, using the **Gin** web framework (`github.com/gin-gonic/gin`)
- **Pattern**: hybrid app — server-rendered HTML pages via `router.LoadHTMLGlob("templates/*.html")` for the UI shell, plus a JSON REST API under `/api/v1/*` that page-level JavaScript calls via `fetch()`. Pages are "dumb" wrappers; real logic lives behind the API.
- **Auth**: JWT (`github.com/golang-jwt/jwt/v5`), issued on login, sent as `Authorization: Bearer <token>`, verified by `middleware.AuthRequired`. Token is stored in the browser via `localStorage`.
- **Database**: PostgreSQL in production, SQLite intended for local dev (`schema.sql` / `schema.sqlite` — see Known Issues, they're currently out of sync with the code).
- **Frontend**: plain HTML + Tailwind CDN + vanilla JavaScript `fetch()` calls. No build step, no npm, no React/Vue. Dark mode via a `localStorage` flag toggling a `dark` class on `<html>`, with manual per-utility-class overrides (not Tailwind's built-in `dark:` variants).
- **Hosting**: Render (`render.yaml`, `Procfile`, `start.sh`), also has a `Dockerfile`.
- **Email**: Resend (`email/resend.go`). **Notifications**: Discord webhook (`notifications/discord.go`).

## Architecture

- `main.go` — registers every route directly (page routes and API routes both live here, no separate routes file).
- `handlers/` — one file per feature area (`auth.go`, `booking.go`, `search.go`, `profile.go`, `availability.go`, `timetable.go`, `admin.go`, `password_reset.go`). Handlers read `db := c.MustGet("db").(*sql.DB)` and write raw SQL inline — no repository layer.
- `middleware/` — `AuthRequired` (JWT verification).
- `config/` — `JWTSecret()`, read from the `JWT_SECRET` env var; the app refuses to start if it's unset or still the old placeholder value.
- `templates/` — one `.html` file per page, Tailwind CDN + inline `<script>` blocks calling the JSON API.
- `db/postgres.go` — connection setup only.
- `email/`, `notifications/` — Resend email and Discord webhook integrations.
- `schema.sql` / `schema.sqlite` — hand-written schema, no migration tool.

## Features Implemented So Far

### Auth
- Signup (`POST /api/v1/signup`) — bcrypt-hashed passwords, unique email enforced.
- Login (`POST /api/v1/login`) — issues a 24-hour JWT.
- Self-service password reset (`POST /api/v1/forgot-password`, `POST /api/v1/reset-password`) — random 32-byte token, 1-hour expiry, single-use, generic response regardless of whether the email exists (no account enumeration), reset link emailed via Resend.
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
- Personal weekly timetable: add/list/delete time blocks (`GET`/`POST /api/v1/timetable`, `DELETE /api/v1/timetable/:id`).
- Reflections tied to a timetable block and date, upsert-style (creating a reflection for an existing block+date updates it) (`POST`/`GET /api/v1/reflections`).

### Admin
- Stats: total users, bookings, skills (`GET /api/v1/admin/stats`).
- List all users with session/rating/suspension info (`GET /api/v1/admin/users`).
- Delete a user (`DELETE /api/v1/admin/users/:id`).
- Suspend/unsuspend a user (`PUT /api/v1/admin/users/:id/suspend`).
- List all bookings across the platform (`GET /api/v1/admin/bookings`).
- Reset the hardcoded admin account's password (`POST /api/v1/admin/reset-password`).

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
   - Timetable blocks for the period (`timetable` table — see Known Issues, this table isn't in the schema yet).
   - Reflections logged against those blocks (`activity_logs` — also not yet in the schema).
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
- Cost/rate-limiting: this is now the first paid third-party AI dependency in the app (alongside Resend and Discord, which are cheap/free) — needs its own error handling and probably a per-user request cap.

**Reference document:** `L2E_PLANNER_PROMPT_v2.md` (included alongside this README) is the source-of-truth for the planner's tone, evaluation format, and flag list — treat it as the spec to genericize, not something to paste in verbatim per-user.

---

## Known Issues / Gaps

Ranked roughly by how much they'd block real usage:

1. **Schema drift — `timetable` and `activity_logs` tables are missing from both `schema.sql` and `schema.sqlite`.** `handlers/timetable.go` and `seed_timetable.sql` query these tables, but running either schema file fresh will make `/api/v1/timetable` and `/api/v1/reflections` fail at runtime with "relation does not exist." These need to be added to both schema files (and kept in sync — Postgres and SQLite currently diverge on syntax, e.g. `TEXT[]` vs `TEXT`, `SERIAL` vs `AUTOINCREMENT`). **This also blocks the AI Planner work above**, since the planner reads directly from these tables.
2. **`is_admin` column is missing from both schema files.** `handlers/admin.go` (`AdminUsers`) and `handlers/profile.go` (`GetProfile`) both select `is_admin` from `users`, but no schema file defines that column. This will also fail at runtime on a fresh database.
3. **Reviews table exists in schema but has no handler or route at all.** `rating` and `total_reviews` are displayed everywhere (profile, dashboard, search results) but there's no way for a user to actually submit a review after a session — those numbers can never move past their defaults. This is the biggest non-AI "next feature" candidate.
4. **JWT is stored in `localStorage`**, not an httpOnly cookie — vulnerable to token theft via any XSS elsewhere on the site.
5. **No rate limiting or lockout on `/api/v1/login`** — unlimited password guesses are possible against any account.
6. **`ResetAdminPassword` hardcodes a specific email and a fixed new password (`admin123`)** — fine as a one-off recovery tool during early development, but shouldn't stay reachable once real users are on the platform; it currently only requires being logged in (`AuthRequired`), not an admin check.
7. **No booking completion / no-show flow.** `bookings.status` supports `completed` and `no_show` in the schema, and `users.no_show_count` exists, but nothing in the handlers ever sets a booking to either state — sessions stay `confirmed` forever, and no-show tracking is unused.
8. **Admin endpoints don't check `is_admin`.** Every `/api/v1/admin/*` route only requires `AuthRequired` (any logged-in user), not an admin check — any authenticated user can currently hit the admin stats, user list, delete-user, suspend, and bookings endpoints.

## Suggested Next Steps (in priority order)

1. Add `timetable`, `activity_logs`, and an `is_admin` column to both `schema.sql` and `schema.sqlite`, and reconcile the two files so they define the same tables/columns going forward — this unblocks both the existing timetable/reflections feature *and* the AI Planner work.
2. Genericize `L2E_PLANNER_PROMPT_v2.md` into a persona/format template + per-user data interpolation, per the design notes above.
3. Build the `ai/` client package, `handlers/planner.go`, and the `planner_sessions` table; wire up `POST /api/v1/planner/analyze`.
4. Build the planner frontend page rendering the structured dashboard from the API's JSON.
5. Add an actual admin-check middleware (or inline check) to every `/api/v1/admin/*` route.
6. Build the reviews feature: submit a review after a completed booking, update `users.rating`/`total_reviews` accordingly.
7. Build a way to mark a booking `completed` (and optionally `no_show`), since nothing currently transitions a booking out of `confirmed`.
8. Move the JWT out of `localStorage` into an httpOnly cookie.
9. Add basic rate limiting to `/api/v1/login`.
10. Retire or lock down `ResetAdminPassword` behind a real admin check before this goes further into production use.

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
- `RESEND_API_KEY` — required for password-reset and booking emails to send.
- `DISCORD_WEBHOOK_URL` — optional; notifications are skipped (logged, not sent) if unset.
- `APP_BASE_URL` — used to build the password-reset link; falls back to `http://localhost:8080`.
- `LLM_API_KEY` *(planned — not yet used in code)* — will be required once the AI Planner integration lands.

### Useful commands

```bash
go mod download
go mod get <package_url>
```
