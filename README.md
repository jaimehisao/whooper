# Whooper

A CLI tool that syncs your [Whoop](https://www.whoop.com/) health data via the Whoop API and presents it in a rich terminal dashboard. Data is stored locally in SQLite for offline access and future extensibility (e.g. Grafana SQL data source).

## Features

- **OAuth2 Authentication** — Browser-based login flow with local callback server
- **Incremental Sync** — Fetches only new data with 1-day overlap for retroactive score updates
- **TUI Dashboard** — 5-tab terminal UI with recovery gauge, sleep stage bars, workout table, trend sparklines, and correlation scatter plots
- **Local SQLite Storage** — Pure-Go SQLite (no CGO), WAL mode, flat schema optimized for analytics
- **Export** — JSON and CSV export for all data types
- **Remote CLI client** — Point `summary`, `status`, and `export` at a Whooper `service`/`serve` backend over HTTP

## Prerequisites

- **Go 1.24.x** or later (for building from source)

## Installation

See [docs/install.md](docs/install.md) for detailed installation instructions, including:
- Installing via `go install`
- Building from source with `make install`
- Downloading pre-built binaries from the repository releases page

## Backlog

- Track upcoming production hardening and testing work in `TODO.md`.

## Quick Start

```bash
# Build
go build -o whooper .

# Configure API credentials (from developer.whoop.com)
whooper config set client-id <your-client-id>
whooper config set client-secret <your-client-secret>

# Authenticate
whooper login

# Sync data
whooper sync

# Launch dashboard
whooper
```

## Commands

| Command | Description |
|---------|-------------|
| `whooper` | Launch TUI dashboard (default) |
| `whooper login` | OAuth2 browser login flow |
| `whooper sync` | Fetch latest data from Whoop API |
| `whooper sync --debug` | Fetch data with local diagnostic output |
| `whooper sync --loop --interval 30m` | Keep syncing in the foreground on an interval |
| `whooper service --addr 0.0.0.0:9464 --allow-remote --token $WHOOPER_SERVE_TOKEN --interval 30m` | Run combined sync loop and HTTP API server |
| `whooper config` | Show current configuration (secrets masked) |
| `whooper config set <key> <value>` | Set config (client-id, client-secret, redirect-url, remote-url, remote-token) |
| `whooper alerts` | Show alert configuration (alias: status, show) |
| `whooper alerts enable` | Enable alerts |
| `whooper alerts disable` | Disable alerts |
| `whooper alerts set <key> <value>` | Set threshold (low-recovery, high-strain) |
| `whooper summary` | Show latest health metrics and sync state (alias: inspect) |
| `whooper doctor --json` | Run machine-readable readiness checks |
| `whooper status --json` | Show local config, token, database, and sync state |
| `whooper export -e <entity> -f <format> [--from YYYY-MM-DD] [--to YYYY-MM-DD]` | Export data (entities: cycles, recoveries, sleeps, workouts; formats: json, csv) |
| `whooper agent …` | **Read-only JSON CLI for agents** (see [Agent CLI](#agent-cli)) |

## Agent CLI

Read-only JSON interface for automation and LLM agents. Always emits **one JSON
object** on stdout (success or failure). Never runs login, sync, config mutation,
alerts mutation, serve, or the TUI.

```bash
whooper agent schema              # command catalog + envelope docs
whooper agent summary             # latest health + last_sync
whooper agent status              # config / db / sync status
whooper agent recovery --from 2024-01-01 --to 2024-01-31 --limit 30
whooper agent sleep --limit 14
whooper agent strain
whooper agent workouts --from 2024-06-01
whooper agent doctor              # readiness (skips Whoop API by default)
whooper agent doctor --api        # also check Whoop API (local token required)
```

Envelope shape:

```json
{
  "ok": true,
  "command": "summary",
  "source": "local",
  "generated_at": "2026-08-03T12:00:00Z",
  "data": { }
}
```

On failure, `ok` is false and `error` is `{ "class": "...", "message": "..." }`.
Classes include `missing_db`, `missing_token`, `unauthorized`, `unreachable`,
`invalid_args`, `http_error`, `internal`. Exit `0` on success, `1` on app error,
`2` on invalid args.

Honors **remote** mode (`remote-url` / `WHOOPER_REMOTE_URL`) for data commands,
same as `summary` / `export`.

## Troubleshooting

Start with local readiness checks:

```bash
whooper doctor --skip-api
whooper status
```

For issue reports or reproducible support bundles, collect machine-readable output:

```bash
whooper doctor --json --skip-api
whooper status --json
whooper sync --debug
```

Use `doctor --skip-api` when credentials or network access are unavailable. Use `sync --debug` when authentication succeeds but sync behavior is unclear; it prints config presence, token loading, database path, selected sync range, sync failures, and alert evaluation counts without printing secrets.

## TUI Keybindings

| Key | Action |
|-----|--------|
| `1-5` | Switch tabs (Dashboard, Recovery, Sleep, Workouts, Correlations) |
| `Tab` | Next tab |
| `s` | Trigger sync |
| `< >` | Change time range (7/14/30/90 days) |
| `j/k` or `↑/↓` | Navigate lists |
| `Enter` | View detail / toggle |
| `[ ]` | Change Y metric (correlations view) |
| `q` | Quit |

## Grafana and Prometheus

Whooper can act as a local bridge between the Whoop API and Grafana. The primary
path today is SQLite:

```text
Whoop API -> whooper sync -> ~/.whooper/whooper.db -> Grafana SQLite datasource
```

Run a sync, then start the bundled Grafana stack:

```bash
whooper sync
docker compose up -d
```

Grafana is exposed at `http://localhost:3000`. The compose file installs the
`frser-sqlite-datasource` plugin, mounts `~/.whooper` read-only at
`/data` (so WAL sidecar files are visible), provisions a `Whooper` datasource, and loads dashboards from
`grafana/provisioning/dashboards/json`.

The compose stack also includes a `whooper` bridge service. It mounts the same
data directory, keeps syncing in a loop, and exposes `/metrics` on port `9464`:

```bash
# Optional: choose a data directory other than ~/.whooper
export WHOOPER_HOME=/var/lib/whooper

# Optional: change the sync cadence
export WHOOPER_SYNC_INTERVAL=15m

docker compose up -d --build whooper grafana
```

Run `whooper login` on the host first so the mounted data directory contains a
valid config and token.

SQLite is the best current source for historical WHOOP data because sync
backfills records and preserves exact sleeps, recoveries, cycles, and workouts.
Prometheus scraping is better suited to current status, health, and alerting
metrics. The `whooper serve` command currently exposes operational endpoints:

```bash
whooper serve
curl http://127.0.0.1:9464/healthz
curl http://127.0.0.1:9464/status
curl http://127.0.0.1:9464/metrics
curl http://127.0.0.1:9464/api/summary
curl http://127.0.0.1:9464/api/recovery
curl http://127.0.0.1:9464/api/sleep
curl http://127.0.0.1:9464/api/strain
curl http://127.0.0.1:9464/api/workouts
```

The `/metrics` endpoint reports bridge health, record counts, sync timestamps,
and latest cached WHOOP health gauges. Health gauges are exposed as
`whooper_latest_health_metric{metric="..."}` for values such as
`recovery_score`, `hrv_rmssd`, `resting_heart_rate`, `sleep_actual_hours`,
`sleep_need_hours`, `sleep_need_gap_hours`, `sleep_efficiency_pct`,
`sleep_performance_pct`, `sleep_consistency_pct`, `day_strain`,
`workout_strain`, and workout heart-rate/distance metrics.

For a simple background bridge on a host, run:

```bash
whooper service --addr 0.0.0.0:9464 --allow-remote --token "$WHOOPER_SERVE_TOKEN" --interval 30m
```

This runs both the periodic sync loop and the HTTP API server in a single
process. Then point Prometheus at `http://<host>:9464/metrics`. For alert
examples, see [docs/prometheus-alerts.md](docs/prometheus-alerts.md).

The `/api/*` endpoints return JSON from the local SQLite cache. `recovery`,
`sleep`, `strain`, and `workouts` accept optional query parameters:

- `limit`: Number of records to return (default: 90, max: 1000). Invalid or missing values fallback to the default.
- `from`: Start date in `YYYY-MM-DD` format.
- `to`: End date in `YYYY-MM-DD` format.

Example: `/api/sleep?limit=30&from=2024-01-01&to=2024-01-31`.

For systemd deployment examples, see `docs/systemd.md`.

### Remote CLI client

Run sync + API on a server, and use the CLI on a laptop without a local SQLite
cache. Configure a backend URL (and bearer token when the server requires one):

```bash
# On the server (after login + sync credentials exist under WHOOPER_HOME):
export WHOOPER_SERVE_TOKEN='long-random-token'
whooper service --addr 0.0.0.0:9464 --allow-remote --token "$WHOOPER_SERVE_TOKEN" --interval 30m

# On the client:
whooper config set remote-url http://whooper-host:9464
whooper config set remote-token "$WHOOPER_SERVE_TOKEN"
# Or env (env wins over config file):
#   export WHOOPER_REMOTE_URL=http://whooper-host:9464
#   export WHOOPER_REMOTE_TOKEN=...   # or WHOOPER_SERVE_TOKEN

whooper summary --json
whooper status
whooper export -e recoveries -f json
```

When `remote-url` / `WHOOPER_REMOTE_URL` is set, `summary`/`inspect`, `status`,
and `export` call the remote HTTP API (`/api/summary`, `/status`, `/api/*`) and
do not require a local `whooper.db`. Unset remote URL to restore local SQLite
mode. `login`, `sync`, and the TUI remain local (Whoop OAuth + local DB).

Remote failures are explicit: `missing_token`, `unauthorized`, and `unreachable`.

### Remote Grafana

If Grafana runs on a different machine, prefer one of these setups:

| Approach | Tradeoff |
|----------|----------|
| Run `whooper` on the Grafana host | Simplest; the SQLite database stays local to Grafana. |
| Copy `whooper.db` to the Grafana host | Works for batch updates; copy to a temp file and atomically rename to avoid partial reads. |
| Expose a Whooper HTTP API | Best fit for a service-style bridge; available through `whooper serve`. |
| Use a network database | Better for multi-host deployments than sharing SQLite over NFS. |

The shipped service-style bridge is:

```text
Whoop API -> whooper daemon/API -> Prometheus and/or Grafana HTTP datasource
```

That would add richer HTTP endpoints for health records and richer Prometheus
metrics while keeping SQLite as the durable local cache.

## Architecture

```
whooper
├── main.go                          Entry point
├── cmd/                             CLI commands (Cobra)
│   ├── root.go                      Default → TUI
│   ├── login.go                     OAuth2 flow
│   ├── sync.go                      API → SQLite
│   ├── config.go                    Show/set credentials
│   ├── export.go                    JSON/CSV output
│   └── tui.go                       Wire views → app
│
└── internal/
    ├── config/                      ~/.whooper/ management
    │   └── config.go                YAML config, paths
    │
    ├── auth/                        Authentication
    │   ├── oauth.go                 OAuth2 config & scopes
    │   ├── server.go                Local :8484 callback server
    │   └── token.go                 Token save/load/auto-refresh
    │
    ├── models/                      Whoop API structs
    │   ├── profile.go               User profile
    │   ├── cycle.go                 Physiological cycles + strain
    │   ├── recovery.go              Recovery score, HRV, RHR, SpO2
    │   ├── sleep.go                 Sleep stages, need, efficiency
    │   ├── workout.go               Workouts, HR zones, sport map
    │   └── pagination.go            Generic paginated response
    │
    ├── api/                         HTTP client
    │   ├── client.go                Bearer auth, rate-limit retry
    │   ├── pagination.go            FetchAll[T] generic fetcher
    │   └── {profile,cycle,...}.go   Endpoint wrappers
    │
    ├── store/                       SQLite persistence
    │   ├── db.go                    Open, WAL mode
    │   ├── migrations.go            Schema DDL (flat tables)
    │   ├── queries.go               Trends, correlations, sync state
    │   └── {profile,cycle,...}.go   CRUD operations
    │
    ├── sync/
    │   └── syncer.go                Incremental sync orchestrator
    │
    ├── analysis/
    │   ├── trends.go                Moving averages, percent change
    │   ├── correlations.go          Pearson correlation
    │   └── summary.go               Weekly report generation
    │
    └── tui/                         Terminal UI (Bubbletea)
        ├── app.go                   Top-level model, tab navigation
        ├── keys.go                  Keybindings
        ├── styles.go                Whoop color palette
        ├── components/
        │   ├── sparkline.go         Unicode block sparklines
        │   ├── barchart.go          Horizontal bar charts
        │   ├── gauge.go             Recovery gauge
        │   └── table.go             Highlighted table
        └── views/
            ├── dashboard.go         Today's summary + sparklines
            ├── recovery.go          HRV/RHR/recovery trends
            ├── sleep.go             Stage bars, duration, efficiency
            ├── workouts.go          Sortable table + detail view
            └── correlations.go      Scatter plot + Pearson r
```

## Data Flow

```mermaid
graph TD
    subgraph External
        WHOOP[Whoop API<br/>REST + OAuth2]
    end

    subgraph Auth
        OAUTH[auth/oauth.go<br/>OAuth2 Config]
        SERVER[auth/server.go<br/>Callback :8484]
        TOKEN[auth/token.go<br/>Token Persistence]
        OAUTH --> SERVER
        SERVER --> TOKEN
    end

    subgraph API Layer
        CLIENT[api/client.go<br/>Bearer Auth + Retry]
        FETCH["api/pagination.go<br/>FetchAll[T] Generic"]
        CLIENT --> FETCH
    end

    subgraph Data Models
        MODELS["models/<br/>Profile, Cycle, Recovery,<br/>Sleep, Workout"]
    end

    subgraph Sync Engine
        SYNCER[sync/syncer.go<br/>Incremental + 1d Overlap]
    end

    subgraph Storage
        DB[(SQLite<br/>WAL Mode)]
        MIGRATIONS[store/migrations.go<br/>Flat Schema DDL]
        CRUD[store/*.go<br/>Batch UPSERT + Queries]
        MIGRATIONS --> DB
        CRUD --> DB
    end

    subgraph Analysis
        TRENDS[analysis/trends.go<br/>Moving Averages]
        CORR[analysis/correlations.go<br/>Pearson r]
        SUMMARY[analysis/summary.go<br/>Weekly Reports]
    end

    subgraph "TUI (Bubbletea)"
        APP[tui/app.go<br/>Tab Navigation]
        STYLES[tui/styles.go<br/>Whoop Colors]
        COMPONENTS["tui/components/<br/>Sparkline, Gauge,<br/>BarChart, Table"]
        VIEWS["tui/views/<br/>Dashboard, Recovery,<br/>Sleep, Workouts,<br/>Correlations"]
        APP --> VIEWS
        STYLES --> VIEWS
        COMPONENTS --> VIEWS
    end

    subgraph CLI
        CMD["cmd/<br/>login, sync, config,<br/>export, tui"]
    end

    WHOOP <-->|"JSON + Bearer Token"| CLIENT
    TOKEN -->|Auto-refresh| CLIENT
    FETCH -->|"[]T records"| SYNCER
    WHOOP -.->|Response structs| MODELS
    SYNCER -->|Batch UPSERT| CRUD
    DB -->|Trend queries| TRENDS
    DB -->|Metric pairs| CORR
    DB -->|Date ranges| SUMMARY
    TRENDS --> VIEWS
    CORR --> VIEWS
    DB -->|CRUD| VIEWS
    CMD -->|Orchestrates| SYNCER
    CMD -->|Launches| APP
    CMD -->|"JSON/CSV"| DB

    style WHOOP fill:#1a1a2e,stroke:#16a085,color:#e0e0e0
    style DB fill:#1a1a2e,stroke:#00d46a,color:#e0e0e0
    style APP fill:#1a1a2e,stroke:#16a085,color:#e0e0e0
    style VIEWS fill:#1a1a2e,stroke:#00d46a,color:#e0e0e0
```

### Sync Sequence

```mermaid
sequenceDiagram
    participant User
    participant CLI as whooper sync
    participant Sync as Sync Engine
    participant API as API Client
    participant Whoop as Whoop API
    participant DB as SQLite

    User->>CLI: whooper sync
    CLI->>DB: GetSyncState("cycles")
    DB-->>CLI: last_synced timestamp
    CLI->>Sync: SyncAll()

    Sync->>API: GetProfile()
    API->>Whoop: GET /v1/user/profile/basic
    Whoop-->>API: Profile JSON
    Sync->>DB: SaveProfile()

    loop For each: Cycles, Recoveries, Sleeps, Workouts
        Sync->>API: GetCycles(start - 1 day)
        API->>Whoop: GET /v1/cycle?start=...
        loop Paginate
            Whoop-->>API: {records, next_token}
            API->>Whoop: GET /v1/cycle?nextToken=...
        end
        API-->>Sync: []Cycle
        Sync->>DB: SaveCycles() (batch UPSERT)
        Sync-->>CLI: progress("cycles", count)
    end

    Sync->>DB: SetSyncState("cycles", now)
    CLI-->>User: Sync complete!
```

### TUI Navigation

```mermaid
stateDiagram-v2
    [*] --> Dashboard: Launch

    Dashboard --> Recovery: 2
    Dashboard --> Sleep: 3
    Dashboard --> Workouts: 4
    Dashboard --> Correlations: 5

    Recovery --> Dashboard: 1
    Recovery --> Sleep: 3
    Recovery --> Workouts: 4
    Recovery --> Correlations: 5

    Sleep --> Dashboard: 1
    Sleep --> Recovery: 2
    Sleep --> Workouts: 4
    Sleep --> Correlations: 5

    Workouts --> Dashboard: 1
    Workouts --> Recovery: 2
    Workouts --> Sleep: 3
    Workouts --> Correlations: 5
    Workouts --> WorkoutDetail: Enter
    WorkoutDetail --> Workouts: Esc

    Correlations --> Dashboard: 1
    Correlations --> Recovery: 2
    Correlations --> Sleep: 3
    Correlations --> Workouts: 4

    state Dashboard {
        [*] --> TodayRecovery
        TodayRecovery --> Metrics
        Metrics --> Sparkline7d
        Sparkline7d --> RecentWorkouts
    }

    state Recovery {
        [*] --> RecoveryTrend
        RecoveryTrend --> HRVTrend
        HRVTrend --> RHRTrend
        note right of RecoveryTrend: < > changes range\n7/14/30/90 days
    }

    state Correlations {
        [*] --> ScatterPlot
        note right of ScatterPlot: < > X metric\n[ ] Y metric\nShows Pearson r
    }
```

### Database Schema

```mermaid
erDiagram
    profile {
        int user_id PK
        text email
        text first_name
        text last_name
    }

    cycle {
        int id PK
        int user_id
        text start
        text end
        text score_state
        real strain
        real kilojoule
        int average_heart_rate
        int max_heart_rate
    }

    recovery {
        int cycle_id PK
        int sleep_id
        int user_id
        text score_state
        real recovery_score
        real hrv_rmssd
        real resting_heart_rate
        real spo2_percentage
        real skin_temp_celsius
    }

    sleep {
        int id PK
        int user_id
        text start
        text end
        int nap
        text score_state
        int total_in_bed_time_milli
        int total_light_sleep_time_milli
        int total_slow_wave_sleep_time_milli
        int total_rem_sleep_time_milli
        real sleep_efficiency_pct
        real sleep_performance_pct
    }

    workout {
        int id PK
        int user_id
        text start
        text end
        int sport_id
        text score_state
        real strain
        int average_heart_rate
        int max_heart_rate
        real kilojoule
        real distance_meter
    }

    sync_state {
        text entity PK
        text last_synced
    }

    cycle ||--o| recovery : "cycle_id"
    sleep ||--o| recovery : "sleep_id"
```

## Design Decisions

- **Flat SQLite schema** — Score sub-objects stored as columns, not separate tables. Optimized for analytics queries and direct use as a Grafana SQL data source.
- **Pure-Go SQLite** — `modernc.org/sqlite` requires no CGO, enabling easy cross-compilation.
- **Generics for pagination** — `FetchAll[T any]` avoids duplicating fetch logic across endpoints.
- **Auto-persisting token source** — Wrapper saves refreshed OAuth2 tokens to disk automatically.
- **Incremental sync with 1-day overlap** — Catches retroactively updated Whoop scores.
- **TUI as default command** — Running `whooper` with no args launches the dashboard.

## Configuration

All data is stored in `~/.whooper/`:

| File | Purpose |
|------|---------|
| `config.yaml` | Client ID, secret, redirect URL |
| `token.json` | OAuth2 access/refresh tokens |
| `whooper.db` | SQLite database |

Set `WHOOPER_HOME` to use a different data directory. This is useful for
containers, systemd services, and hosts where Grafana should read from a
specific volume:

```bash
WHOOPER_HOME=/var/lib/whooper whooper sync
```

## Operations

### Supported Runtime

- Go `1.24.x` is the primary development and CI target.
- Linux, macOS, and Windows artifacts are produced through GoReleaser.

### Failure Modes and Recovery

- **OAuth/login failure**: rerun `whooper login`; verify `client-id`, `client-secret`, and redirect URL in `whooper config`.
- **API/rate-limit failure**: rerun `whooper sync`; the syncer uses retries and incremental sync with overlap.
- **Partial sync failure**: no sync-state checkpoint is persisted when sync fails, so rerunning is safe.
- **Local DB issues**: move or back up `~/.whooper/whooper.db` and run sync again to rebuild data.

### Backup and Restore

- Backup: copy `~/.whooper/config.yaml`, `~/.whooper/token.json`, and either checkpoint SQLite (`PRAGMA wal_checkpoint(TRUNCATE)`) then copy `whooper.db`, or copy `whooper.db` together with `whooper.db-wal` / `whooper.db-shm` while the writer is stopped.
- Restore: place files back in `~/.whooper/` with permissions `0700` on directory and `0600` on files.

### Security Notes

- Credentials and tokens are stored on local disk only.
- Secret/token files should never be committed; `.gitignore` and CI secret scanning are configured to help prevent leaks.
- Report suspected vulnerabilities using `SECURITY.md`.

## Development

### CI

End-to-end smoke tests run on every push and pull request to `main`. The workflow verifies:
- Building the binary and running unit tests
- CLI command execution (`version`, `config`, `doctor`, `status`, `export`)
- API server construction and endpoint availability (`serve`)

Tests use a temporary `WHOOPER_HOME` and do not require external API access or browser interaction.

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/charmbracelet/bubbletea` | Terminal UI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components |
| `golang.org/x/oauth2` | OAuth2 flow |
| `modernc.org/sqlite` | Pure-Go SQLite driver |
| `github.com/go-resty/resty/v2` | HTTP client |
| `gopkg.in/yaml.v3` | Config parsing |
