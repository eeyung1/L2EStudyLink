# L2EStudyLink

A peer tutoring / study-session booking platform (part of the Learn2Earn ecosystem) where students and tutors list skills and availability, search each other, book sessions, and track personal study timetables and reflections.

## Redesign roadmap — November 2026

**Stage: redesign implementation and automated release checks complete. Production smoke and physical-device acceptance remain pending.** This roadmap follows the owner-supplied `frontend-engineer.md` guidance and the agreed redesign plan. Checkboxes below describe the new redesign, not whether an older version of a screen exists.

### Product outcome

Make L2EStudyLink a coherent learning workspace where a fellow can see today's commitments, study with others, record a reflection and understand their progress. Prioritize the dashboard, timetable and reflection journey while retaining peer tutoring, projects, learning content and administration.

### Baseline and boundaries

Source inspection at commit `2b92f6056e99f28ee8c5cfb9e377d91c5d17ab37` confirms Go/Gin routing, HTML templates, shared CSS, vanilla JavaScript and PostgreSQL. Preserve this stack and existing API contracts. A new frontend framework, database migration, hosting move or AI-provider change is not required by this redesign.

| Area | Existing source | Redesign implication |
| --- | --- | --- |
| Dashboard | Profile/setup, heatmap and collaboration discovery | Give today's study actions priority; move community suggestions below them. |
| Navigation | Sidebar markup repeated across templates; destinations differ | Establish one route inventory and consistent grouping on every signed-in page. |
| Styling | Shared dashboard, theme and polish styles plus inline/page rules | Consolidate semantic tokens and components incrementally; remove obsolete rules only after verifying their consumers. |
| Planner | Weekly metrics, Groq client, authenticated page and analysis endpoint exist | Redesign existing output; do not describe the whole feature as unimplemented. Saved analysis history remains separate work. |
| Other learning tools | Focus, buddies, notifications, group masterclasses and portfolio handlers/routes exist | Expose and validate existing flows before proposing replacements. Source presence does not prove live acceptance or every backlog requirement is complete. |
| Testing | Go tests/build, JavaScript tests and SQLite schema validation in CI | Retain these gates and add browser/layout/accessibility checks for redesigned flows. |

Older implementation notes below retain useful history. Where an older section calls an existing module entirely “planned,” the source-backed baseline here takes precedence. Reconcile remaining stale documentation during the audit rather than treating all old checklists as current defects. `Agent.md` also contains historical architecture/security statements that require checking against current code.

### Experience and navigation

Desktop: a persistent, grouped sidebar and a compact page header with page title, notifications, theme and account controls. Mobile: a compact header, thumb-friendly primary navigation and a More menu for secondary destinations. Final mobile labels and geometry must be verified in the first preview.

| Group | Destinations |
| --- | --- |
| My learning | Today/dashboard, timetable, reflections, weekly coach, focus rooms |
| Learn together | Tutor search, bookings, buddies, masterclasses |
| Explore | Learning Hub, projects, learner portfolio |
| Account | Profile, skills, availability and preferences; admin only for permitted accounts |

Keep existing URLs and browser back/forward behavior. Navigation visibility is not authorization; server-side access checks remain authoritative.

### Visual direction and benchmarks

Use a calm learning workspace with readable typography, clear hierarchy, restrained shadows, rounded cards and an emerald primary accent. Define shared canvas, surface, text, border, focus, status and spacing tokens for both light and dark themes. Status uses text/icons as well as colour.

Benchmark the task/schedule relationship in [Notion Calendar](https://www.notion.com/help/guides/getting-started-with-notion-calendar), availability and booking clarity in [Calendly](https://calendly.com/help/how-to-set-your-availability), and entry into a focused session in [Focusmate](https://www.focusmate.com/how-it-works/). Adapt useful patterns to this product; do not copy their content or add unrelated features.

Use short hover, focus, save and loading transitions with reduced-motion alternatives. Study and reading screens should remain visually quiet. Any decorative landing-page motion requires a demonstrated product purpose and a mobile performance check.

### Delivery batches and acceptance criteria

#### Batch 1 — Audit, shared foundation, dashboard and timetable preview

- [x] Inventory public, signed-in and admin routes, navigation links, forms, dialogs and shared style consumers.
- [x] Record baseline screenshots and loading, empty, populated, error and expired-session states.
- [x] Record existing behavior and defects separately from proposed design improvements.
- [x] Prepare phone and desktop dashboard/timetable previews using the agreed design direction.
- [x] Define theme, typography, spacing, elevation, icon and component tokens.
- [x] Establish shared buttons, inputs, cards, tabs, dialogs, feedback and loading patterns within the existing template stack.
- [x] Apply consistent grouped navigation, active-page labels and keyboard-accessible mobile menus.
- [x] Reorganize the dashboard around today's blocks, next booked session, Add reflection, focus access and weekly coverage.
- [x] Put setup tasks and collaboration recommendations below primary daily actions.
- [x] Present a day agenda on phones and an accessible weekly overview on larger screens.
- [x] Preserve selected timetable day after existing create/delete/reflection actions.
- [x] Add browser checks for shared navigation, dashboard and timetable before merging this batch.

**Foundation evidence:** [audit and preview guide](docs/redesign/foundation-audit.md), [generated source inventory](docs/redesign/source-inventory.json), and `/static/redesign/index.html`. The first five checked items cover source inventory, fictional-fixture baseline captures of Dashboard/Timetable, recorded findings, responsive previews and scoped tokens. CI artifact `redesign-foundation-evidence` contains screenshots and measurements. The app integration is delivered below; production UI smoke and physical-device acceptance remain unchecked.

**Acceptance:** a fellow can identify the next action, reach existing learning tools and browse their timetable on phone and desktop. Placeholder previews are explicitly labelled; production cards use actual API data and honest empty states. Review preview screenshots before rollout; physical-device acceptance is tracked separately.

#### Batch 2 — Reflection entry and history

- [x] Unify Write/Speak reflection modes with persistent field labels and clear focus states.
- [x] Show recording, duration, stop, audio preview, re-record, upload and save states accurately.
- [x] Handle microphone denial, unsupported recording, interrupted uploads and save failures without silently losing text.
- [x] Make dialogs/sheets fit small screens and on-screen keyboards; support Escape and restore focus.
- [x] Improve history filtering, date/activity labels and audio playback.
- [x] Keep personal reflections private and account-scoped.
- [x] Verify reflection retention against current schema and deployed migrations before changing deletion copy; any data-model fix must be a separately tested, explicit change.
- [x] Verify text and voice journeys, validation, retry and account separation.

**Acceptance:** users can review what will be saved, recover from failure, and find previous entries. Do not imply transcription, offline persistence or permanent retention unless implemented and verified.

#### Batch 3 — Tutor discovery, availability and booking

- [x] Make tutor cards and skill/availability filters easy to scan and operate.
- [x] Clearly separate tutoring availability from personal study timetable blocks.
- [x] Clarify selected tutor, date, time, timezone and meeting type before booking.
- [x] Preserve validation, conflict handling, self-booking protections and existing notifications.
- [x] Present pending, confirmed, cancelled, completed and no-show states with permitted actions.
- [x] Refine post-session reviews and empty/loading/error states.

**Acceptance:** search → tutor → slot → booking → status/review works end to end, including unavailable slots, cancellation and role-specific controls.

#### Batch 4 — Community, projects and learning content

- [x] Align project discovery, details, applications, invitations and team status.
- [x] Make collaboration opt-in and request decisions clear.
- [x] Expose and validate the existing buddy request/response journey.
- [x] Align masterclass listing, creation, RSVP, seat availability and cancellation.
- [x] Improve Learning Hub article hierarchy, discussion/reply layout and moderation feedback.
- [x] Unify notification tray, unread states and navigation to relevant records.
- [x] Test long content, empty lists, permissions and partial API failures.

**Acceptance:** existing community actions are discoverable and retain their authorization and data-visibility rules. Do not advertise automated buddy matching, external nudges or notification coverage beyond verified implementation.

**Community delivery evidence:** [PR #71](https://github.com/eeyung1/L2EStudyLink/pull/71) and the [community and post-session guide](docs/redesign/community.md) records the five-item scope, boundaries and CI/browser checks. Landing and login/account pages remain in Batch 6 below.

#### Batch 5 — Weekly coach, focus and learner identity

- [x] Present weekly coverage, daily breakdown and coaching in readable layouts.
- [x] Clearly label scheduled/reflection-covered minutes as estimates, not verified hours worked.
- [x] Show provider-unavailable, rate-limit, empty-week and truncated-context states.
- [x] Keep the planner author's personal history out of other users' coaching.
- [x] Align focus-room join/leave, timer, participant and reflection handoff states.
- [x] Align learner portfolio visibility, handle selection and public/private presentation.
- [x] Verify these flows using each user's own records.

**Acceptance:** observed activity is visually distinct from AI suggestions. Saved reviews, long-term trends and new coaching capabilities are separate product work until delivered.

**Learning/coverage evidence:** [PR #72](https://github.com/eeyung1/L2EStudyLink/pull/72) and the [Learning Hub, notification and coach guide](docs/redesign/learning-notifications-coach.md) records this five-item delivery, API boundaries and verification.

**Focus/identity evidence:** [PR #73](https://github.com/eeyung1/L2EStudyLink/pull/73) and the [weekly coaching states, focus and learner identity guide](docs/redesign/focus-identity.md) records delivery, privacy assertions, existing API compatibility and verification boundaries.

#### Batch 6 — Landing, account and administration

- [x] Refresh the landing page around the real plan → study → reflect journey and clear entry actions.
- [x] Align signup, login, reset-code and recovery screens.
- [x] Preserve safe return-to-article behavior and existing session handling.
- [x] Refine profile, skills, availability and product-email preference controls.
- [x] Make admin tables/cards, filters and consequential-action confirmations readable on phones.
- [x] Keep admin-only actions authorized on the server and public content free of private reflections.

**Acceptance:** public/account/admin journeys share the design system, with correctly labelled forms, clear recovery paths and no unintended changes to consent.

**Delivery evidence:** [PR #74](https://github.com/eeyung1/L2EStudyLink/pull/74) and the [public, account and admin guide](docs/redesign/public-account-admin.md) record scope, automated checks and remaining acceptance limits. The server authorization/public-content audit is recorded in the [release audit](docs/redesign/release-audit.md).

#### Batch 7 — Release audit and acceptance

- [x] Complete the viewport and accessibility matrix below.
- [x] Test loading, empty, error, retry, slow-network and expired-session states.
- [x] Verify browser back/forward, repeated saves and double-submit prevention.
- [x] Check 200% zoom, keyboard navigation, focus visibility and dialog focus restoration.
- [ ] Record production smoke results after the deployment associated with the merged commit.
- [ ] Obtain and record owner acceptance on a physical phone and laptop.
- [x] Update remaining limitations and checked/unchecked roadmap items honestly.

**Automated release evidence:** [PR #75](https://github.com/eeyung1/L2EStudyLink/pull/75) and the [release audit guide](docs/redesign/release-audit.md) records the production-route authorization checks, all-screen viewport matrix, recovery/navigation checks and keyboard/reflow coverage. The 200% reflow check uses the equivalent reduced CSS viewport; physical browser zoom, screen-reader and device acceptance remain owner checks.

### Post-redesign improvement batch 1

The owner selected this separate product batch after the redesign implementation checklist was exhausted.

- [x] Add bounded admin account pages with server search/status filters and load-more recovery.
- [x] Add bounded admin booking pages with server search/status filters and load-more recovery.
- [x] Paginate Learning Hub guides and discussion, including discovery across the full guide collection.
- [x] Protect tutoring availability edits against stale-tab overwrites with account-scoped revisions.
- [x] Warn before leaving unsaved profile, skill, availability, reflection, review, discussion, project, masterclass and guide-publishing work.

**Evidence and remaining work:** [PR #76](https://github.com/eeyung1/L2EStudyLink/pull/76) and the [list and editing safety guide](docs/redesign/list-safety.md). Existing clients retain their response contracts; the redesigned lists opt in to bounded pages. Unsaved warnings do not store drafts or guarantee recovery after closing a tab. Hosting is restarted manually by the owner.

### Remaining features and acceptance

Only these two release acceptance tasks remain from the redesign checklist:

1. Record a production smoke check against the deployed merged commit, including account and mail behavior.
2. Record owner phone/laptop acceptance, with actual browser zoom and screen-reader review; record Safari/iPhone coverage where available.

The following are **separate future product proposals**, not unfinished redesign screens and not included in this batch:

| Proposal | Current boundary |
| --- | --- |
| Saved coaching analyses/history | Coaching is generated on demand and is not saved. |
| Long-term learning trends | The weekly view uses current timetable estimates; a historical model needs its own design. |
| Personalized coaching memory | Each request uses the selected week's own completed reflections; no long-term model memory. |
| Automated opt-in buddy matching | Requests and accepted connections exist; discovery is currently manual. |
| Scheduled accountability nudges and external milestone alerts | In-app activity events exist; no automatic missed-reflection scheduling or outbound buddy campaign. |
| More notification events | Booking, buddy and reflection events exist; project requests, tutor reviews and guide comments need additional event coverage. |
| Durable private draft recovery | In-page drafts and leave warnings exist; no automatic browser/database draft persistence. |
| Selected reflection publishing | Reflections remain private. Public stories need explicit selection, preview, revocation and moderation design. |
| Automated product email campaigns/synchronization | Explicit opt-in audience export exists; campaigns remain a separate manual Brevo process. |
| Aggregate network/IP abuse controls | Account-specific login throttling exists; broader traffic controls need deployment-aware design. |

Legacy proposal sections later in this README are historical context. This table and the delivered batch evidence take precedence when deciding what remains.

### Required verification matrix

| Area | Checks |
| --- | --- |
| Compact phones | 320, 360, 375, 390, 412 and 480 CSS pixels; no unintended page overflow or clipped primary controls. |
| Tablet/desktop | 768, 820, 1024, 1280, 1440 and 1920 CSS pixels; spot-check 2560/3840 wide layouts. |
| Geometry | Header does not obscure title/eyebrow/forms; fixed bars include safe-area insets; overlays fit the viewport. |
| Touch/forms | Primary touch targets at least 44 × 44px; mobile inputs at least 16px; correct input types/input modes and persistent labels. |
| Themes | Both light and dark themes; readable text/icons/badges, visible borders/focus and no theme flash where avoidable. |
| Accessibility | Axe scans targeting WCAG 2.1 AA plus manual keyboard, reading order and screen-reader-label checks; fix or explicitly document residual issues. |
| Motion | Reduced-motion preference respected; no essential information depends on animation. |
| Data/security | Account isolation, admin restrictions, existing cookie/session handling and private/public content boundaries. |
| Performance | Compare request counts, asset sizes, loading behavior and layout shift against baseline; report measurements rather than claiming zero shift without evidence. |
| Devices | Physical Android and laptop review; iPhone/Safari review where available, with unavailable coverage explicitly recorded. |

Automated browser checks must assert key element bounds, horizontal overflow and complete relevant user journeys. Mock fixtures may cover repeatable states, but do not count as proof that production APIs work. Use a dedicated test database and disposable fixtures for writes, emails, uploads and admin actions; do not exercise destructive actions on production accounts.

### Workflow for every batch

1. Start from current `main`; create a descriptive feature branch such as `agent/redesign-foundation`.
2. Implement one coherent batch, preserving existing API behavior unless an explicit, tested change is necessary.
3. Run `go test ./...`, `go build ./...`, `node --test static/js/*.test.cjs` and fresh SQLite schema validation. Use `TEST_DATABASE_URL` with a disposable PostgreSQL database for database-backed tests.
4. During the first implementation batch, add the browser/accessibility harness and document its exact commands. The verification-only `package.json` now provides `npm run test:ui` and `npm run audit:ui`; see the foundation guide for browser installation and evidence details. There is no production frontend build or TypeScript check to assume exists. Add `go vet ./...` and the new browser gates to CI as part of that tooling work.
5. Review the diff and screenshots, commit clearly without `Co-authored-by:` trailers, push the branch and open a PR describing behavior, verification and limitations.
6. Wait for all applicable checks on the current PR head to pass. Fix failures and rerun; never bypass checks to merge.
7. Squash-merge through the PR. Update README completion boxes in the same change only for delivered work, linking the PR/verification record. Keep device acceptance unchecked until it actually occurs.
8. Verify the resulting deployment, sync the local base branch when a checkout is used, and delete only the completed task branch after confirming merge. Preserve unrelated branches and uncommitted work.

Related implementation, interface and meaningful tests can be delivered together. Keep database migrations and unrelated features out of a visual batch unless explicitly needed. The first foundation implementation delivers the five items listed above. The remaining production integration tasks continue in order through the same workflow.

### Tracking rules and separate product work

- `[ ]` means pending; `[x]` means delivered with evidence, not merely planned or present in an old mockup.
- PR descriptions should record changed screens, test commands, viewports/themes, screenshots, known limitations and deployment status.
- Saved planner analyses/history, personalized coaching memory, automated buddy matching/nudges, new portfolio evidence claims and additional notification event coverage require their own scoped product tasks.
- Never claim “verified learning hours” from timetable durations or “mastered skills” from an unverified self-report.
- Confirm current hosting/deployment settings before release; repository hosting files alone do not establish the active production host.

---

## Tech Stack

- **Language**: Go, using the **Gin** web framework (`github.com/gin-gonic/gin`)
- **Pattern**: hybrid app — server-rendered HTML pages via `router.LoadHTMLGlob("templates/*.html")` for the UI shell, plus a JSON REST API under `/api/v1/*` that page-level JavaScript calls via `fetch()`. Pages are "dumb" wrappers; real logic lives behind the API.
- **Auth**: JWT (`github.com/golang-jwt/jwt/v5`) issued on login and password reset in a 24-hour Secure, HttpOnly, SameSite=Strict cookie, verified by `middleware.AuthRequired`. Cookie-authenticated writes require a same-origin `Origin` header. Older Bearer tokens remain accepted until their existing 24-hour expiry; signed-in pages migrate them to cookies and clear browser token storage.
- **Database**: PostgreSQL in production, SQLite schema is maintained for local use (`schema.sql` / `schema.sqlite`); `db/postgres.go` uses pgx's `database/sql` adapter in simple protocol mode to work through the production connection pooler.
- **Frontend**: plain HTML, Tailwind CDN on signed-in pages, shared CSS for navigation and account pages, and vanilla JavaScript `fetch()` calls. No production frontend build step or React/Vue. npm dependencies are used only for the browser/accessibility verification harness. Dark mode uses a `localStorage` flag and manual style overrides.
- **Hosting**: Render (`render.yaml`, `Procfile`, `start.sh`), also has a `Dockerfile`.
- **Installable app**: web manifest and 192/512px icons enable installation from supported browsers. Dashboard shows an Install app control when the browser offers installation; iOS users can use Safari's Share → Add to Home Screen. The service worker caches only a public offline explanation and icons; account pages and API responses always use the network.
- **Email**: Brevo transactional API (`email/brevo.go`) for account and booking messages. Opted-in product-update audiences can be exported by an admin for a separate Brevo marketing campaign. **Notifications**: Discord webhook (`notifications/discord.go`).

## Architecture

- `main.go` — registers every route directly (page routes and API routes both live here, no separate routes file).
- `handlers/` — one file per feature area (`auth.go`, `booking.go`, `search.go`, `profile.go`, `availability.go`, `timetable.go`, `admin.go`, `password_reset.go`). Handlers read `db := c.MustGet("db").(*sql.DB)` and write raw SQL inline — no repository layer.
- `middleware/` — `AuthRequired` (JWT verification) and `AdminRequired` (checks the current database role and suspension status).
- `config/` — `JWTSecret()`, read from the `JWT_SECRET` env var; the app refuses to start if it's unset or still the old placeholder value.
- `templates/` — one `.html` file per page, with inline `<script>` blocks calling the JSON API. `static/js/read-api.js` handles retries for timetable and reflections GET requests; `static/js/mobile-nav.js` controls the signed-in mobile menu. Account pages share `static/css/public-account.css` and the reading design system.
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
- The timetable opens on today’s blocks using the user’s device date. Day buttons show only the selected day; **All** shows the full week. The selection stays active after adding/deleting blocks or saving a reflection, with accessible controls on desktop and phones.
- Personal weekly timetable: add/list/delete time blocks (`GET`/`POST /api/v1/timetable`, `DELETE /api/v1/timetable/:id`). The read handler returns `HH:MM` times and an empty JSON array when there are no blocks.
- Reflections tied to a timetable block and date, upsert-style (creating a reflection for an existing block+date updates it) (`POST`/`GET /api/v1/reflections`). The read handler returns `YYYY-MM-DD` dates, `HH:MM` times, and an empty JSON array when there are no reflections.
- Voice reflection recording & uploads: audio recordings can be uploaded (`POST /api/v1/voice/upload`) and saved to `uploads/voice/`, storing an `audio_url` reference on `activity_logs` with audio player preview support.
- Habit consistency heatmap & streak tracking: `GET /api/v1/me/streaks` computes the user's active daily streak, longest streak, and total active reflection days, powering an interactive GitHub-style contribution heatmap on both Dashboard and Reflections pages.
- Saving a reflection from the timetable shows a styled, accessible confirmation with a link to the Reflections page; validation and save errors appear within the form instead of browser alert dialogs.
- Deleting a timetable block retains saved reflections and their activity snapshot. Private history remains account-scoped, including audio access. The confirmation explains retention before deletion; see the reflection reliability/privacy guides.
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
The public `/` route presents an interactive, explicitly fictional plan → study → reflect example, links to the Learning Hub and account entry points, and explains reflection privacy, optional portfolio publishing and requested coaching. Login, signup and reset-code recovery share labelled forms, appearance controls and safe article return links. Profile and optional product-email preferences are available on `/skills`, alongside skill editing; `/availability` publishes weekly tutoring hours separately from the personal timetable. Admin uses responsive record cards, account/booking filters and native confirmations for consequential actions. These pages retain the Go/Gin and vanilla JavaScript architecture. Physical-device and production acceptance remain outstanding; earlier owner-reported mobile feedback is not a substitute for the current release checks.

**Future landing-page stories (planned, not implemented):** let users preview and explicitly select individual reflections to publish, showing exactly the text, name or attribution, and date that visitors would see. Default to private, provide a way to revoke visibility, and exclude unpublished reflections at the API/database level. Build this only after reviewing consent and moderation needs; do not surface private reflection data in the public landing-page response.

---

## Product update email campaign

**Status:** campaign copy and eligible-audience export prepared; no mass email has been sent. The original signup did not collect product-email consent, so existing registration addresses must not be enrolled automatically. Suspended accounts and users who switch the preference off are excluded. Opt-in is available on signup and the dashboard profile. Existing PostgreSQL databases gain `marketing_opt_in_at` automatically at startup; a null timestamp means unsubscribed.

An admin can check the eligible count and download a CSV in Admin → **Product email audience**. For a one-time or scheduled send, import that CSV into a dedicated Brevo marketing list, use the short copy in [docs/product-update-email.md](docs/product-update-email.md), include Brevo's unsubscribe link, preview and schedule the campaign. Re-export immediately before sending, replacing the list and honoring Brevo's own unsubscribes. The existing transactional Brevo API is reserved for account and booking mail. Automatic recurring campaigns and recipient synchronization with Brevo are not implemented; a fresh export is required for each send.

---

## AI Planner — Delivery Plan

**Status correction (November 2026):** planner routes, deterministic metrics, a Groq client and a planner page now exist in source. Saved analysis storage/history remains a follow-up. The numbered delivery steps below are the earlier design proposal, not a current unimplemented-feature checklist. `L2E_PLANNER_PROMPT_v3.md` is the current reference prompt in this repository. It describes one person's history and must be adapted into a reusable, user-scoped planner before integration. The first release can use the existing timetable and reflections without waiting for automated scheduling.

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

Skills have clear Edit and Remove controls with accessible labels and a responsive action layout. The Availability page shows weekly slots in a day-by-day presentation with a slot count, clearer Remove actions and an encouraging empty state. Availability reads offer explicit retry on failure; writes are never automatically retried. The redesigned editor uses a saved revision to reject conflicting writes from stale tabs. The availability API now checks row scan and iteration errors and returns time values consistently as `HH:MM`.

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

## Product Roadmap & Features Yet to Be Implemented

**Historical product backlog:** several modules below now have handlers, routes and tests. Reconcile each requirement against current source before implementation; module existence does not prove every requirement is delivered. The redesign roadmap at the top is the current execution checklist. Parallel worker execution is not required for this documentation or redesign workflow.

### 1. Add Reflection: OPay-Standard UI (Seamless Text vs. Voice Switching)
- **Context & Current Gap:** Currently, reflections support text entry and an initial audio upload pipeline, but the reflection modal lacks a unified, high-standard interaction flow. Voice recording appears as an unstyled button next to a text label without dedicated recording states, audio waveform visualization, or mode selection.
- **OPay UI Standard Requirements:**
  - **Segmented Pill Mode Switcher:** Top of the modal features a high-contrast segmented pill switch:
    - `[ ✍️ Write Reflection ]` vs. `[ 🎙️ Speak Reflection ]`
    - Styled with OPay's signature vibrant emerald green accent (`#00B875` / `#059669`), crisp border radii, smooth tab transition animations, and dark-mode compatibility.
  - **Voice Mode Experience:**
    - **Prominent Mic Action:** Large central circular record button with tactile tap feedback and clear visual states (Ready, Recording, Reviewing).
    - **Pulsing Ripple & Waveform:** Concentric pulsing ripple effect or live equalizer bars indicating active audio input.
    - **Live Duration Timer:** Real-time recording counter (e.g., `00:18 / 03:00 max`).
    - **Post-Recording Audio Preview:** An in-modal audio player preview with scrubbing controls, enabling users to listen before saving.
    - **Re-record & Discard:** One-tap action to discard the take and record again without losing modal context.
    - **Direct Upload Integration:** Seamless upload to `/api/v1/voice/upload` with automatic attachment of `audio_url` to the reflection log payload.
  - **Text Mode Experience:**
    - Clean, spacious inputs with crisp focus rings, character counters, and guided prompts for Activity, Challenges, and Key Learnings.
  - **Mobile-First Ergonomics:**
    - Bottom-sheet capable modal presentation on mobile viewports (`max-w-md`), accessible keyboard trapping, Escape key dismissal, and non-blocking background backdrop.

### 2. Reflection Data Safety & Timetable Cascade Decoupling
- **Context & Current Gap:** In `schema.sql` and `schema.sqlite`, `activity_logs.timetable_id` is defined with `ON DELETE CASCADE`. When a user deletes or modifies a weekly timetable slot, historical reflection entries (and associated streaks/heatmaps) for that slot are wiped.
- **Requirements:**
  - Alter the foreign key constraint to `ON DELETE SET NULL`.
  - Add an `activity_name VARCHAR(255)` column to `activity_logs` as an immutable snapshot captured at reflection creation time.
  - Update Go reflection handlers (`handlers/timetable.go`) to preserve and display reflections independently of active timetable blocks, ensuring historical learning records and streak calculations remain permanent.

### 3. AI Growth Coach & Weekly Review Engine (`L2E_PLANNER_PROMPT_v3.md`)
- **Context & Current Gap:** `L2E_PLANNER_PROMPT_v3.md` outlines an intelligent reflection evaluation model, but the backend AI execution pipeline, database storage, and front-end coaching view are not yet implemented.
- **Requirements:**
  - **Deterministic Metrics Engine (Go):** Compute objective weekly stats (scheduled hours, completed reflection hours, reflection adherence rate, streak gains/losses, consistency index).
  - **LLM Coaching Integration:** Adapter calling a model provider (Gemini / Claude / OpenAI) via server-side `LLM_API_KEY`. Adapt `L2E_PLANNER_PROMPT_v3.md` to evaluate the user's weekly activities, isolate bottlenecks, celebrate wins, and ask a single high-leverage coaching question.
  - **Data Persistence:** Migration adding `planner_analyses` table (`user_id`, `week_start`, `metrics_json`, `ai_review_text`, `created_at`).
  - **Phone-First `/planner` Interface:** An on-demand dashboard to view weekly breakdowns, trends over time, and past coaching evaluations.

### 4. Proof-of-Work Public Learner Portfolio (`/@username`)
- **Context & Current Gap:** Fellow profiles are currently private or limited to tutor discovery. Students lack a public, shareable showcase of their verified learning track record.
- **Requirements:**
  - Publicly accessible fellow route (`/@username` or `/u/:username`).
  - Showcases verified learning hours, interactive reflection heatmap, mastered skills with peer reviews, completed collaborative projects, and authored guides.
  - User-configurable privacy settings (toggle public visibility on/off).

### 5. Virtual Silent Focus / Co-Working Rooms (Live Pomodoro)
- **Context & Current Gap:** Solo studying often suffers from isolation and lack of focus.
- **Requirements:**
  - Synchronized Pomodoro focus rooms (`/focus`) with structured 25-minute focus and 5-minute break cycles.
  - Live participant roster showing who is currently focusing and their session target.
  - Ambient audio cues and one-click post-session reflection logging.

### 6. Peer Accountability Buddy Matching & Nudges
- **Context & Current Gap:** Habits are more effectively maintained when paired with peer accountability.
- **Requirements:**
  - Opt-in matching algorithm pairing fellows with similar study schedules and target skills.
  - Automated Discord and Brevo transactional email alerts when a study partner logs a milestone or misses their scheduled reflection.
  - Shared streak badges celebrating mutual accountability.

### 7. Real-Time In-App Notifications (Server-Sent Events)
- **Context & Current Gap:** All current notifications rely on Discord webhooks or Brevo transactional emails. The in-app experience has no real-time push mechanism.
- **Requirements:**
  - Lightweight SSE endpoint (`GET /api/v1/notifications/stream`).
  - Header notification tray with unread badge counter.
  - Real-time alerts for booking confirmations, project join requests, tutor reviews, and guide comments.

### 8. Community Group Masterclasses & Workshops
- **Context & Current Gap:** `/masterclass` currently only supports 1-on-1 tutor booking.
- **Requirements:**
  - Tutors can create group learning sessions with fixed date/time, topic agenda, and student seat limits (e.g. 10 attendees).
  - Multi-student enrollment with real-time seat availability tracking.

---

## Known Issues / Gaps

Ranked roughly by how much they'd block real usage:

> **Recently resolved:** `timetable`, `activity_logs`, and `users.is_admin` are now defined in both `schema.sql` and `schema.sqlite` (kept in sync, dialect-correct). Fresh databases created from either file fully support `/api/v1/timetable`, `/api/v1/reflections`, `GetProfile`, and `AdminUsers` — this also unblocks the AI Planner work above. ⚠️ **Existing databases are not changed by these files:** `CREATE TABLE IF NOT EXISTS` adds the new tables but won't add `is_admin` to an existing `users` table. Apply it manually — Postgres: `ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT FALSE;` plus the two tables from `schema.sql`; SQLite: `ALTER TABLE users ADD COLUMN is_admin INTEGER DEFAULT 0;` plus the tables from `schema.sqlite`.

> **Resolved and checked live (2026-09-24):** Render logs showed PostgreSQL `08P01` result-format mismatches and `26000` missing unnamed prepared statements across Projects, Collaborators, Timetable, and Reflections. The shared driver now avoids that prepared-statement protocol. After the owner opened the deployed pages, each affected endpoint returned HTTP 200 and no new query errors or HTTP 500 appeared in the observed logs. This confirms those requests during the check; monitor later traffic for recurrence.

1. **Account-specific login throttle is in place; broader IP-based abuse controls are still possible.** Shared or rotating identities can still generate aggregate load.

## Suggested Next Steps

1. Manually redeploy/restart the merged commit and record production smoke results.
2. Complete the current release's physical phone/laptop and assistive-browser acceptance.
3. Choose the next scoped product batch from the **Remaining features and acceptance** table above. Existing weekly metrics, Groq coaching, focus rooms, portfolios, notifications, buddies and group masterclasses already exist and must not be reimplemented from the historical proposals.

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
- `LLM_API_KEY` — server-side key for the optional Groq planner; `GROQ_API_KEY` is the fallback. `GROQ_MODEL` optionally selects the model. Metrics remain available without a provider key.

### Useful commands

```bash
go mod download
go mod get <package_url>
```

### Daily workspace integration (tasks 6–10)

The first five integration tasks now use the real Dashboard/Timetable templates: shared semantic controls and tokens, one grouped navigation partial across 13 workspace screens, read-only daily cards from existing account APIs, setup/collaboration below daily actions, and responsive phone agenda/desktop week views. See [implementation and verification notes](docs/redesign/workspace-integration.md). The standalone sample preview remains available. No database, permission, booking, or reflection API contract is changed. Production deployment and physical-device acceptance remain unchecked.

### Reflection reliability (tasks 11–15)

Selected timetable day is retained across create/delete/reflection saves and verified by browser fixtures. The required shared-navigation, Dashboard and Timetable browser checks run in CI. Write/Speak now share persistent labelled text fields, keyboard tabs and accurate recorder/upload/save states. Failed uploads retain a retryable local take; failed saves retain text and attachment. Drafts are isolated by activity/date in current-page memory; late microphone/upload callbacks cannot attach to a reopened different reflection. No automatic transcription is claimed. See [reflection reliability notes](docs/redesign/reflection-reliability.md) for evidence and boundaries.

### Reflection history, dialogs and privacy (tasks 16–20)

Dialogs now use the visible viewport with fixed action areas, scrollable fields, Escape and focus restoration. History adds search/date/format filters, detached-block labels and private-audio retry. Upload and playback routes require sign-in and file ownership; saved recordings cannot be attached across accounts. Reflection rows have app-only PostgreSQL RLS/grants. CI includes a browser journey through real Gin handlers, isolated PostgreSQL, temporary audio storage and two authenticated accounts.

The source retention rule and legacy migration are covered by tests. Deletion verifies the connected database's retention constraints before enabling confirmation. Independent verification of the correctly connected `l2e-studylink` database is complete as of 22 November 2026; its SET NULL parent/snapshot checks passed and the reflection access policy was applied and verified. Production UI smoke and physical-device acceptance remain pending. See [evidence and verification](docs/redesign/reflection-history-privacy.md).

### Tutoring and bookings (next five tasks; retention carryover closed)

The connected `l2e-studylink` database was verified on 22 November 2026: the reflection parent is nullable, the activity snapshot is non-null and the validated foreign key uses SET NULL. The previously delivered reflection access policy was applied as `studylink_private_reflection_access` and rechecked: RLS enabled and no anon/authenticated/PUBLIC grants. No reflection rows or other project tables were changed.

Tutor discovery now has labelled skill/date/availability filters, scan-friendly cards and a request review showing tutor, date, published time, shared booking timezone and meeting type. Tutoring hours are clearly separate from personal study blocks. Session cards expose all five statuses and only server-permitted actions. Booking creation validates active users and published hours, rejects past/self/overlapping requests, and serializes concurrent requests for both participants. Existing participant notifications and review flows are preserved. See [implementation and verification](docs/redesign/tutoring-bookings.md).

### Deployment build/start verification

The Docker image and CI use Go 1.26.1. `sh scripts/build.sh` builds the package; `sh start.sh` executes the compiled server. Docker, Blueprint and Procfile share these entrypoints. CI boots the built image against isolated PostgreSQL and checks health, templates and assets. Botkeep startup logs later confirmed exhausted build storage. The Render backend is live at https://l2e-studylink.onrender.com; its health endpoint returned HTTP 200 on 23 November 2026.

### Vercel frontend and Render backend

- [x] Add Vercel static-asset build and reverse-proxy routing for the existing Go-rendered pages/API; preserve server-side guards, private recordings and streaming routes.
- [x] Add exact `PUBLIC_ORIGIN` support with origin-validation regressions; never trust arbitrary forwarded hosts or wildcard preview domains.
- [ ] Verify the deployed Vercel origin, cookie writes and public pages after the backend update.

Set `PUBLIC_ORIGIN=https://l2e-studylink.vercel.app` on Render. All database, JWT, email and AI secrets stay on Render; Vercel needs none. `sh scripts/build-vercel.sh` copies public assets only into `vercel-public`; all other paths forward to Render without rewrite caching. Protected HTML stays server-rendered rather than exported publicly. The secondary project alias redirects to the canonical frontend address. Production preview hostnames are not trusted for cookie writes.

The initial Vercel project is deployed through the connected API because Vercel reported missing GitHub integration access. Automatic source deployments require granting Vercel access to `eeyung1/L2EStudyLink`, then selecting Other framework, repository root, build `sh scripts/build-vercel.sh`, empty install command and output `vercel-public`. Render continues automatically deploying merged main commits. Existing voice files still require durable storage/migration; local free-service uploads are not durable.
