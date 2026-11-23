# List pagination and editing safety

The owner selected five improvements after the redesign implementation and automated audit were complete: admin account pagination, admin booking pagination, Learning Hub pagination, availability edit-conflict protection and unsaved-work warnings. No schema migration, external message, live hosting access or production record edit is included.

## Delivered behavior

1. **Admin accounts:** the redesigned dashboard requests 20 accounts at a time, ordered by account ID, with server search and active/suspended/admin filtering. Load more appends deduplicated records. A failed next page retains records and the cursor for an explicit retry. Filter changes reset paging and stale search responses cannot replace newer results.
2. **Admin bookings:** the same bounded controls load newest booking IDs first. Tutor/learner/ID search and status filtering run on the server across the full collection. Server admin middleware remains required for each page.
3. **Learning Hub:** guide discovery requests 20 newest-added guides at a time, searches the full collection and obtains categories independently of the current page. Article discussion loads 20 newest comment IDs at a time, retaining loaded entries on next-page failure. Older pages can complete threads whose parent is outside the current slice; existing explicit orphan-thread copy remains in use. Publishing a comment refreshes the newest discussion slice. Content remains plain text and existing moderation/posting limits apply.
4. **Availability conflicts:** reads can return `{slots, revision}`. The editor sends that account-scoped content revision through `If-Match`. The server locks the account row, checks the current committed snapshot and replaces slots in one transaction. Stale writes receive 412 before any deletion. The editor disables writes, keeps the proposed form entries and offers an explicit reload/retry. Changes from other tabs are never automatically merged or silently overwritten by the redesigned editor.
5. **Unsaved work:** profile/skill/availability, reflection/review/discussion, project/masterclass creation and guide-publishing drafts prompt before page-link navigation. Browser refresh/close uses the browser's native before-unload warning when supported. Stay keeps the page; Leave deliberately discards. Successful saves and explicit form resets/cancels clear the relevant warning. Reflection/review maps retain their existing in-page lifetimes. Logout asks before starting the server request; expired sessions bypass the warning so recovery is not blocked. Draft content is never written to storage by this feature.

## Compatibility and limits

- Existing list responses are preserved when `paged=1` is absent. Opt-in list calls accept `limit` (1–100, default 20), `after` (positive last ID), and relevant `q`, `state` or `category` filters. Responses add `has_more` and `next_cursor`; admin responses use `items`, public responses retain `articles`/`comments`. No new route bypasses existing authorization.
- Pagination uses stable ID keysets rather than shifting offsets. Lists are live, not immutable snapshots; fresh records become visible on refresh. Newest-added ordering uses IDs; old public calls retain their existing publication-date order/caps. Legacy unpaged admin consumers remain unbounded until migrated.
- `GET /availability` still returns an array by default; `?revision=1` returns the snapshot envelope and an ETag. Legacy writes without `If-Match` remain accepted for compatibility and therefore do not themselves receive stale-write protection. All writes share the account lock so the conditional editor checks the latest committed state.
- Warnings are not draft persistence, autosave or transcription. They cannot guarantee recovery after a browser crash, forced shutdown, session expiry or a browser that suppresses before-unload dialogs. In-page switches keep their existing discard/retention rules. Password/account-entry forms are not tracked. Cross-account scope changes bypass old-page warnings.

## Verification

- JavaScript syntax/unit checks, source inventory, SQLite schema, Go build/vet/tests and container boot remain required.
- `TestListSafetyFeatures` uses a disposable schema and the production pgx simple protocol. It walks 45 accounts/guides/comments and 44 bookings in small pages, verifies cross-page search, invalid inputs, admin denial and legacy responses, and races conditional availability writes.
- Its real Gin/browser journey verifies load-more failure/retry, server search beyond the first page, guide/comment continuation, both-theme responsive admin accessibility, two-tab conflict/reload/retry and Stay/Leave/save warning behavior. Existing fixture and real-server journeys still run. Their deliberate navigation defaults accept native leave dialogs; the focused safety journey explicitly tests cancellation and prompt counts.
- CI evidence is uploaded under `test-results/ui/`, including `list-safety-evidence.json` and representative screenshots. No transactional email is sent by these checks.

## Remaining work

The README's **Remaining features and acceptance** table is the current list. Release still needs owner-run production smoke and physical/device/assistive-browser acceptance. Saved coaching history, longer-term trends, personalized memory, automated matching/nudges, more notification event coverage, durable draft recovery, selected reflection publishing, automated campaign synchronization and broader traffic controls are separate future proposals. The five improvements above do not claim these are implemented.
