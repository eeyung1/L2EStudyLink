# Community and post-session redesign

This batch follows the next five pending roadmap items after PR #70. The landing and login/signup/recovery pages remain explicitly included in Batch 6 and unchecked until delivered.

| Roadmap item | Delivered behavior |
| --- | --- |
| Post-session reviews and session feedback | Labelled rating, comment limits, optional book-again preference, pending save, inline failure and retained in-page draft. Successful publication removes the form; server uniqueness remains authoritative. Session loading, filtered empty state and retry remain supported. |
| Projects | Discovery search/recruitment/owned filters; focused details beside the main flow, URL/back navigation, roles, commitment, safe repository links, team membership, applications and owner recruitment controls. |
| Collaboration | Explicit opt-in disclosure, disabled control until preference loads, independent retry, rollback on failed preference save, owner/application vs recipient/invitation controls and consequential-action confirmation. |
| Buddies | Separate incoming/sent/accepted states, recipient consent confirmation, shared activity streaks only after acceptance, honest counts and failure/retry feedback. No automatic matching or external nudges. |
| Masterclasses | One group-class catalog with a separate tutoring link, labelled search and host form, device timezone label, seat counts, RSVP, reservation cancellation and host cancellation with native confirmation. Existing capacity/role/time checks remain on the server. |

## Boundaries

Go/Gin, inline SQL, vanilla JavaScript and existing URLs/API shapes are retained. No schema migration or production data edits. Existing email preferences and notifications are unchanged. Project lists remain capped by the API (100 records); filters operate on that returned list. A closed project may become unavailable to a nonmember; the detail retry reports the API result honestly. Pending applications are visible in project detail, while the inbox retains its existing owner/invitee scope.

Review text is held only in page memory across filter changes and failed publication; reloads do not preserve an unpublished draft. Publication is a direct, labelled user action. Server rules allow only the student of a completed session and one review per session. Book-again consent is optional. Streaks summarize recorded activity and are not proof of hours worked.

Buddy summaries now release the relationship cursor before querying accepted-buddy streaks; this prevents a nested read from waiting on its own connection in a one-connection pool.

## Verification

- `node --test static/js/*.test.cjs`, Go tests/build/vet, SQLite schema check and source inventory check remain required.
- `tests/ui/community.mjs` exercises responsive layouts at 320/390/768/1440 CSS pixels in both themes; Axe scans, unsafe link/content rejection, preference retry, application draft retry, role controls, buddy consent, RSVP/cancellation/retry and review draft retry.
- `TestCommunityBrowserJourney` runs the actual Gin handlers against an isolated PostgreSQL schema with one connection and two cookie-authenticated users. The browser creates a project, opts in, invites/accepts membership, requests/accepts a buddy, hosts/reserves/cancels a class and publishes a review. Durable SQL outcomes, guest denial and non-owner project restrictions are asserted.
- Existing booking/reflection browser journeys and the compiled-container startup test remain merge gates.
- CI uploads screenshots and JSON evidence in `redesign-foundation-evidence`.

Production UI smoke, physical phone/laptop acceptance, and broader long-content/partial-failure audits remain unchecked. Hosting is not accessed or restarted in this batch; the owner redeploys merged main manually.
