# L2EStudyLink

A peer tutoring / study-session booking platform (part of the Learn2Earn ecosystem) where students and tutors list skills and availability, search each other, book sessions, and track personal study timetables and reflections.

## Tech Stack

- **Language**: Go, using the **Gin** web framework (`github.com/gin-gonic/gin`)
- **Pattern**: hybrid app — server-rendered HTML pages via `router.LoadHTMLGlob("templates/*.html")` for the UI shell, plus a JSON REST API under `/api/v1/*` that page-level JavaScript calls via `fetch()`. Pages are "dumb" wrappers; real logic lives behind the API.
- **Auth**: JWT (`github.com/golang-jwt/jwt/v5`) issued on login and password reset in a 24-hour Secure, HttpOnly, SameSite=Strict cookie, verified by `middleware.AuthRequired`. Cookie-authenticated writes require a same-origin `Origin` header. Older Bearer tokens remain accepted until their existing 24-hour expiry; signed-in pages migrate them to cookies and clear browser token storage.
- **Database**: PostgreSQL in production, SQLite schema is maintained for local use (`schema.sql` / `schema.sqlite`); `db/postgres.go` uses pgx's `database/sql` adapter in simple protocol mode to work through the production connection pooler.
- **Frontend**: plain HTML, Tailwind CDN on signed-in pages, shared CSS for navigation and account pages, and vanilla JavaScript `fetch()` calls. No build step, no npm, no React/Vue. Dark mode uses a `localStorage` flag and manual style overrides.
- **Hosting**: Render (`render.yaml`, `Procfile`, `start.sh`), also has a `Dockerfile`.
- **Installable app**: web manifest and 192/512px icons enable installation from supported browsers. Dashboard shows an Install app control when the browser offers installation; iOS users can use Safari's Share → Add to Home Screen. The service worker caches only a public offline explanation and icons; account pages and API responses always use the network.
- **Email**: Brevo transactional API (`email/brevo.go`) for account and booking messages. Opted-in product-update audiences can be exported by an admin for a separate Brevo marketing campaign. **Notifications**: Discord webhook (`notifications/discord.go`).

## Architecture

- `main.go` — registers every route directly (page routes and API routes both live here, no separate routes file).
- `handlers/` — one file per feature area (`auth.go`, `booking.go`, `search.go`, `profile.go`, `availability.go`, `timetable.go`, `admin.go`, `password_reset.go`). Handlers read `db := c.MustGet("db").(*sql.DB)` and write raw SQL inline — no repository layer.
- `middleware/` — `AuthRequired` (JWT verification) and `AdminRequired` (checks the current database role and suspension status).
- `config/` — `JWTSecret()`, read from the `JWT_SECRET` env var; the app refuses to start if it's unset or still the old placeholder value.
- `templates/` — one `.html` file per page, with inline `<script>` blocks calling the JSON API. `static/js/read-api.js` handles retries for timetable and reflections GET requests; `static/js/mobile-nav.js` controls the signed-in mobile menu. Account pages share `static/css/login.css`.
- `db/postgres.go` — connection setup and repeatable startup table creation for password recovery and project collaboration, plus an existing-user product-email preference column.
- `email/`, `notifications/` — Brevo email and Discord webhook integrations.
- `schema.sql` / `schema.sqlite` — hand-written schema, no migration tool; Postgres and SQLite dialects kept in sync. Existing PostgreSQL databases automatically create the missing `password_reset_tokens` table and index at startup. Other existing schema changes still need manual migration (see Known Issues).

## Features Implemented So Far

### Learning Hub — public guides and discussion

The public landing page now links to **Learning Hub** (`/learn`). Visitors can read the article list and individual guides (`/learn/:slug`) without an account. The initial published guide covers a practical agentic coding workflow: choose a small outcome, supply context, define checks, review the changes, and verify the user flow. This editorial content is separate from private timetable reflections; no personal reflection is published automatically.

Any signed-in fellow can ask a question or comment on an article. Comments appear publicly with the account name, newest first. Posting is limited to one comment per account per minute, with a 3–1200 character limit; text is rendered as text, not HTML. An admin can publish further plain-text articles and remove comments in the Admin dashboard. Publishing is immediate, so admins should review article text before submitting. The public API is `GET /api/v1/articles`, `GET /api/v1/articles/:slug`, and `GET /api/v1/articles/:slug/comments`; authenticated posting is `POST /api/v1/articles/:slug/comments`; admin routes are `POST /api/v1/admin/articles` and `DELETE /api/v1/admin/articles/comments/:id`.

The Learning Hub opens with a featured agentic-workflow guide, clear reading and discussion calls to action, and a responsive reading list. Discussions support one-level threaded replies: visitors can read all comments and replies, while signed-in fellows can reply to a specific question or ask a follow-up. Replies belong to the same article and the same top-level thread; the existing one-post-per-minute limit applies to replies. Admin removal of a top-level comment also removes its replies. The public API returns the newest 200 comments per article, so a larger community will need pagination. The PostgreSQL startup migration adds `parent_id` to existing comments without changing them; fresh PostgreSQL and SQLite schemas include it. Learning Hub tables enable RLS and revoke client Data API grants because the Go server provides the public read and authenticated write APIs.

The initial reading list includes guides on practising an agentic workflow without a paid model plan and turning weekly goals into evidence of progress. Seeded guides use `ON CONFLICT (slug) DO NOTHING` so later startup runs do not overwrite editorial edits. The featured guide is omitted from the reading-list cards while other guides are available, avoiding duplicate cards.

After signing in, fellows arrive at `/learn`, with **Dashboard** at the top right. The dashboard opens the existing study workspace, and every signed-in sidebar has a **Learning Hub** link back to `/learn`. Public home, hub, and article pages show Dashboard in place of guest account links when a valid session exists. If someone signs in or signs up from an article, the safe `next=/learn/...` route takes them back to that article after login so they can comment. Other `next` values are ignored.

PostgreSQL creates the two new tables and seeds the first guide on startup through `db/learning_schema.sql`; fresh PostgreSQL and SQLite schemas include equivalent tables. The first guide is seeded only if its slug is absent. This initial release has simple rate limiting and admin removal; further moderation tools, abuse monitoring, search, and notifications can follow once real discussion volume warrants them.

### Auth
- Signup (`POST /api/v1/signup`) — bcrypt-hashed passwords, unique email enforced.
- Login (`POST /api/v1/login`) — issues a 24-hour JWT in an HttpOnly cookie; the JSON response does not expose it to page JavaScript. `POST /api/v1/logout` clears the cookie and browser cache scope. Legacy Bearer sessions move via `POST /api/v1/session/migrate` on their next page visit.
- Self-service password reset (`POST /api/v1/forgot-password`, `POST /api/v1/reset-password`) — six-digit email code, 10-minute expiry, five attempts, one-minute resend cooldown and single-use reset. The code is stored as a keyed digest; the request endpoint gives the same response for known and unknown accounts. Successful reset signs the user in and opens their dashboard.
- Password codes and booking emails use Brevo. Existing PostgreSQL databases create the password-reset table and index on startup. The project owner confirmed the live reset flow works after merging PR #11; this is user verification, not an automated end-to-end test.
- JWT secret is loaded from an environment variable and validated at startup (fails fast if missing or left as the old placeholder).
- Login limits each normalized email to five failed attempts within 15 minutes; further attempts return HTTP 429 until cooldown. Failure counters are keyed by a digest and stored in PostgreSQL so they survive restarts. A successful login clears that email's failures.
- A valid password can recover an account during a cooldown; invalid passwords still receive HTTP 429. Successful signup clears guesses recorded against that email before it existed, so a new fellow can sign in immediately. Signup stores normalized lowercase email addresses.

### Profile
- View own profile (`GET /api/v1/me`) — includes skills list, bio, Discord username, rating, review/session counts.
- Update profile (`PUT /api/v1/profile`) — bio, Discord username.
- Add/remove skills with proficiency level (`POST /api/v1/skills`, `DELETE /api/v1/skills/:skill`).
- Choose optional product emails on signup or turn them on or off on the dashboard (`PUT /api/v1/me/product-emails`). Existing accounts start unsubscribed; an explicit choice is required.

### Availability
- Get/set weekly availability slots (`GET`/`PUT /api/v1/availability`), keyed by day-of-week + time range.
- Availability accepts start/end times to the minute (for example 09:15–10:45). The browser offers hour and minute controls; the API validates the range and saves the whole schedule in a transaction so an invalid edit preserves existing hours.
- On startup the app synchronizes the availability row ID sequence with existing rows under a database lock. This repairs databases whose sequence lagged behind imported rows and caused `availability_pkey` errors; database details remain in server logs instead of being shown in the page.

### Search
- Search tutors by skill (`GET /api/v1/search`).
- Search tutors by skill with availability embedded in one query, deduplicated and grouped in Go to avoid N+1 queries (`GET /api/v1/search-with-availability`).
- View a specific tutor's public profile, skills, and availability (`GET /api/v1/tutors/:id`).

### Bookings
- Create a booking (`POST /api/v1/bookings`) — blocks self-booking and double-booking the same tutor slot; triggers a Discord notification (with @mentions if Discord usernames are set) and emails to both parties.
- List my bookings, as either tutor or student (`GET /api/v1/bookings`).
- Cancel a pending or confirmed booking (`DELETE /api/v1/bookings/:id`) until its scheduled start; notifies both parties. Cancellation feedback appears inline on My Study Sessions.
- Tutor confirms or cancels a booking (`PUT /api/v1/bookings/:id/status`) — only the tutor can change status; notifies both parties.
- After a confirmed session ends, the tutor can record `completed` or `no_show` (`PUT /api/v1/bookings/:id/outcome`). A completed session updates both fellows' session counts; a reported no-show increments the student's no-show count. Outcomes are final and duplicate decisions are rejected.
- The student can review a completed session once (`POST /api/v1/bookings/:id/reviews`, 1–5 stars and a 10–1000 character comment). The tutor's average rating and review count update in the same transaction. Review controls appear on My Study Sessions; the reviews table is created at startup on existing databases.

### Timetable & Reflections
- Personal weekly timetable: add/list/delete time blocks (`GET`/`POST /api/v1/timetable`, `DELETE /api/v1/timetable/:id`). The read handler returns `HH:MM` times and an empty JSON array when there are no blocks.
- Reflections tied to a timetable block and date, upsert-style (creating a reflection for an existing block+date updates it) (`POST`/`GET /api/v1/reflections`). The read handler returns `YYYY-MM-DD` dates, `HH:MM` times, and an empty JSON array when there are no reflections.
- Saving a reflection from the timetable shows a styled, accessible confirmation with a link to the Reflections page; validation and save errors appear within the form instead of browser alert dialogs.
- Deleting a timetable block currently cascades to its reflections in PostgreSQL; a styled confirmation dialog now explains this explicitly before deletion. Timetable creation feedback and validation appear in the page and form instead of browser popups. Recovering deleted entries requires a backup or another saved copy. Retaining reflections independently of a block remains a follow-up data-model change.
- Both pages retry transient network or server failures on GET requests up to three attempts, redirect to login on an expired session, and show a **Try again** control for errors. The timetable remains viewable when its separate reflections request fails. Database query and scan failures are logged on the server rather than silently dropping rows.

### Admin
- Every `/api/v1/admin/*` API route requires authentication and a current database check that `is_admin` is true and `is_suspended` is false. An old JWT does not preserve admin access after a role change or suspension.
- Stats: total users, bookings, skills (`GET /api/v1/admin/stats`).
- List all users with session/rating/suspension info (`GET /api/v1/admin/users`).
- Delete a user (`DELETE /api/v1/admin/users/:id`).
- Suspend/unsuspend a user (`PUT /api/v1/admin/users/:id/suspend`).
- List all bookings across the platform (`GET /api/v1/admin/bookings`).
- Count or export an audience of active users who opted into product emails (`GET /api/v1/admin/product-email-audience` and `.csv`). The CSV contains only deduplicated addresses, requires current admin access and is marked `no-store`.
- The former hardcoded admin password reset route and handler have been removed. `GET /api/v1/admin/users` scans PostgreSQL boolean fields as booleans.

### Pages
The public `/` route now renders a dedicated mobile-friendly landing page, with a goal → timetable → reflection example and direct login/signup links. It explains how a user's own entries build a personal learning record and describes a future personal AI agent as a possibility, not an active feature. The illustrated reflections are fictional examples; no real user's reflections are public. The phone login screen has a branded background and a home link. Login, signup, forgot/reset password, dashboard, search, my-bookings, timetable, reflections, projects, admin remain available as HTML pages with vanilla JS, with a generic `/page/:name` route for simpler pages. Six signed-in pages have a compact mobile header and menu, plus narrower card, form, modal and table layouts. The owner reports a hands-on mobile accessibility and UI/UX review with a good result; device-specific coverage and measured checks have not been provided.

**Future landing-page stories (planned, not implemented):** let users preview and explicitly select individual reflections to publish, showing exactly the text, name or attribution, and date that visitors would see. Default to private, provide a way to revoke visibility, and exclude unpublished reflections at the API/database level. Build this only after reviewing consent and moderation needs; do not surface private reflection data in the public landing-page response.

---

## Product update email campaign

**Status:** campaign copy and eligible-audience export prepared; no mass email has been sent. The original signup did not collect product-email consent, so existing registration addresses must not be enrolled automatically. Suspended accounts and users who switch the preference off are excluded. Opt-in is available on signup and the dashboard profile. Existing PostgreSQL databases gain `marketing_opt_in_at` automatically at startup; a null timestamp means unsubscribed.

An admin can check the eligible count and download a CSV in Admin → **Product email audience**. For a one-time or scheduled send, import that CSV into a dedicated Brevo marketing list, use the short copy in [docs/product-update-email.md](docs/product-update-email.md), include Brevo's unsubscribe link, preview and schedule the campaign. Re-export immediately before sending, replacing the list and honoring Brevo's own unsubscribes. The existing transactional Brevo API is reserved for account and booking mail. Automatic recurring campaigns and recipient synchronization with Brevo are not implemented; a fresh export is required for each send.

---

## AI Planner — Delivery Plan

**Status:** planned; no planner routes, AI client, planner storage, or planner page exist yet. `L2E_PLANNER_PROMPT_v3.md` is the current reference prompt in this repository. It describes one person's history and must be adapted into a reusable, user-scoped planner before integration. The first release can use the existing timetable and reflections without waiting for automated scheduling.

**First useful release:** an authenticated user chooses a past week and taps **Analyze my week**. The page shows a factual activity summary and concise coaching based only on that user's timetable and reflections, with a link to prior analyses. It distinguishes scheduled time, reflected activity, and missing reflections; a reflection's existence alone does not prove the full block was completed.

1. **Define the output and metrics before calling a model.** Specify one user-owned week, timezone, empty-week behavior, and a versioned JSON response: scheduled blocks/minutes, reflected blocks, coverage percentage, factual highlights, suggested priorities, and one follow-up question. Compute counts and scheduled minutes in Go; do not infer actual hours worked or adherence from free-text reflections. Add tests for week boundaries, missing logs, and another user's data.
2. **Prepare per-user context.** Read only the authenticated user's timetable, reflections, and selected prior summaries. Bound the date range and text size; treat reflection text as untrusted input and never allow it to override system instructions. Adapt the v3 reference into a generic coaching prompt without distributing its author's personal history to other fellows. Keep the model responsible for qualitative feedback, not arithmetic.
3. **Choose a provider and build a narrow client.** Keep the server-side key in an environment variable and never expose it to page JavaScript. Configure timeouts, output size, retries for transient failures, schema validation, and a useful failure response. Do not make an AI key mandatory for existing app startup while the planner is optional. Test with a fake HTTP provider before using a real key.
4. **Persist and expose analyses.** Add a versioned PostgreSQL migration for planner analyses, plus the matching SQLite schema. Store user ID, week, deterministic metrics, validated model response, prompt/model version, and timestamps. Add `POST /api/v1/planner/analyze` and `GET /api/v1/planner/history`, scoped to the logged-in user; prevent duplicate submissions for the same week and set a per-user usage limit.
5. **Build the phone-first planner page.** Render numbers and feedback from validated JSON as text, never model HTML. Show loading, empty, error, retry, and history states. Ensure keyboard access, readable charts/tables on narrow screens, and a clear distinction between observed activity and AI suggestions.
6. **Pilot before expanding.** Compare generated summaries against sample weeks, check privacy boundaries and costs, and collect feedback from a small group. Only then consider recurring flags, daily views, scheduled reports, or an interactive follow-up conversation.

**Decision for the first release:** on-demand weekly analyses using each user's own schedule. Define the user's timezone and whether they want a weekly goal before adding adherence or streak scores. Provider and budget still need product decisions; they do not block the deterministic metrics prototype.

## Mobile UX — Delivery Plan

**Status:** mobile UX implementation and the owner's hands-on mobile accessibility and UI/UX review are complete; the owner reported a good result on 2026-09-24. The code audit below covers the core phone journeys. Specific Android/iPhone browser, screen reader, 200% zoom, and measured task-completion checks have not been reported; keep them as follow-up quality checks.

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

**Verification completed:** JavaScript syntax and diff checks for affected changes, targeted checks for booking date/slot transitions and reflection filtering, the project owner's live phone review through PR #16, the owner's final acceptance of the mobile UX pass after PR #23, and a subsequent owner-reported hands-on mobile accessibility and UI/UX review on 2026-09-24. No Go handlers were changed in that mobile pass. Acceptance is based on the owner's confirmation; it does not establish that every device and accessibility scenario below was exercised.

**Follow-up device and accessibility checklist (not individually verified):** check 320px and common phone widths on Android and iPhone where available. Open/close Menu with touch and keyboard; search, change a tutor's booking date, select a meeting type, and submit a test booking; inspect booking actions; add a block and reflection; jump across timetable days; filter and clear reflection history; use login, signup and reset fields; scroll admin tables without moving the whole page. Confirm no horizontal page overflow, clipped modal buttons, keyboard-covered actions, or unreadable light/dark text. Do not submit admin deletion during this audit. Record any device-specific defect as a new focused task.

1. **Audit real tasks at phone widths.** Check signup, login, OTP reset, search and tutor profile, booking and cancellation, timetable creation, reflections, and admin (for admins) at 320, 375, 390, and 768 CSS pixels. Record horizontal overflow, clipped controls, keyboard overlap, slow states, and confusing navigation. Test at least one actual phone, including a slow network, before calling the release mobile ready.
2. **Create a consistent interaction system.** Reuse the existing shared styles for type, spacing, buttons, fields, focus indicators, error and success messages, and touch targets. Keep navigation reachable with one hand; preserve desktop behavior. Ensure menu focus and Escape behavior, visible labels, accessible modal focus, and reduced-motion support.
3. **Improve core journeys in small PRs.** First make search results, tutor details, and booking actions easy to scan and tap. Then make timetable and reflections quick to enter and review by date. Show clear loading, empty, validation, success, and retry states. Use mobile cards or scoped horizontal scrolling where data tables cannot fit; never make the whole page scroll sideways.
4. **Verify each journey.** Test a 320px viewport, keyboard-only navigation, screen reader labels, zoom to 200%, light/dark contrast, and an Android and iPhone browser where available. Measure real task completion and perceived speed with users before considering an installable PWA. A PWA is an optional later milestone, not a prerequisite for phone access through the browser.

---

## Profile editor and personal page performance — Implemented

The dashboard profile uses an inline, keyboard accessible form with Save and Cancel, validation feedback, and fields labelled **Discord username** and **Bio**. The existing `discord_username` API field and account data remain compatible.

Timetable and Reflections use a short (20 second) browser session cache of successful authenticated GET responses, keyed to a one-way fingerprint of the current non-secret session scope (or a legacy token during migration). Simultaneous reads of the same resource share one request; timetable blocks and reflections are loaded concurrently. The dashboard warms both personal pages after it becomes interactive. Successful timetable and reflection changes invalidate their affected cache entries before reloading, and logging out from those planning pages or the dashboard clears both cached resources. These personal responses are never stored in a shared server cache; different sign-ins cannot reuse another account's cached data. Creating the same timetable block on multiple days submits those independent days concurrently, with the Save button disabled until they settle. The PostgreSQL pool has a bounded maximum of eight open and three idle connections. The cache improves repeat navigation and reduces duplicate reads, but its short lifetime means a change made in another browser can take up to 20 seconds to appear.

`node --test static/js/read-api.test.cjs` verifies concurrent request sharing, account separation and invalidation, and is part of GitHub Actions alongside the Go tests.

---

## Skills and availability refinement — Implemented

Skills have clear Edit and Remove controls with accessible labels and a responsive action layout. The Availability page shows weekly slots in a day-by-day presentation with a slot count, clearer Remove actions and an encouraging empty state. Availability reads retry briefly if the server is temporarily unavailable and offer an in-page retry when they still fail; writes are never automatically retried. The availability API now checks row scan and iteration errors and returns time values consistently as `HH:MM`.

---

## Interface refresh — Implemented

All signed-in pages now share the same navigation grouping, page title hierarchy, controls, spacing, focus states, responsive layouts, and light/dark surfaces. Dashboard, discovery, sessions, skills, availability, projects, Masterclass, timetable, reflections, and admin each have a clear page introduction. The timetable, session list, and reflections show clearer next actions and empty states. Login, signup, forgot password, and reset password have a consistent responsive presentation. The update changes presentation and page navigation cues; existing API contracts and booking or project workflows are preserved.

To review after deployment, check one auth page, the dashboard, each sidebar destination, and the booking form at desktop and phone widths. Confirm light/dark appearance, keyboard focus, long lists and tables, and that buttons still complete their original action.

---

## Dashboard discovery, skills and Masterclass — Implemented

The signed-in dashboard previews open projects and fellows who opted in to project invitations. Matches are ordered by overlap with the viewer's listed skills; when no skills match, other open projects and opted-in fellows remain discoverable. Project cards link to the Projects page. Fellow cards show skills and weekly availability; the full opted-in fellow list and availability are on `/projects#fellows`. Availability is visible in this collaboration list only after the fellow opts in. The existing study-partner search shows tutors' availability separately.

The two dashboard recommendations load independently: if one endpoint fails, the other section remains visible and the failed section offers a styled retry control. PostgreSQL integration tests cover another fellow seeing an open project and an opted-in collaborator with their skills and hours. Project and collaborator listing failures write their database error to server logs while returning a generic API error to the browser. Production logs on 2026-09-24 identified prepared-statement protocol errors affecting both lists and personal planning pages; the database driver now uses the simple query protocol. After the owner opened the deployed pages, Render logged HTTP 200 for both lists, timetable and reflections, with no new database query errors or HTTP 500 in that observation window.

A fellow can add, rename, change proficiency, and remove skills on the dedicated `/skills` page. Weekly tutoring hours are managed on `/availability`; both pages appear in the signed-in sidebar. `PUT /api/v1/skills/:skill` updates only their own existing skill; conflicts and invalid levels are rejected. The dashboard gives a concise preview of skills and availability and links to their settings pages.

`/masterclass` is the first learning discovery page: search a topic to see fellows with that skill and their weekly availability, then choose a date, available time, and meeting type and book the selected fellow on the Masterclass page. This version does not create or advertise group classes, scheduled events, or class seats. The Masterclass link appears in the signed-in navigation; a follow-up can add hosted group sessions after the individual learning flow is validated.

---

## Project Collaboration — Implemented

Fellows can post projects with a description, needed roles, expected time commitment and optional repository link. Project listings are available to signed-in fellows; owners can close and reopen recruitment. Members of a closed project and its owner can still view it.

A fellow can opt in to invitations on the Projects page. Owners can browse opted-in fellows and invite one; another fellow can request to join an open project with a note. Invitations and applications appear in the relevant person's Projects inbox. The owner decides applications, while the invited fellow decides invitations. Acceptance adds the fellow to the team; duplicate requests and self-joining are blocked. The team and request status are visible in project details. Invitations are opt-in and no email is sent in this first version; fellows check the in-app inbox.

`/projects` is a mobile-friendly page. Authenticated API routes are `GET`/`POST /api/v1/projects`, `GET /api/v1/projects/:id`, `PUT /api/v1/projects/:id/status`, `POST /api/v1/projects/:id/requests`, `GET`/`PUT /api/v1/collaboration/preferences`, `GET /api/v1/collaborators`, `GET /api/v1/project-requests`, and `PUT /api/v1/project-requests/:id/status`. Project tables are added automatically to existing PostgreSQL databases on startup; both schema files define them for fresh databases. GitHub Actions runs Go tests/build and a fresh SQLite schema check.

Project discovery and team membership live here; code, issues and day-to-day team communication remain in the tools fellows already use. Assess real project posts, relevant requests, accepted teammates, and first working sessions during a pilot before expanding to messaging or automated matching.

---

## Known Issues / Gaps

Ranked roughly by how much they'd block real usage:

> **Recently resolved:** `timetable`, `activity_logs`, and `users.is_admin` are now defined in both `schema.sql` and `schema.sqlite` (kept in sync, dialect-correct). Fresh databases created from either file fully support `/api/v1/timetable`, `/api/v1/reflections`, `GetProfile`, and `AdminUsers` — this also unblocks the AI Planner work above. ⚠️ **Existing databases are not changed by these files:** `CREATE TABLE IF NOT EXISTS` adds the new tables but won't add `is_admin` to an existing `users` table. Apply it manually — Postgres: `ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT FALSE;` plus the two tables from `schema.sql`; SQLite: `ALTER TABLE users ADD COLUMN is_admin INTEGER DEFAULT 0;` plus the tables from `schema.sqlite`.

> **Resolved and checked live (2026-09-24):** Render logs showed PostgreSQL `08P01` result-format mismatches and `26000` missing unnamed prepared statements across Projects, Collaborators, Timetable, and Reflections. The shared driver now avoids that prepared-statement protocol. After the owner opened the deployed pages, each affected endpoint returned HTTP 200 and no new query errors or HTTP 500 appeared in the observed logs. This confirms those requests during the check; monitor later traffic for recurrence.

1. **Account-specific login throttle is in place; broader IP-based abuse controls are still possible.** Shared or rotating identities can still generate aggregate load.

## Suggested Next Steps (in priority order)

1. The owner confirmed hands-on mobile accessibility and UI/UX review on 2026-09-24. If a device-specific defect appears later, fix it on a focused branch. Recheck reset with a fresh code when account behavior changes.
2. Monitor Projects, Collaborators, Timetable, and Reflections for any recurrence of production query errors.
3. Build and test deterministic weekly metrics from the current user’s data; document the week/timezone and missing-reflection rules.
4. Obtain the planner reference prompt, choose a provider and budget, then deliver the scoped weekly planner API and mobile page in reviewable steps above.
5. Confirm cookie sign-in, password reset, logout, and legacy-session migration in the deployed browser; consider network-level rate limits for aggregate abuse.

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
