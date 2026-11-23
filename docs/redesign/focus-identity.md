# Weekly coaching states, focus and learner identity

This batch completes the five remaining Batch 5 roadmap items after PR #72. Landing/account/admin redesign and release acceptance remain pending in Batches 6 and 7.

| Item | Delivered behavior |
| --- | --- |
| Coaching availability and limit states | Explicit ready, unconfigured, empty, loading, provider failure, quota and success states. Failed coaching keeps metrics visible. A quota response uses Retry-After to disable repeat requests within the page; the server remains authoritative across reloads. Bounded/truncated context has a prominent disclosure next to the result. No automatic provider retries are added by the UI. |
| Planner author/privacy separation | The generic server prompt is the only system message. Selected-account, selected-week, completed reflections form the user context. Real transport assertions exclude other-account, other-week and incomplete-entry canaries. No personal planning reference file is loaded into requests. |
| Focus room flow | Shared tokens, labelled lobby/create/join controls, loading and failure retry, private member roster, server-anchored focus/break timer, reconnect disclosure, optional chime, target editing, native leave confirmation and explicit reflection handoff. Leaving does not record a session as completed or write a reflection. |
| Learner portfolio | Distinct owner controls and visitor view, validated/conflicting handles, confirmation for publication and public URL changes, clear shared-data disclosure and revocation feedback. Private/suspended portfolios are unavailable publicly; owner operations check current suspension. No-store responses avoid stale visibility caching. |
| Own-record verification | Isolated PostgreSQL/pgx, two signed-in accounts and an anonymous visitor exercise account-specific metrics/context, consent/roster/timer behavior, explicit reflection save, handle conflicts, publication and revocation. |

## Claims and boundaries

Portfolio session hours are scheduled durations of sessions marked completed by the tutor, not measured work, independently verified attendance or proof of mastery. `scheduled_session_hours` and `hours_basis` are additive API fields. The legacy `verified_hours` numeric key remains for API compatibility but the interface no longer calls it verified. Skills are self-reported. Closed recruitment does not establish project completion; project labels now say so. Publishing exposes reflection activity dates/counts and streaks alongside profile, skills, reviews, durations and project participation, while reflection text/audio remain private.

Portfolio visibility uses the existing account opt-in; no new schema or production data migration. Unpublishing prevents future API reads; it cannot recall copies already made by visitors. Public URL changes are explicit. API error handling now recognizes production pgx uniqueness errors as handle conflicts as well as the older test driver.

Focus rooms remain process-local and ephemeral. Restarts clear them; multi-instance synchronization requires existing affinity/shared storage work outside this batch. Timer/roster updates are server anchored; disconnected countdowns are clearly labelled as potentially out of date. An optional chime requires user opt-in. Membership and elapsed time do not prove study or create a reflection. The handoff asks the user to choose their own timetable block and explicitly save a reflection.

Coaching still uses the configured Groq provider and existing three-attempt, per-process quota. Failed provider calls consume attempts. No saved coaching history, new provider or automated monitoring is added. Privacy tests use synthetic canaries and a local fake provider transport; no external model call is made by CI.

## Verification

- Existing Go tests/build/vet, JS tests, SQLite schema, source inventory and compiled-container boot gates remain.
- `tests/ui/focus-identity.mjs`: light/dark, 320/390/768/1440 widths, WCAG Axe checks, coach failure/quota/empty/truncation, room retry and leave cancellation/failure/handoff, portfolio handle conflict and publication/revocation, safe public content and long text.
- `TestFocusIdentityBrowserJourney`: real Gin, production pgx and an isolated schema; selected-account coaching transport checks; server quota/provider failure; two learners joining one room and receiving the same accelerated focus-to-break transition; consent-preserving leave and explicit save on the learner's own timetable; public/private portfolio and reflection-content isolation. Test-only time controls never enter application routes. The accelerated fixture uses a longer presence lease so a time jump does not expire membership; production lease settings are unchanged.
- Existing learning/community/booking/reflection browser journeys remain merge gates.
- CI uploads JSON evidence and screenshots in `redesign-foundation-evidence`.

Production smoke, physical phone/laptop acceptance and unavailable-browser coverage remain unchecked. Hosting and production databases are not accessed in this batch.
