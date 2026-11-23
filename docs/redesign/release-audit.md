# Redesign release audit

This batch completes the final Batch 6 server/public-boundary item and the first four automated release-audit items in Batch 7. It does not claim production acceptance or completion of physical-device testing.

## Five delivered checks

1. **Authorization and public data.** `TestReleaseAuthorizationAndPublicBoundary` constructs the same complete router as the server, using an isolated PostgreSQL schema through pgx. Every registered admin endpoint is checked with no session, a regular member, a suspended admin, a demoted admin and a deleted admin. New admin routes must be added to the matrix. Active admin reads must succeed and disable caching. Canary reflection text, recording URLs and password hashes must be absent from landing/learning/search/tutor/public-portfolio responses; private reads require sign-in, reflection reads remain account-scoped, and portfolio revocation takes effect. Existing per-feature authorization tests remain required. Explicitly published comments, bios and portfolio aggregates retain their documented public behavior.
2. **Viewport and accessibility matrix.** `tests/ui/release-audit.mjs` covers 22 representative public/account/workspace/admin routes in both themes at 320, 360, 375, 390, 412, 480, 768, 820, 1024, 1280, 1440, 1920, 2560 and 3840 CSS pixels. It asserts no horizontal overflow, mobile header clearance, form text at least 16px and primary button dimensions. Each view receives a phone WCAG 2.1 A/AA main-content scan. Existing suites cover populated records, long text, dialogs and state-specific scans. Shared navigation is also checked by the existing workspace suite.
3. **Load and recovery states.** Existing dashboard/timetable baseline state coverage is supplemented by explicit unavailable/offline/retry/empty settings states, a held-response slow-read state that disables edits, failed-write draft retention and expired-session navigation for eight signed-in route families. Held fixture requests are released explicitly; no production API or email is used. Existing feature suites cover booking, reflection, coaching-provider, community and portfolio state variants.
4. **History and repeated actions.** Project deep links restore the correct record on browser Back and Forward. Profile forms reject a second submit while a save is pending, accept a later deliberate save, retain a failed draft and unlock after success. Existing timetable, session/review, signup/recovery and native-confirmation tests continue to run.
5. **Reflow and keyboard.** Every representative view is checked in the 720×500 CSS-pixel content area corresponding to 200% reflow of a 1440×1000 desktop content area, with visible keyboard focus. Native confirmation checks cover initial focus, keyboard containment, Escape cancellation and return to the initiating button. Existing suites test mobile-menu focus and reflection dialogs under a simulated software keyboard. This is automated reflow coverage, not a claim that Chrome toolbar zoom, an assistive reader, Safari or a physical phone has been manually tested.

## Changes arising from the audit

- Production route construction is now reusable by tests. The server still uses Gin, the same middleware/handlers and inline SQL; no API or schema is replaced.
- The generic `/page/:name` route permits known page templates and returns 404 for unknown names and internal partials. It no longer attempts to render arbitrary/missing templates. HTML shells contain no private account records; API middleware is the data boundary.
- Canonical planner/focus/buddy/masterclass HTML navigation with a missing or expired session now redirects to login instead of showing raw JSON. HTML and API guards share token validation; API authorization and same-origin cookie-write rules remain unchanged.
- Admin middleware marks successful and denied responses `private, no-store`, including permission failures before handlers run.
- Native confirmations now explicitly cycle Tab/Shift+Tab at their first/last controls, avoiding focus escaping to browser chrome. The shared behavior also covers focus, booking, portfolio and article moderation dialogs.
- The expanded scan found low-contrast dashboard next-step captions and profile labels. They now use the shared theme-aware secondary-text token. Further corrections are recorded in the PR.

## Run and inspect

The normal CI workflow runs `go test ./...`, build/vet, JavaScript unit tests, SQLite schema validation, all fixture browser checks, real Gin/PostgreSQL browser journeys and compiled-container boot. `TEST_DATABASE_URL` must identify a disposable test database; the authorization test creates and drops only its own schema. The container job uses its own database. No live hosting or production data is accessed.

```bash
npm ci --ignore-scripts
npx playwright install --with-deps chromium
L2E_UI_SUITE=release npm run test:ui
TEST_DATABASE_URL='postgres://test:test@localhost:5432/l2e_test?sslmode=disable' go test ./... -run TestReleaseAuthorizationAndPublicBoundary -count=1
```

The full browser run is `npm run test:ui`. Evidence is generated under `test-results/ui/evidence.json`, (release-only runs write `test-results/ui/release/evidence.json`), including per-view/theme/width observations and state/interaction records, with representative screenshots. CI uploads the evidence artifact. Fixture data is fictional and separate from real-server integration tests.

## Remaining acceptance

- Owner redeploy/restart, then record production smoke against the merged commit.
- Physical Android and laptop review, actual browser toolbar zoom and screen-reader reading order/labels; iPhone/Safari where available.
- Record acceptance and any remaining limitations before declaring the full redesign complete.
- Existing limits remain: unpaginated admin records, full-list tutoring availability writes, ephemeral unsaved drafts/coaching and reliance on configured transactional mail. The audit does not make these new product features or verify email delivery.
