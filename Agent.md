# Agent.md — L2EStudyLink

You are assisting with development on **L2EStudyLink**, a peer tutoring /
study-session booking platform (part of the Learn2Earn ecosystem) where
students and tutors list skills and availability, search each other, book
sessions, and track personal study timetables/reflections.

Read this entire file before making any changes. If a request is ambiguous
or not covered here, ask before writing code — do not assume.

---

## TECH STACK (do not introduce a different one)

- **Language**: Go, using the **Gin** web framework (`github.com/gin-gonic/gin`)
  — unlike this developer's other projects, this one is NOT stdlib
  `net/http`. Don't mix routing styles.
- **Pattern**: hybrid app — server-rendered HTML pages via
  `router.LoadHTMLGlob("templates/*.html")` for the UI shell, PLUS a JSON
  REST API under `/api/v1/*` that the page-level JavaScript calls via
  `fetch()`. Pages are "dumb" wrappers; all real logic lives behind the API.
- **Auth**: JWT (`github.com/golang-jwt/jwt/v5`), issued on login, sent as
  `Authorization: Bearer <token>`, verified by `middleware.AuthRequired`.
  Token is stored in the browser via `localStorage` (see Known Issues —
  this is a security gap, not a pattern to replicate elsewhere).
- **Database**: PostgreSQL in production, SQLite for local dev
  (`schema.sql` / `schema.sqlite` — keep both in sync when changing schema,
  see Known Issues, they are currently NOT in sync).
- **Frontend**: plain HTML + Tailwind CDN (`<script src="https://cdn.tailwindcss.com">`)
  + vanilla JavaScript `fetch()` calls. No build step, no npm, no
  React/Vue. Dark mode is implemented via a `localStorage` flag toggling a
  `dark` class on `<html>`, with manual per-utility-class overrides in a
  `<style>` block on each page (not Tailwind's built-in dark mode config —
  don't assume `dark:` variants work, they don't here).
- **Hosting**: Render (`render.yaml`, `Procfile`, `start.sh`), also has a
  `Dockerfile`.
- **Email**: Resend (`email/resend.go`). **Notifications**: Discord webhook
  (`notifications/discord.go`).

---

## ARCHITECTURE PATTERN

- `main.go` — registers every route directly (no separate routes file like
  the Convention project). Page routes and API routes are both defined here.
- `handlers/` — one file per feature area (`auth.go`, `booking.go`,
  `search.go`, `profile.go`, `availability.go`, `timetable.go`, `admin.go`).
  Handlers read `db := c.MustGet("db").(*sql.DB)` and write raw SQL directly
  inline — there is no separate repository layer here (unlike the Convention
  project's `repository/` folder). Keep this project's existing convention:
  don't introduce a repository layer for one feature while every other
  handler still queries inline, unless explicitly asked to refactor the
  whole file.
- `middleware/` — `AuthRequired` (JWT verification).
- `templates/` — one `.html` file per page, Tailwind CDN + inline `<script>`
  blocks calling the JSON API. No shared layout partial system is actually
  used despite `layout.html` existing — check whether a given page actually
  extends it before assuming it does.
- `db/postgres.go` — connection setup only.
- `schema.sql` / `schema.sqlite` — hand-written schema, no migration tool.

---

## CURRENT STATE — WHAT'S BUILT (routes confirmed in `main.go`)

- **Auth**: signup, login (JWT issuance)
- **Profile**: view/update, add/remove skills
- **Availability**: get/set
- **Search**: search tutors, search tutors with availability, get tutor
  profile by ID
- **Bookings**: create, list mine, cancel, update status
- **Admin**: stats, list/delete/suspend users, list bookings, reset admin
  password
- **Timetable**: get/add/delete time blocks
- **Reflections**: add/get (tied to timetable entries via `activity_logs`)

---

## KNOWN ISSUES — CONFIRMED, NOT ASSUMPTIONS

Ranked by severity. Do not treat any of these as already fixed unless the
project owner confirms it.

1. **🚨 Hardcoded JWT signing secret** (`middleware/auth.go` AND
   `handlers/auth.go` both define `var jwtSecret = []byte("your-super-secret-key-change-this-later")`).
   This is committed directly in source, not read from an environment
   variable. Anyone with the code can forge a valid token for any
   `user_id`. **This is a live auth bypass and the highest priority fix in
   this codebase.** Fix means: move to an env var (`JWT_SECRET`), fail
   startup if it's unset or still the placeholder value, and update BOTH
   files (they currently duplicate the secret independently — deduplicate
   this too while fixing it).

2. **JWT stored in `localStorage`**, not an httpOnly cookie — vulnerable to
   token theft via any XSS elsewhere on the site. Worth revisiting once the
   secret itself is fixed, but is a separate, lower-priority issue.

3. **Schema drift**: `handlers/timetable.go` and `seed_timetable.sql` query
   `timetable` and `activity_logs` tables that **do not exist** in either
   `schema.sql` or `schema.sqlite`. Running either schema file fresh will
   cause `/api/v1/timetable` and `/api/v1/reflections` to fail at runtime
   with "relation does not exist." These tables need to be added to both
   schema files before this is trustworthy.

4. **`reviews` table exists in schema but has no handler or route at all**
   — designed but never built. Don't assume review functionality exists
   anywhere in the app.

5. **Redundant backup files committed to the repo** — currently:
   - `handlers/admin.go.backup-20260503-150656`
   - `handlers/search.go.backup-20260505-223637`
   - `templates/dashboard.html.backup`
   - `templates/dashboard.html.backup-20260503`
   - `templates/search-page.html.backup-20260503`
   - `templates/search-page.html.backup-20260505-223648`

   These should be deleted from the repo, and `*.backup*` added to
   `.gitignore` so this doesn't recur. **This is the first cleanup task the
   project owner wants done, before any security or feature work.**

6. **No rate limiting or lockout on `/api/v1/login`** — unlimited password
   guesses are possible against any account.

7. **No password-reset flow for regular users** — only
   `handlers.ResetAdminPassword` exists, which is admin-only.

8. **`.gitignore` correctly covers `.env`** — unlike a companion project
   this developer works on, there's no evidence this one has leaked
   credentials. Keep it that way.

---

## WORKFLOW RULES (strict — follow exactly)

1. **One step at a time.** Give one file, or one small paired edit, per
   response. Do not bundle cleanup, security fixes, and features into one
   giant reply.
2. **Current phase order, as directed by the project owner**: (1) remove
   redundant/backup files first, (2) patch security issues (starting with
   the hardcoded JWT secret), (3) then move to feature work. Don't jump
   ahead to features while cleanup or security items are still open unless
   explicitly told to.
3. After each step, **stop and wait** for confirmation that the change was
   made and tested before giving the next step.
4. **Never assume unstated requirements.** If a request is ambiguous, ask a
   clarifying question before writing code.
5. **Follow the existing conventions exactly** — inline SQL in handlers (no
   repository layer), Gin route registration in `main.go`, Tailwind CDN +
   vanilla JS on the frontend. Don't introduce a different architecture
   partway through.
6. When giving code, give the exact full block to paste — not a vague diff
   — unless the edit is a small, precise change to one existing file.
7. Don't touch unrelated features while working on one area, unless the
   task explicitly requires it (e.g. fixing the shared JWT secret does
   legitimately require touching both `middleware/auth.go` and
   `handlers/auth.go` — that's expected, not scope creep).

Wait for the project owner to confirm before starting; do not assume the
backup-file cleanup is already done just because it was listed above.