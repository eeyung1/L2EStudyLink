# Redesign foundation audit and review guide

Baseline: `5e2309e14a98ed7f03aca5c8620b599cbdb1c581`. Scope: the first five unchecked tasks in Batch 1. This is an audit and an interactive layout preview, not a production dashboard replacement.

## Deliverables and evidence

| README task | Deliverable | Evidence |
| --- | --- | --- |
| Route, navigation, form/dialog and style inventory | `source-inventory.json`, generated from every current template and main/handler route declarations | `python3 scripts/audit-ui.py --check` rejects stale output. |
| Baseline states | Screenshots of unchanged Dashboard and Timetable templates at 390px and 1440px, for populated, empty, loading, error and expired-session fixtures | CI artifact `redesign-foundation-evidence`, including `evidence.json` and the baseline images. |
| Behavior/defect separation | Source findings and design proposals recorded separately below | Source files and baseline captures, with uncertainty explicitly recorded. |
| Dashboard/timetable previews | `/static/redesign/index.html`, using fictional, labelled sample data | Light/dark screenshots at 390px/1440px plus geometry tests across 13 widths. |
| Semantic tokens | `/static/redesign/tokens.css`, scoped to `.l2e-preview` | Shared theme, text, surface, accent, status, focus, type, spacing, radius, elevation, icon, touch and duration variables; accessibility scans on both themes. |

Open the preview from a running Go application at `/static/redesign/index.html`. It is a public static design study, has `noindex`, and uses no private account data. It has no API calls, authentication, database access or account writes. The **Open current app** link and expired-session sign-in link deliberately navigate to the existing app.

## Analysis of the five tasks

1. **Inventory first:** navigation and CSS are repeated across templates. A generated record makes the later shared-shell migration reviewable and identifies consumers before rules are removed. The file lists local and external assets; counts do not establish code quality or runtime authorization.
2. **Capture behavior:** screenshots must show more than a populated happy path. Both first-priority screens have five state fixtures. A 401 capture follows the existing session redirect to login; the preview presents its own explicit expired-state design. Coverage of other screens is reserved for their own implementation batches.
3. **Separate defects from preferences:** inaccessible destinations and missing error handling are different from a proposed layout. Do not label every older design decision a bug or treat a historical README statement as the current schema.
4. **Prototype the real priority:** show a daily plan, clear reflection/focus actions and weekly coverage before setup and collaboration content. Desktop shows a full sample week; mobile shows the selected day's agenda. The actions return an honest preview-only status rather than pretending to save records.
5. **Define a reusable language:** use readable emerald surfaces on light and dark backgrounds, consistent focus and touch geometry, and one icon family. Tokens are opt-in until the next integration tasks; production styles have not been globally replaced.

## Route and screen groups

| Workflow | Representative routes | Notes |
| --- | --- | --- |
| Public | `/`, `/learn`, `/learn/:slug`, `/u/:handle` | Public article and portfolio visibility remains controlled by their existing APIs. |
| Account | `/page/login`, `/page/signup`, `/page/forgot-password`, `/page/reset-password` | Generic `/page/:name` renders static shells; inventory is not an access-control endorsement. |
| Personal learning | `/dashboard`, `/timetable`, `/reflections`, `/planner`, `/focus`, `/portfolio` | Many legacy page shells are directly served HTML; API middleware remains authoritative for private records. |
| Learning with others | `/search-page`, `/my-bookings`, `/availability`, `/skills`, `/buddies`, `/masterclass`, `/masterclasses`, `/projects` | Role-sensitive decisions must retain backend checks when UI is integrated. |
| Administration | `/admin`, `/api/v1/admin/*` | Admin shell visibility is different from API authorization. |

The inventory includes all 22 existing HTML templates, their stylesheet/script consumers, links, control counts, forms and dialog/modal identifiers. Relative route declarations retain their original group spelling. API registrations in handlers are listed separately; dynamic routes and any future registrars need corresponding runtime coverage.

## Existing behavior and source findings

| ID | Finding and evidence | Classification | Follow-up |
| --- | --- | --- | --- |
| F01 | Dashboard sidebar contains the core study links but not every planner/focus/buddy destination; other templates have shorter menus. See `templates/dashboard.html`, `planner.html`, `focus.html`, `buddy.html`. | Source-confirmed navigation inconsistency | Next shared-navigation task; preserve role restrictions. |
| F02 | Dashboard leads with setup, ratings, skills and availability, then community discovery. There is no primary daily timetable agenda in its markup. | Existing behavior, not a functional defect | Proposed daily-action hierarchy in preview. |
| F03 | Multiple shared stylesheets and inline styles are combined differently by templates. See generated `styles` and `inline_styles` records. | Source-confirmed maintenance risk; actual conflicts require rendered evidence | Incremental token adoption and consumer tests. |
| F04 | `loadProfile()` parses the profile response without checking `res.ok`, and is invoked without a final catch. `session.js` redirects on 401; server/network failure can leave incomplete profile presentation. | Source-confirmed error-handling gap; not a production outage claim | Follow-up dashboard integration/error-state task. |
| F05 | Timetable keeps selected day state and separately retries timetable/reflection reads. | Existing behavior to preserve | Current JS regression tests remain required. |
| F06 | Current schema and startup migration use `ON DELETE SET NULL` with captured `activity_name`, while older README notes still describe cascading deletion. | Documentation drift; deployed migration status not inspected | Reconcile copy during reflection-retention review; no schema change here. |
| F07 | Planner, focus, buddy, notification and group masterclass handlers/routes already exist. | Source-confirmed capability presence, not full live acceptance | Redesign and verify their actual journeys rather than reimplementing from old backlog text. |

The baseline browser job also records overflow observations rather than changing old screens to make them pass. Its observations must be distinguished from source-confirmed findings. No claims are made about actual user task completion, production data, or physical-device performance.

## Preview behavior

- Today and Timetable switch locally; mobile bottom navigation and More menu demonstrate the proposed structure.
- Day buttons select the phone agenda and highlight the corresponding desktop day.
- State selector demonstrates populated/empty/loading/error/expired states. Retry restores populated sample data.
- Appearance toggles light/dark, stored under `l2e-redesign-theme`, separate from the existing app's preference.
- Buttons for reflection, focus, peers and projects explain that they are preview actions. They do not open pretend working dialogs.
- Three of five sample due blocks have reflection records (60% coverage). This is labelled reflection coverage, not verified study time.
- Preview dates, names and study entries are fictional and explicitly labelled. No real person's email or reflection is included.

## Verification commands and boundaries

```bash
npm ci --ignore-scripts
python3 scripts/audit-ui.py --check
npx playwright install --with-deps chromium
npm run test:ui
node --test static/js/*.test.cjs
```

Node dependencies are verification-only. The Go app still serves plain HTML/CSS/JavaScript with no frontend build or framework migration. Browser CI pins Ubuntu 24.04 for its pinned Playwright version.

Browser checks cover 320/360/375/390/412/480/768/820/1024/1280/1440/1920/3840px, header clearance, 44px control geometry, no unintended preview page overflow, both themes, state retry, selected day, menu Escape/focus restoration, theme persistence and no preview API calls. Axe scans target WCAG 2.1 A/AA across both views, both themes and all five preview states. Passing scans do not certify every accessibility criterion, assistive device or live user journey.

Baseline captures use the unchanged static HTML/JavaScript (there are currently no Go template actions in these files) with intercepted fictional API responses on a read-only localhost fixture host. Tailwind/font CDN assets remain the original external dependencies; failures are listed in `evidence.json`. These screenshots do not demonstrate a production API, real session expiry, email or data persistence. Baseline existing failures are not asserted away. No test writes to production or starts the production app.

Existing Go tests/build, JavaScript tests and SQLite schema validation stay in CI; `go vet ./...` is added. Screenshots and measurements are uploaded to the Actions artifact rather than committing baseline binaries into the source tree.

## Next tasks and acceptance

The next unchecked task is shared components, followed by navigation integration and actual Dashboard/Timetable redesign. Approve the visual direction through review of this preview as implementation progresses. Physical Android/laptop acceptance, Safari testing, the remaining screens, real authenticated end-to-end flows and production integration are not completed by this foundation batch.
