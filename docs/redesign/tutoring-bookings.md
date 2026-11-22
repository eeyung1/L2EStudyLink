# Tutoring and booking redesign

## Delivered scope

| Task | Implementation | Verification |
| --- | --- | --- |
| Tutor cards and skill/availability filters | Persistent labels, published-hours summaries, ratings/bio, date/availability toggle, self-profile link and safe DOM rendering. Errors retain filters and offer retry. | Both themes, four widths, injection-shaped fixture text and combined availability filtering. |
| Separate tutoring hours from study blocks | Search, availability settings and session pages explicitly distinguish published tutoring hours from the personal timetable, with direct links to each. | Existing availability/timetable APIs and records are unchanged. |
| Review selected tutor, date, time, timezone and meeting type | Native dialog shows a live request summary before submission. A public booking-clock endpoint returns the actual database timezone/date/clock used by existing naive date/time storage. Requests are labelled pending until tutor acceptance. | Review values, shared UTC clock vs a Lagos device, conflict retry, duplicate-submit lock, form preservation and real API request. |
| Validation/conflicts/self-booking/notifications | Active participants, valid future date/time/topic and exact published tutoring hours are checked server-side. Transactions take ordered advisory locks for both participants and reject overlapping bookings. Existing Discord/email calls remain after commit; durable participant notification triggers are retained. | Isolated PostgreSQL checks include concurrent competing requests, non-identical overlaps, unpublished/past/self/suspended requests, valid requests, acceptance/cancellation and durable notification counts. No production booking or external message is sent by tests. |
| Five session states and permitted actions | Pending, confirmed, cancelled, completed and no-show have explicit labels. The existing array response gains server-calculated action booleans and timezone; clients do not infer permissions from their device clock. Role/status filters and native action confirmation preserve the existing review workflow. | Both themes/four widths, filter/action/retry fixtures and server flag tests. Existing outcome/review authorization tests remain required. |

## Time semantics

The current schema stores `session_date`, `start_time`, `end_time` and weekly availability without offsets or per-user timezones. This batch does not reinterpret those stored clocks or migrate them to UTC instants. It exposes and labels the actual shared database session timezone. A device timezone is shown separately during discovery. A per-user timezone conversion/storage redesign would need its own migration and acceptance work.

Published availability is booked as a whole slot, preserving the existing selection model. Adjacent non-overlapping slots are allowed; pending/confirmed/completed records block overlaps, while cancelled/no-show records follow the existing conflict exclusion. Browser clocks are not authoritative for acceptance, cancellation or outcomes. Server checks still decide after every action, including races between a rendered card and a request.

The existing GET `/api/v1/bookings` response remains an array. `timezone`, `can_cancel`, `can_accept`, `can_decline`, `can_complete`, `can_no_show`, and `can_review` are additive fields, so existing dashboard consumers continue to work.

## Closed retention carryover and live access policy

On 22 November 2026, the correctly connected `l2e-studylink` project was inspected through catalog-only SQL. `activity_logs.timetable_id` is nullable, `activity_name` is non-null with an empty-string default, and the validated parent foreign key uses ON DELETE SET NULL. The user foreign key still uses ON DELETE CASCADE. No retention-model change or user-data write was needed.

The project had not yet received the source's app-only reflection access policy. `db/reflection_access.sql` was applied through the named Supabase migration `20261122151343_studylink_private_reflection_access`. Verification returned RLS=true, public-role grant count=0, parent nullable=true, snapshot non-null=true, validated SET NULL FK=true. Security advisors were inspected before and after. RLS-without-policy is intentional for this app-owned table: browser access goes through the authorized Go API; no public Supabase Data API policy is added. Other project schemas and grants were not changed.

This verifies the connected database's retention and access policy, not which Git commit a Render service is currently running or physical-device acceptance.

## Tests and boundaries

- Existing Go test/build/vet, JavaScript, schema and browser checks remain required.
- `booking_redesign_test.go` verifies actual PostgreSQL validation, concurrency, action flags and in-app notification effects with temporary fixtures and empty external notification credentials.
- `tests/ui/bookings.mjs` verifies responsive discovery/review/session states, accessibility and recoverable failures.
- `TestBookingBrowserJourney` and `real-bookings.mjs` use actual Gin endpoints, an isolated PostgreSQL schema and two session-cookie browser contexts. They search, request, accept and cancel, checking both participants and denying guest booking reads.
- The standalone fixtures and real-server evidence are uploaded with the existing UI artifact. Tests do not create production bookings or send messages to real people.

Post-session review redesign and broader empty/loading/error refinements remain the next unchecked task; existing reviews are preserved. Production UI smoke and physical phone/laptop acceptance remain unchecked.
