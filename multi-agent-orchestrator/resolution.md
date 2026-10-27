# Resolution

Append-only completed-work evidence.


- 2026-10-27T21:47:43Z — Codex-20261027-0dda2028: Completed implementation batch: schema.sql/schema.sqlite now use nullable timetable_id with ON DELETE SET NULL and activity_name VARCHAR(255); db/postgres.go adds transactional repeatable startup backfill/constraint migration; handlers/timetable.go snapshots owned activity on atomic upsert and reads deleted-parent history with LEFT JOIN; handlers/timetable_test.go adds PostgreSQL API/migration tests plus executed SQLite regression. SQLite fresh schema, delete preservation, and foreign-key checks PASS. node --test static/js/*.test.cjs: 5 passed. Go checks in progress using GOCACHE=/tmp/codex-decouple-go-cache. PostgreSQL integration remains unrun without TEST_DATABASE_URL.

## task-decouple-reflections — implementation handoff
- Owner/session: Codex-20261027-0dda2028; claim claim-codex-decouple-20261027.
- Branch: agent/task-decouple-reflections.
- Worktree: /home/recoding/Projects/L2EStudyLink/.agent-worktrees/task-decouple-reflections.
- Commit: ec3a47867d969b8aa884354ea2dbc3be18bdabbe. Worktree clean after commit. PR: not created; branch push failed DNS resolution for github.com. Connector also reported local base 380dbff absent from remote.
- Owned files changed: schema.sql, schema.sqlite, db/postgres.go, handlers/timetable.go, handlers/timetable_test.go. Staged paths and full branch diff verified against claim.
- Result: nullable SET NULL timetable reference; server-controlled activity-name snapshot retained on updates; ownership-checked atomic reflection upsert; deleted-parent history returned with null timetable ID, snapshot activity and safe missing schedule metadata; repeatable atomic PostgreSQL migration backfills existing snapshots and replaces custom-named cascading FKs.
- PASS: GOCACHE=/tmp/codex-decouple-go-cache /home/recoding/sdk/go/bin/go build ./...
- PASS with skips: GOCACHE=/tmp/codex-decouple-go-cache /home/recoding/sdk/go/bin/go test -v ./handlers -run TestTimetable. TestTimetableReflectionSQLite passed; TestTimetableReflectionHistory and TestTimetableReflectionMigration skipped because TEST_DATABASE_URL is unset. PostgreSQL behavior is implemented but NOT runtime-verified locally.
- Full suite: GOCACHE=/tmp/codex-decouple-go-cache /home/recoding/sdk/go/bin/go test ./... — handlers and middleware passed; suite FAILED in pre-existing email TestSendEmailUsesBrevoTransactionalAPI because sandbox prohibits local listening sockets (socket: operation not permitted). Same limitation reproduced with optional nomsgpack build tag; tagged application build passed.
- PASS: node --test static/js/*.test.cjs (5 tests); git diff --check; standalone fresh SQLite schema + deletion preservation + foreign-key_check.
- Remaining gates: push branch and create focused PR, execute PostgreSQL regressions and full CI on exact commit in normal CI environment, human review/integration. User/authorized human integrator owns merge and deployment. No deployment or field verification performed.
- Deployment needs: existing PostgreSQL startup migration requires DDL privileges, takes a bounded exclusive lock, and fails startup if preservation migration fails. Back up before deployment; previously cascaded records cannot be recovered by this migration. Old handler rollback cannot read orphaned NULL timetable IDs; retain nullable-aware handlers or use a forward fix. SQLite changes cover fresh schema; runtime application remains PostgreSQL.
- Intended lifecycle: HANDOFF, backlog Done-waiting; retain claim/branch ownership pending ORCHESTRATOR resolution of CI/publication gates. No claim transfer or integration authorized by this evidence.
- Recorded UTC: 2026-10-27T21:56:06Z
