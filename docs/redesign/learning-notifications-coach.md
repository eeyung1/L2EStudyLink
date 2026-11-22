# Learning Hub, notifications and weekly coverage

This batch delivers the next five unchecked redesign roadmap items after PR #71.

| Item | Delivered behavior and boundary |
| --- | --- |
| Learning Hub hierarchy, replies and moderation feedback | Search/category filters, dated guide cards, readable article and discussion hierarchy, public posting labels, one-level replies, retained in-page drafts, independent article/discussion retries and explicit admin removal confirmation. Existing server role, suspension, reply-target and rate-limit rules remain authoritative. |
| Notification tray, unread states and record navigation | Token-based phone sheet/desktop dialog, native keyboard containment/Escape, focus restoration, explicit Read/Unread labels, mark-read/all, pagination, live updates and retry. Existing booking event keys produce links to their session card without a trigger/schema migration. Other existing event destinations remain unchanged. |
| Long content, empty lists, permissions and partial failures | Browser fixtures exercise long project/article/discussion/notification/coaching content, filtered empty projects/guides, empty requests, full classes, independent collaborator/discussion failures and recovery. Real two-account journeys verify moderation, rate limiting, guest denial, notification ownership and session navigation. |
| Weekly coverage, daily breakdown and coaching layouts | Shared workspace components, week/timezone caption, readable summary cards and seven dated daily cards with text values alongside decorative bars. Generated suggestions occupy a separate section with disclosure and existing request controls. No new coaching provider or analysis-history feature. |
| Honest time estimates | Summary labels and daily values say estimated scheduled/reflection-covered minutes. Copy states these are not verified hours worked or skill mastery, use the current recurring timetable and change when the timetable is edited/deleted. Missing reflections do not prove no learning happened. |

## Existing rules preserved

Guides and discussions are public. Comments use plain text and the existing 200-comment API cap; guide discovery filters the existing 40-guide response. Admin removal cascades to replies under the current foreign key and cannot be undone. Only server-authorized admins can remove comments. Drafts remain in page memory across retries or reply switches; reloads do not preserve them. Successful posting clears its draft even if refreshing the discussion subsequently fails, preventing a misleading retry of a successful write.

Notifications remain scoped to the signed-in account. Reading an update is separate from following its safe same-origin destination. Following a link attempts a read mark with keepalive and still navigates if marking fails; Refresh obtains the authoritative state later. Existing SSE events, pagination and all-read snapshot cutoff remain intact. No new event coverage or external delivery is advertised. The tray can retry an initial widget fetch failure.

API fields, duration rules and data schema are unchanged. A discovered weekday bug is corrected: the calculator now uses the timetable’s stored Monday=0…Sunday=6 values directly, so dated daily rows and same-day reflection coverage agree with the timetable. Tests cover all seven stored values, Monday/Sunday reflections, overnight blocks and existing date/timezone boundaries. Coaching still requires the existing provider configuration, explicit user request and quota. Reflection content is not included in notifications. Dedicated provider/rate-limit/empty-week/truncation presentation and planner privacy acceptance remain separately tracked next items; existing error messages and truncation disclosure are preserved here.

## Verification

- Local JS syntax checks, existing JS unit suite, source inventory and whitespace checks.
- CI Go tests/build/vet, SQLite schema, container build/boot/health/templates/assets.
- `tests/ui/learning-coach.mjs`: 320/390/768/1440 widths in both themes, Axe WCAG scans, filters, long text, partial failures, draft retry, moderation visibility, estimated metrics, safe AI text, notification bounds/keyboard/read/retry/link fallback.
- Extended `tests/ui/community.mjs`: long project cards, independent fellow failure, empty inbox/filter state, buddy retry and full-class actions.
- `TestLearningCoachBrowserJourney`: actual Gin/PostgreSQL, guest/member/admin contexts, reply/rate-limit/moderation cascade, account-specific 60/90-minute weekly reports, coaching via a local fake transport (no external provider calls), notification read/record navigation and account isolation. Existing community/booking/reflection server journeys remain gates.
- Screenshots and JSON evidence are uploaded in `redesign-foundation-evidence`.

No production records or hosting settings are changed. Physical phone/laptop acceptance and production smoke remain unchecked. Landing and login/signup/recovery remain included and pending in Batch 6.
