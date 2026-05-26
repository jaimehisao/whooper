# Agents

## Cursor Cloud specific instructions

### Overview

Whooper is a Go CLI/TUI that syncs Whoop health data via OAuth2 and stores it locally in SQLite (pure-Go, no CGO). No external databases or services are needed for development or testing.

### Key commands

All standard dev commands are in the `Makefile`:

- `make build` — build the `whooper` binary
- `make test` — run all tests (`go test ./... -v`)
- `make lint` — lint with `go vet ./...`
- `make benchmark` — SQLite store benchmarks

### Running the app without Whoop credentials

Tests and most CLI commands work without real Whoop API credentials. Use a temporary home directory to isolate state:

```bash
export WHOOPER_HOME=$(mktemp -d)
./whooper doctor --skip-api
./whooper status --json
./whooper export -e cycles -f json
```

The HTTP server (`whooper serve --addr 127.0.0.1:9464`) starts and responds on `/healthz`, `/status`, `/metrics`, and `/api/*` without credentials.

### Gotchas

- The TUI (`./whooper` with no args) requires a real terminal (TTY). In headless CI or non-interactive shells, test via `doctor`, `status`, `export`, or `serve` commands instead.
- SQLite is embedded (pure-Go via `modernc.org/sqlite`). There is no external database process to start.
- Docker/Compose is only needed for the optional Grafana visualization stack, not for core dev/test.
