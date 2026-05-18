# TODO Backlog

This backlog tracks the next production-hardening and test-improvement work.

## 1) TUI behavior tests

- [ ] Add model tests for `internal/tui/app.go` (`Init`, `Update`, `View`).
- [ ] Add tab switch and keybinding navigation tests.
- [ ] Add sync state tests (`syncing`, success message, error message, clear message tick).
- [ ] Add window resize propagation tests to child views.
- [ ] Add range-change and workout detail toggle tests for `internal/tui/views/*`.

## 2) Command-layer tests and dependency injection

- [x] Add test seams/factories in `cmd/sync.go` for OAuth token source and syncer creation.
- [x] Add test seams in `cmd/login.go` for OAuth flow runner and token persistence.
- [x] Add test seams in `cmd/tui.go` for sync function construction and app runner.
- [ ] Add unit tests for success/failure paths in `sync`, `login`, and `tui` commands.

## 3) End-to-end smoke tests in CI

- [x] Add CI smoke test job that builds the binary and runs core CLI commands in a temp home dir.
- [x] Add smoke checks for `whooper config`, `whooper export`, and non-interactive command paths.
- [ ] Ensure smoke tests are configured as required checks for protected branches.

## 4) Data-quality and property tests

- [ ] Add fuzz/property tests for pagination token handling in `internal/api/pagination.go`.
- [ ] Add property tests for trend/correlation query invariants in `internal/store/queries.go`.
- [ ] Add formatter edge-case tests (empty data, NaN handling, large ranges, invalid timestamps).

## 5) Migration resilience tests

- [x] Add upgrade tests from older schema versions to current in `internal/store/migrations.go`.
- [x] Add idempotency test for reopening DB and rerunning migrations.
- [x] Assert expected index existence after migration.

## 6) Security testing and hardening checks

- [ ] Add tests for repeated OAuth callbacks and mismatched-state flows.
- [ ] Add malformed callback query tests (missing code/state, unexpected params).
- [ ] Add CI checks for dependency/security drift on PRs (fail on high-severity findings where appropriate).

## 7) Observability and supportability

- [x] Add `--debug` or verbose mode with structured logs around sync retries/failures.
- [x] Add tests validating expected debug log events for sync and API failures.
- [x] Document troubleshooting workflow for collecting logs and reproducing failures.
