# Reflection dialogs, history, privacy and retention

## Scope and status

| Next README task | Implementation | Verification boundary |
| --- | --- | --- |
| Small screens, keyboards, Escape, focus | Header/body/footer layout; actions stay in the visual viewport; fields scroll independently; modal background is inert; Escape and trigger restoration remain available. Save completion focuses the selected-day control after a rerender. | Both themes and four widths; simulated VisualViewport height/offset events. Physical keyboards, Android/iOS device acceptance are not claimed. |
| History filters, labels, playback | Combined search/date/voice filters, clear/count/empty/error/retry states, calendar-date labels, captured activity names, explicit removed-block labels, same-origin audio controls and retry. | Browser fixtures verify safe DOM text, filter combinations, retry, actual WAV playback and no requests to invalid external media URLs. |
| Private account reflections | An active, non-suspended account is required for reflection reads/writes, both upload aliases and all audio GET/HEADs; new files carry owner-prefixed UUIDs, legacy files require a saved reflection owned by the caller. Only owned, existing local recordings can be attached. JSON/media use private no-store; media supports range requests and nosniff. PostgreSQL RLS and revoked public API roles restrict reflection table access to the Go app. | Real authenticated Gin/PostgreSQL tests reject guest/other-user reads and writes, attachment reuse, unowned legacy/orphan files, suspended/deleted users and unsafe URLs. An unprivileged DB role sees no private rows. Existing API and portfolio tests remain required. |
| Retention against current schema and deployed migrations | Existing SET NULL/history snapshot migration remains; no retention model change. Runtime policy query validates nullable parent, non-null snapshot and SET NULL foreign keys. Server deletion locks history, verifies policy and refuses unsafe schemas before deleting. UI changes copy/enables confirmation only after the database reports verified retention. | Fresh and legacy-schema tests, idempotent migration tests and real-server browser deletion demonstrate retention. Independent live-database metadata verification is still blocked by the current connection mismatch; the original roadmap checkbox stays unchecked. |
| Text/voice validation, retry, account separation | Existing lifecycle fixtures retained; server summary/date/attachment validation added; browser now saves audio/text through real HTTP/session cookies, real Gin handlers, temporary files and isolated PostgreSQL, then deletes the block and reads/plays retained history. | Two users and a guest in separate browser contexts. All writes are confined to temporary CI schema/storage. No production records or real microphone content are used. |

## Recording access compatibility

The public `/uploads` static mount is replaced with authenticated handlers at the same URLs. Existing UUID file paths still work for the user whose saved reflection references them; unlinked legacy files are denied because their owner cannot be proven. New names are `<user-id>-<uuid>.<audio-extension>` and are created with private file permissions. A file is not a publicly shareable link. No new database table is needed for ownership. Cookies authorize native audio; existing legacy bearer sessions still use the app's session-migration path.

`VOICE_UPLOAD_DIR` optionally selects private local storage and defaults to the existing `uploads` directory. The hosting service must still provide durable storage for durable audio; this batch does not move files to a new storage provider, recover missing files, or delete files after account removal. Missing audio has a visible retry explanation and does not hide the written reflection.

The SQL access-policy change is explicit: `activity_logs ENABLE ROW LEVEL SECURITY` and revocation from `anon`/`authenticated` are applied in the fresh PostgreSQL schema and the existing startup migration. App table owners retain PostgreSQL owner access. SQLite keeps application-enforced authorization and its existing retention model. There is no new public policy, privileged function, data backfill or retention-model mutation.

## Live connection verification on 22 November 2026

The connected Supabase project's catalog returned no `public.activity_logs` or `public.timetable`. The Render connector exposed a single Infotrust workspace without an established StudyLink deployment mapping. These are not evidence about the deployment's actual schema. No user data was queried and no external schema was changed. The owner was asked to reconnect/identify the correct account; independent production verification remains pending.

Once the correct deployment database is available, run the read-only `scripts/check-reflection-retention.sql`. Record its schema/nullable-parent/snapshot/foreign-key/RLS results and confirm the deployment uses the tested startup migration. Do not mark the independent deployment verification complete from CI results or a different project's catalog.

## Evidence

- `go test ./...`, build, vet and existing JavaScript/schema checks.
- `handlers/reflection_privacy_test.go`: authenticated ownership, safe attachment validation, range requests, legacy compatibility, retention safety and RLS.
- `tests/ui/history.mjs`: history and keyboard-viewport fixtures, both themes and four widths.
- Existing workspace/reflection fixtures: failure, timeout, selected-day and recording regression coverage.
- `TestReflectionBrowserJourney` with `L2E_BROWSER_E2E=1` in the UI CI job: actual Go/PostgreSQL/storage/browser journey, with `real-server-evidence.json` and screenshot added to the existing Actions artifact.

Production smoke, physical-device acceptance and the independent deployed-migration check remain pending. The four completed tasks can be merged without claiming that outstanding live verification.
