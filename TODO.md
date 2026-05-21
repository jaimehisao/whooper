# TODO Backlog

This backlog tracks production hardening, correctness fixes, feature work, and
test improvements for Whooper.

## 1) Correctness fixes

- [x] Fix alert false positives when today's recovery or strain data is missing; do not treat missing data as zero.
- [x] Validate `whooper sync --since` as `YYYY-MM-DD` and fail early for invalid dates.
- [x] Return structured JSON errors from `/api/*` endpoints instead of plain text responses.
- [x] Review existing TODO entries against the current code and remove stale items as features land.

## 2) API and export improvements

- [x] Add `from=YYYY-MM-DD` and `to=YYYY-MM-DD` filters to `/api/recovery`, `/api/sleep`, `/api/strain`, and `/api/workouts`.
- [x] Keep `limit` handling for API endpoints, but document defaults, maximums, and invalid-value behavior.
- [x] Add `whooper export --from YYYY-MM-DD --to YYYY-MM-DD` for date-bounded exports.
- [x] Add tests for API date filters, limit bounds, and JSON error responses.
- [x] Add tests for export date filtering across cycles, recoveries, sleeps, and workouts.

## 3) CLI usability features

- [x] Add `whooper summary` or `whooper inspect` with latest recovery, HRV, sleep debt, strain, workouts, and last sync state.
- [x] Add alert configuration commands, such as enabling/disabling alerts and setting low-recovery/high-strain thresholds.
- [x] Improve command errors with next-step hints for missing config, missing token, empty database, and failed database open.
- [x] Add unit tests for success/failure paths in `sync`, `login`, `tui`, summary/inspect, and alert configuration commands.

## 4) TUI behavior and polish

- [x] Add model tests for `internal/tui/app.go` (`Init`, `Update`, `View`).
- [x] Add tab switch and keybinding navigation tests.
- [x] Add sync state tests (`syncing`, success message, error message, clear message tick).
- [x] Add window resize propagation tests to child views.
- [x] Add range-change and workout detail toggle tests for `internal/tui/views/*`.
- [x] Add explicit empty states when no local data exists.
- [x] Show last-sync timestamp and stale-data state in the dashboard/status area.
- [x] Add a compact today/detail panel with current recovery, sleep, strain, and workout context.

## 5) End-to-end smoke tests in CI

- [x] Add CI smoke test job that builds the binary and runs core CLI commands in a temp home dir.
- [x] Add smoke checks for `whooper config`, `whooper export`, and non-interactive command paths.
- [x] Add smoke checks for `whooper doctor --skip-api`, `whooper status --json`, and API/server construction where feasible.
- [x] Ensure smoke tests run on pull requests and block merges on failure.

## 6) Data-quality and property tests

- [x] Add fuzz/property tests for pagination token handling in `internal/api/pagination.go`.
- [x] Add property tests for trend/correlation query invariants in `internal/store/queries.go`.
- [x] Add formatter edge-case tests (empty data, NaN handling, large ranges, invalid timestamps).
- [x] Add tests for SQL views (`daily_recovery`, `daily_sleep`, `daily_strain`, `workout_summary`) to lock their public shape.

## 7) Migration resilience tests

- [x] Add upgrade tests from older schema versions to current in `internal/store/migrations.go`.
- [x] Add idempotency test for reopening DB and rerunning migrations.
- [x] Assert expected index existence after migration.

## 8) Security testing and hardening checks

- [x] Add tests for repeated OAuth callbacks and mismatched-state flows.
- [x] Add malformed callback query tests (missing code/state, unexpected params).
- [x] Add CI checks for dependency/security drift on PRs (fail on high-severity findings where appropriate).
- [x] Review token/config file permissions in tests for supported platforms.

## 9) Observability and supportability

- [x] Add `--debug` or verbose mode with structured logs around sync retries/failures.
- [x] Add tests validating expected debug log events for sync and API failures.
- [x] Document troubleshooting workflow for collecting logs and reproducing failures.
- [x] Add metrics for stale-sync age, API failure counts, last successful sync per entity, and alert state.
- [x] Add documented Prometheus alert examples for stale data, missing token, failed sync, and low recovery/high strain.

## 10) Grafana bridge and service mode

- [x] Add `whooper sync --loop --interval <duration>` or a dedicated daemon command for background syncing.
- [x] Add HTTP API endpoints for Grafana-friendly health data queries backed by the local SQLite cache.
- [x] Expand `/metrics` beyond bridge health to include latest recovery, HRV, RHR, sleep, strain, and workout summary gauges.
- [x] Document Prometheus scrape configuration and Grafana dashboard setup for service-mode deployments.
- [x] Add SQL views or stable query examples for Grafana panels (`daily_recovery`, `daily_sleep`, `daily_strain`, `workout_summary`).
- [x] Add a combined service mode that syncs on an interval and serves HTTP from one command/process.

## 11) Release and distribution

- [x] Add GoReleaser configuration for tagged releases, checksums, and packaged binaries.
- [x] Add an install script or documented `go install` flow with versioned examples.
- [x] Add release smoke checks that verify the packaged binary starts and reports the expected version.
