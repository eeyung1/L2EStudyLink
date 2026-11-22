# Reflection reliability: tasks 11–15

## Five-task analysis

| README item | Finding | Implementation and evidence |
| --- | --- | --- |
| Selected timetable day after create/delete/reflection | Existing reloads already preserve a variable, but reflection saves had no browser regression case. | Kept existing selection behavior and extended actual handler-driven fixture checks through create, delete and successful reflection saves on a non-current day. |
| Browser checks before merging shared navigation/dashboard/timetable | The previous integration delivered most required checks. | Retained them as a CI gate and completed reflection-mutation coverage; the roadmap item is now checked on that evidence. |
| Unified Write/Speak | Speak hid the written summary even though save required it; both tabs were in the normal tab order. | Both modes retain visible shared labelled fields, counters, selected/focus states and Arrow/Home/End tab navigation. Voice attachment status is visible in either mode. |
| Accurate recording/upload/save states | Recorder permission could be requested twice, elapsed time counted timer callbacks, and saving was possible while upload was pending. | Explicit permission/recording/stopping/uploading/retry/attached states, elapsed-time cap, local playback and duration fallback, same-take upload retry, atomic save lock and clear status. |
| Recoverable failures | Re-recording deleted prior text/transcript; uploads had no retry and late callbacks could cross activity boundaries. | Local blob and written fields retained on upload/save failure; prior take kept until replacement recording succeeds; abort plus generation checks isolate callbacks; permission and unsupported-browser recovery preserve text. |

## Draft and lifecycle rules

- Drafts are keyed by timetable block ID and reflection date, and remain in this signed-in page's memory. They are not written to localStorage, shared caches, analytics, or a new database table.
- Switching Write/Speak does not discard text or attached audio. Switching to Write while recording stops and uploads the take; cancelling a pending permission request invalidates its callback.
- Closing a dialog cancels an active recording, pending permission and upload request. It retains existing text and the last completed local take in the page's draft map. Reopening that activity restores the draft and permits retry. The visible hint explains that an unfinished recording is cancelled.
- Closing during a reflection save is blocked until the server responds. Repeated Save is disabled. Only a successful save clears the draft; failures keep fields and attachments editable.
- Refresh, navigation, sign-out or a page unload clears unsaved drafts. This batch does not add durable draft storage or change session redirects. It is not a promise of recovery after leaving the page.
- The existing upload API returns placeholder `text`, not a real transcript. The client ignores it and explicitly asks for a written summary. No new transcription provider or API contract is introduced.
- Cancelling upload cannot undo a file already received by the server. This batch does not add file deletion, change upload storage authorization/retention or reconcile historical timetable deletion copy. Those remain separate roadmap work.

## Verification and limits

`node --test static/js/*.test.cjs`, Go test/build/vet, template execution and SQLite schema checks remain required. Browser checks in `tests/ui/workspace.mjs` cover shared navigation, Dashboard and Timetable plus selected-day mutation behavior. `tests/ui/reflections.mjs` covers both themes with synthetic permission/MediaRecorder events, a playable generated WAV fixture, upload retry/timeout, late permission and upload callbacks, text isolation between activities, failure/retry, a three-minute elapsed-time stop, a locked save and unsupported recording with text-only save. Four widths and both modes are scanned for WCAG 2.1 A/AA and captured in the CI artifact.

These are localhost browser fixtures with intercepted, in-memory writes. They do not record a user's real microphone or claim physical-device, Safari, production storage, or authenticated Go-server end-to-end acceptance. The source inventory is regenerated for the shared fields and new controls. Results live in `evidence.json` alongside screenshots in the Actions artifact.

The next unchecked task is dialog/sheet behavior with small screens and on-screen keyboards, followed by reflection history, account privacy and retention/deployed-migration verification. Basic existing focus containment and Escape behavior are preserved; full keyboard/device acceptance remains unchecked.
