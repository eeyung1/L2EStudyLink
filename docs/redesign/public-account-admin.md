# Public, account and administration redesign

This delivery implements the first five checklist items in README Batch 6. The sixth item, the comprehensive server authorization and public-content audit, remains open. Release audit, production smoke and physical phone/laptop acceptance remain Batch 7 work.

## Delivered behavior

1. The landing page explains the actual plan → study → reflect journey through a keyboard-operable, labelled fictional example. It offers guest account entry, member workspace entry and public learning links, with explicit private-reflection and optional portfolio/coaching boundaries. No private reflection or fabricated member statistic is used.
2. Login, signup, forgot-password and six-digit reset screens share the reading design system, light/dark appearance, persistent labels and inline feedback. Failed requests retain the current form entries; repeated submission is blocked while pending. Password visibility has an accessible pressed state. Reset confirmation catches mismatches before submitting. Passwords are never saved in browser storage.
3. All account links carry the existing same-origin Learning Hub return allowlist. Login and successful reset retain the existing HttpOnly cookie session contract. Signup requires a deliberate transition to sign in. Optional product email consent starts unchecked. Recovery delivery still depends on configured transactional email; the UI does not claim delivery for an unknown address.
4. `/skills` provides account identity, bio/Discord editing, optional email preferences, skill add/edit/removal and a portfolio settings link. Names/emails remain read-only. `/availability` provides labelled, minute-precision weekly tutoring slots on the shared booking clock. A failed initial read disables editing until a successful retry; failed writes retain input. Removing a skill or slot requires confirmation. The dashboard profile save button now unlocks after successful saves as well as failures.
5. Admin records use responsive cards, account and booking filters, honest count/read failures and independently retryable sections. Sensitive content stays hidden until the account is verified as admin; existing server middleware remains authoritative. Suspensions/restorations, deletion, public guide publishing and comment removal require native confirmations. The current admin's own destructive controls are omitted in this UI. Audience export retains existing opt-in rules and sends no mail.

## API compatibility

No schema migration, framework, hosting or production-data changes are included. Existing endpoints are retained. `PUT /api/v1/admin/users/:id/suspend` additionally accepts `{ "suspended": true }` or `false` so a repeated confirmed request sets the desired state. A bodyless request retains the previous toggle behavior. Malformed/missing desired states in a provided body receive 400. Account/admin reads explicitly prohibit shared caching. Stats and booking rows now surface database errors rather than presenting incorrect totals or partially scanned records.

## Verification

- JavaScript syntax checks, existing 14 unit tests, source inventory audit and diff whitespace checks.
- Browser fixture coverage in `tests/ui/public-account-admin.mjs`: both themes and 320/390/768/1440 widths; screenshots and WCAG A/AA scans on phone/desktop; landing keyboard tabs; failed account request retention; recovery chaining; reset mismatch and session return; malicious return fallback; signup consent; profile retry; skill cancel; availability read/write failure and minute values; admin filters, cancel, mutation retry and unavailable access.
- An isolated PostgreSQL/pgx + real Gin browser journey in `TestPublicAccountBrowserJourney`: actual signup/login, HttpOnly cookies, article return, profile/skills and consent saves, one-use password reset, non-admin denial, explicit suspension retry/restore and legacy bodyless toggle. Reset codes are seeded in the disposable test schema; this does not verify real email delivery or send mail.
- Existing regression journeys and Go build/vet/tests, SQLite schema and compiled container boot checks remain required in CI before merge. Evidence is attached to the GitHub Actions run.

## Boundaries and follow-up

Live email delivery, production smoke and owner physical-device acceptance are not verified by fixtures. The owner manually restarts/redeploys hosting after merge. Public guides remain plain text and use existing moderation endpoints. Admin lists remain unpaginated and settings availability retains the existing full-list replacement API; larger datasets and concurrent-editor conflict handling need separate work. This batch does not claim the entire frontend redesign or release audit is complete.
