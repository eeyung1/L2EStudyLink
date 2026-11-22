# Daily workspace integration: tasks 6–10

## Analysis and implementation

| Task | Problem | Delivered change |
| --- | --- | --- |
| Shared controls | Screens combine divergent styles and repeated local controls. | Semantic tokens now cover workspace navigation; `.l2e-ui` opts Dashboard/Timetable into shared buttons, fields, cards, selected day tabs, feedback/loading, and dialog styling. Existing plain HTML, Gin templates and vanilla JS remain. The add-block sheet traps focus, supports Escape and restores the opener. |
| Grouped navigation | Planner, focus and buddies were missing from many menus; small-page menu markup crashed the old controller. | One Go partial across 13 sidebar screens, grouped as My learning, Learn together and My profile. Exact path sets one `aria-current`. Mobile Escape, focus wrapping, resize behavior and inert background are shared. Admin APIs remain server-authorized; hiding a link grants no access. |
| Daily dashboard | Setup and reputation led the page; no daily plan was present. | Today's recurring timetable blocks, reflection shortcuts, next upcoming pending/confirmed session, weekly server-calculated reflection coverage, and a direct focus link. Independent errors/retries preserve other cards. No automatic coaching request or mutation is introduced. |
| Secondary content | Setup and recommendations competed with daily actions. | Existing skills, availability, ratings, profile editing, habits and project/fellow discovery remain below the daily workspace. Profile failure has a visible recovery action and no longer prevents discovery from being attempted. |
| Responsive timetable | A single filtered list did not provide a default desktop week. | Phone/tablet defaults to the current day, desktop above 1100px defaults to seven day columns. All days remain reachable with buttons; Week gives a stacked overview on narrow screens. Explicit day choice survives resizing and existing reloads. Block rendering uses delegated reflection buttons so quoted activity names are safe. |

## Data and behavior boundaries

- Dashboard uses existing authorized GETs: timetable, reflections, bookings, planner/week. It does not calculate its own competing adherence formula or call the AI coaching endpoint.
- Weekly metrics use an explicit browser timezone and Monday date. Coverage is completed reflections against due blocks, not proof of time worked.
- Bookings currently expose a date and clock time without an offset. The card preserves that representation and labels it as scheduled local time; it filters out past/cancelled/completed sessions and orders upcoming pending/confirmed sessions.
- Add reflection opens the existing timetable reflection form for a matching block today. A missing/deleted/already-reflected block falls back to the day agenda and a clear notice. Forms still save only through existing server handlers.
- The original timetable reflection-date semantics for other days, deletion-copy/schema reconciliation, recorder refinements, and all remaining roadmap tasks are separate work. No schema or retention behavior changes here.
- Shared components are integrated on Dashboard/Timetable; other feature bodies retain their current designs while consuming the shared navigation. This is not a claim that every page has been redesigned.

## Verification

- Existing Go tests, build, vet, PostgreSQL integration tests and SQLite schema check stay required.
- `templates_test.go` parses the real Go templates and executes all 13 shared-navigation consumers, rejecting duplicate menus or missing planner/focus/buddy links.
- JavaScript regression tests cover weekday defaults, filters, desktop week default and selection retention after resizing.
- Browser fixtures capture current app states and isolated sample previews. Additional integrated checks cover both themes, 320/390/768/1024/1440/1920 widths, overflow/touch geometry, desktop week visibility, mobile menu Escape and inert background, scoped WCAG scans, unsafe-looking activity text, add/delete fixture actions, reflection handoff, partial errors and retry.
- Browser API responses and writes are fictional in-memory fixtures. No production records are read or written; these checks are not authenticated Go-server end-to-end tests or physical-phone acceptance.
- Baseline screenshots remain associated with the earlier foundation run. Current artifact images use `app-*` and `integrated-*`, with source commit recorded in `evidence.json`.

Workflow: branch → PR → all CI jobs green → squash merge. README tasks 6–10 are delivered by this batch; selected-day and broader browser acceptance checklist items remain available for their explicit follow-up scope. Deployment smoke and owner device acceptance remain pending.
