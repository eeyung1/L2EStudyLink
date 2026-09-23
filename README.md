# L2EStudyLink

A peer tutoring / study-session booking platform (part of the Learn2Earn ecosystem) where students and tutors list skills and availability, search each other, book sessions, and track personal study timetables and reflections.

## Tech Stack

- **Language**: Go, using the **Gin** web framework (`github.com/gin-gonic/gin`)
- **Pattern**: hybrid app — server-rendered HTML pages via `router.LoadHTMLGlob("templates/*.html")` for the UI shell, plus a JSON REST API under `/api/v1/*` that page-level JavaScript calls via `fetch()`. Pages are "dumb" wrappers; real logic lives behind the API.
- **Auth**: JWT (`github.com/golang-jwt/jwt/v5`), issued on login, sent as `Authorization: Bearer <token>`, verified by `middleware.AuthRequired`. Token is stored in the browser via `localStorage`.
- **Database**: PostgreSQL in production, SQLite schema is maintained for local use (`schema.sql` / `schema.sqlite`); the current `db/postgres.go` connection code uses PostgreSQL.
- **Frontend**: plain HTML, Tailwind CDN on signed-in pages, shared CSS for navigation and account pages, and vanilla JavaScript `fetch()` calls. No build step, no npm, no React/Vue. Dark mode uses a `localStorage` flag and manual style overrides.
- **Hosting**: Render (`render.yaml`, `Procfile`, `start.sh`), also has a `Dockerfile`.
- **Email**: Brevo transactional API (`email/brevo.go`). **Notifications**: Discord webhook (`notifications/discord.go`).

## Architecture

- `main.go` — registers every route directly (page routes and API routes both live here, no separate routes file).
- `handlers/` — one file per feature area (`auth.go`, `booking.go`, `search.go`, `profile.go`, `availability.go`, `timetable.go`, `admin.go`, `password_reset.go`). Handlers read `db := c.MustGet("db").(*sql.DB)` and write raw SQL inline — no repository layer.
- `middleware/` — `AuthRequired` (JWT verification) and `AdminRequired` (checks the current database role and suspension status).
- `config/` — `JWTSecret()`, read from the `JWT_SECRET` env var; the app refuses to start if it's unset or still the old placeholder value.
- `templates/` — one `.html` file per page, with inline `<script>` blocks calling the JSON API. `static/js/read-api.js` handles retries for timetable and reflections GET requests; `static/js/mobile-nav.js` controls the signed-in mobile menu. Account pages share `static/css/login.css`.
- `db/postgres.go` — connection setup only.
- `email/`, `notifications/` — Brevo email and Discord webhook integrations.
- `schema.sql` / `schema.sqlite` — hand-written schema, no migration tool; Postgres and SQLite dialects kept in sync. Existing PostgreSQL databases automatically create the missing `password_reset_tokens` table and index at startup. Other existing schema changes still need manual migration (see Known Issues).

## Features Implemented So Far

### Auth
- Signup (`POST /api/v1/signup`) — bcrypt-hashed passwords, unique email enforced.
- Login (`POST /api/v1/login`) — issues a 24-hour JWT.
- Self-service password reset (`POST /api/v1/forgot-password`, `POST /api/v1/reset-password`) — six-digit email code, 10-minute expiry, five attempts, one-minute resend cooldown and single-use reset. The code is stored as a keyed digest; the request endpoint gives the same response for known and unknown accounts. Successful reset signs the user in and opens their dashboard.
- Password codes and booking emails use Brevo. Existing PostgreSQL databases create the password-reset table and index on startup. The project owner confirmed the live reset flow works after merging PR #11; this is user verification, not an automated end-to-end test.
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
Login, signup, forgot/reset password, dashboard, search, my-bookings, timetable, reflections, admin — HTML pages with vanilla JS, with a generic `/page/:name` route for simpler pages. Login, signup and recovery pages share the blue account layout. Six signed-in pages have a compact mobile header and menu, plus narrower card, form, modal and table layouts. The brand links to the dashboard on signed-in pages and login on account pages. These responsive changes were merged, but a device-level usability audit is still needed.

---

## AI Planner — Delivery Plan

**Status:** planned; no planner routes, AI client, planner storage, or planner page exist yet. The reference `L2E_PLANNER_PROMPT_v2.md` is mentioned in earlier project notes but is not in this repository. Obtain and review it before implementing its detailed feedback format. The first release can use the existing timetable and reflections without waiting for automated scheduling.

**First useful release:** an authenticated user chooses a past week and taps **Analyze my week**. The page shows a factual activity summary and concise coaching based only on that user's timetable and reflections, with a link to prior analyses. It distinguishes scheduled time, reflected activity, and missing reflections; a reflection's existence alone does not prove the full block was completed.

1. **Define the output and metrics before calling a model.** Specify one user-owned week, timezone, empty-week behavior, and a versioned JSON response: scheduled blocks/minutes, reflected blocks, coverage percentage, factual highlights, suggested priorities, and one follow-up question. Compute counts and scheduled minutes in Go; do not infer actual hours worked or adherence from free-text reflections. Add tests for week boundaries, missing logs, and another user's data.
2. **Prepare per-user context.** Read only the authenticated user's timetable, reflections, and selected prior summaries. Bound the date range and text size; treat reflection text as untrusted input and never allow it to override system instructions. Write a generic coaching prompt after obtaining the reference document. Keep the model responsible for qualitative feedback, not arithmetic.
3. **Choose a provider and build a narrow client.** Keep the server-side key in an environment variable and never expose it to page JavaScript. Configure timeouts, output size, retries for transient failures, schema validation, and a useful failure response. Do not make an AI key mandatory for existing app startup while the planner is optional. Test with a fake HTTP provider before using a real key.
4. **Persist and expose analyses.** Add a versioned PostgreSQL migration for planner analyses, plus the matching SQLite schema. Store user ID, week, deterministic metrics, validated model response, prompt/model version, and timestamps. Add `POST /api/v1/planner/analyze` and `GET /api/v1/planner/history`, scoped to the logged-in user; prevent duplicate submissions for the same week and set a per-user usage limit.
5. **Build the phone-first planner page.** Render numbers and feedback from validated JSON as text, never model HTML. Show loading, empty, error, retry, and history states. Ensure keyboard access, readable charts/tables on narrow screens, and a clear distinction between observed activity and AI suggestions.
6. **Pilot before expanding.** Compare generated summaries against sample weeks, check privacy boundaries and costs, and collect feedback from a small group. Only then consider recurring flags, daily views, scheduled reports, or an interactive follow-up conversation.

**Decision for the first release:** on-demand weekly analyses using each user's own schedule. Define the user's timezone and whether they want a weekly goal before adding adherence or streak scores. Provider, budget, and the absent reference prompt still need product decisions; they do not block the deterministic metrics prototype.

## Mobile UX — Delivery Plan

**Status:** the first implementation pass is merged. The code audit below covers the core phone journeys; actual Android/iPhone viewport and assistive-technology acceptance remains to be done. Continue with the existing Go, HTML, CSS, and vanilla JS stack and review further changes on separate branches.

### Code audit and implemented changes (2026-09-23)

| Journey | Finding and merged improvement | Review |
| --- | --- | --- |
| Signed-in navigation | The full-screen menu let keyboard focus escape behind it. Tab now cycles inside; Escape returns focus to Menu. | PR #13 |
| Tutor search | Narrow cards squeezed details beside the booking action. Details now stack above a full-width action. | PR #14 |
| Tutor booking | The meeting type used a browser popup, and changing the date left stale slots. The form now has an explicit selector and refreshes/blocks slots for the selected date. | PRs #15, #18 |
| Timetable and reflections entry | Placeholder-only fields and small actions made phone entry hard. Persistent labels and larger controls were added. | PR #16 |
| My bookings | Action buttons were small, long content crowded cards, and loading failures had no retry. Cards wrap, actions grow, and the page offers Try again. | PR #17 |
| Account pages | Shared form cards could exceed narrow widths and 14px fields could trigger iPhone focus zoom. Card sizing and phone input text were corrected. | PR #19 |
| Admin | Wide tables needed an explicit contained scroll region and larger row actions. Both tables now provide a swipe hint and keyboard focus. | PR #20 |
| Timetable navigation | Seven day sections required excessive scrolling. Phone shortcuts jump to each day and highlight today. | PR #21 |
| Reflection history | A long list had no way to find a date. A local date filter and Show all action were added. | PR #22 |

**Verification completed:** JavaScript syntax and diff checks for affected changes, targeted checks for booking date/slot transitions and reflection filtering, and the project owner's live phone review through PR #16. No Go handlers were changed in this pass. The account forms, admin page, and changes in PRs #17–#22 still need the final live device review.

**Final acceptance on the deployed site:** check 320px and common phone widths on Android and iPhone where available. Open/close Menu with touch and keyboard; search, change a tutor's booking date, select a meeting type, and submit a test booking; inspect booking actions; add a block and reflection; jump across timetable days; filter and clear reflection history; use login, signup and reset fields; scroll admin tables without moving the whole page. Confirm no horizontal page overflow, clipped modal buttons, keyboard-covered actions, or unreadable light/dark text. Do not submit admin deletion during this audit. Record any device-specific defect as a new focused task.

1. **Audit real tasks at phone widths.** Check signup, login, OTP reset, search and tutor profile, booking and cancellation, timetable creation, reflections, and admin (for admins) at 320, 375, 390, and 768 CSS pixels. Record horizontal overflow, clipped controls, keyboard overlap, slow states, and confusing navigation. Test at least one actual phone, including a slow network, before calling the release mobile ready.
2. **Create a consistent interaction system.** Reuse the existing shared styles for type, spacing, buttons, fields, focus indicators, error and success messages, and touch targets. Keep navigation reachable with one hand; preserve desktop behavior. Ensure menu focus and Escape behavior, visible labels, accessible modal focus, and reduced-motion support.
3. **Improve core journeys in small PRs.** First make search results, tutor details, and booking actions easy to scan and tap. Then make timetable and reflections quick to enter and review by date. Show clear loading, empty, validation, success, and retry states. Use mobile cards or scoped horizontal scrolling where data tables cannot fit; never make the whole page scroll sideways.
4. **Verify each journey.** Test a 320px viewport, keyboard-only navigation, screen reader labels, zoom to 200%, light/dark contrast, and an Android and iPhone browser where available. Measure real task completion and perceived speed with users before considering an installable PWA. A PWA is an optional later milestone, not a prerequisite for phone access through the browser.

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

1. Run the final live phone acceptance above for PRs #17–#22 and capture any device-specific defect in a focused branch. Confirm reset with a fresh code because shared account CSS changed in PR #19.
2. Investigate any recurring timetable/reflections 500 using the server's query logs; add focused handler tests and run `go test ./...` in a Go-enabled environment.
3. Build and test deterministic weekly metrics from the current user’s data; document the week/timezone and missing-reflection rules.
4. Obtain the planner reference prompt, choose a provider and budget, then deliver the scoped weekly planner API and mobile page in reviewable steps above.
5. Add booking completion and reviews. Address JWT storage and login rate limiting before broad rollout.

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
- `LLM_API_KEY` *(planned — not yet used in code)* — server-side only for the optional AI planner once a provider is chosen.

### Useful commands

```bash
go mod download
go mod get <package_url>
```
