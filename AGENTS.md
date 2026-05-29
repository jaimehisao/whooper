# Agents

## Cursor Cloud specific instructions

### Overview

Whooper is a Go 1.24 CLI tool that syncs Whoop health data to local SQLite and provides a TUI dashboard, HTTP API server, and Prometheus metrics. No external services are required for development or testing.

### Build, Lint, Test

Standard commands are in the `Makefile`:

- **Build:** `make build` (or `go build -o whooper .`)
- **Lint:** `make lint` (runs `go vet ./...`)
- **Test:** `make test` (runs `go test ./... -v`)
- **Benchmark:** `make benchmark`

### Running the HTTP server

```bash
WHOOPER_HOME=/tmp/whooper-test whooper serve --addr 127.0.0.1:9464
```

Endpoints: `/healthz`, `/status`, `/metrics`, `/api/summary`, `/api/recovery`, `/api/sleep`, `/api/strain`, `/api/workouts`.

### Gotchas

- **WHOOPER_HOME env var:** Do NOT export `WHOOPER_HOME` in the shell where you run tests; the `internal/config` tests expect the default `$HOME/.whooper` path. Set it only per-command when running the binary.
- **SQLite in serve mode:** The `whooper serve` command may fail with "out of memory (14)" if the database file doesn't exist yet. Run `whooper status` or `whooper doctor` once with the same `WHOOPER_HOME` to initialize the DB before starting the server.
- **No CGO needed:** SQLite is pure-Go via `modernc.org/sqlite`; no C compiler required.
- **Docker Compose is optional:** The `docker-compose.yml` provides Grafana visualization but is not required for dev/test.
