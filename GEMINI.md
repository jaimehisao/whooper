# Whooper: Gemini Context & Instructions

Whooper is a Go-based CLI tool and Terminal User Interface (TUI) for syncing and visualizing health data from the Whoop API. It uses a local SQLite database for offline access and provides rich analytics through a dashboard.

## Project Overview

- **Purpose**: Sync Whoop health data (cycles, recovery, sleep, workouts) and provide a terminal-based dashboard for analysis.
- **Main Technologies**:
  - **Language**: Go 1.24+
  - **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea), [Lipgloss](https://github.com/charmbracelet/lipgloss), [Bubbles](https://github.com/charmbracelet/bubbles)
  - **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
  - **Storage**: SQLite (Pure-Go via `modernc.org/sqlite`)
  - **API Client**: [Resty](https://github.com/go-resty/resty/v2)
  - **Authentication**: OAuth2 (Standard flow with local callback server)

## Architecture

- `main.go`: Application entry point.
- `cmd/`: CLI command definitions (Cobra).
  - `root.go`: Default entry point (launches TUI).
  - `login.go`: OAuth2 browser-based authentication flow.
  - `sync.go`: Data synchronization logic.
  - `config.go`: Configuration management (API credentials).
  - `export.go`: JSON/CSV data export.
- `internal/`: Core application logic.
  - `api/`: Whoop REST API client with rate-limiting and pagination support.
  - `auth/`: OAuth2 token management and callback server.
  - `store/`: SQLite persistence layer, migrations, and analytical queries.
  - `sync/`: Incremental synchronization orchestrator.
  - `models/`: Data structures mapping to Whoop API responses.
  - `analysis/`: Analytical logic (moving averages, Pearson correlations).
  - `tui/`: Bubbletea application, components, and views.

## Building and Running

- **Build**: `make build` or `go build -o whooper .`
- **Run TUI**: `./whooper`
- **Sync Data**: `./whooper sync`
- **Authentication**: `./whooper login`
- **Test**: `make test` or `go test ./... -v`
- **Lint**: `make lint` (runs `go vet`)
- **Benchmark**: `make benchmark` (specifically for `internal/store`)
- **Install**: `make install`

## Development Conventions

- **Pure-Go SQLite**: Use `modernc.org/sqlite` to avoid CGO dependencies for easier cross-compilation.
- **Incremental Sync**: The sync logic implements a 1-day overlap to ensure retroactive score updates from Whoop are captured.
- **Concurrency**: `internal/sync` uses goroutines to fetch different data entities in parallel.
- **Generic Pagination**: `internal/api/pagination.go` provides a generic `FetchAll[T]` function to handle paginated API responses.
- **Error Handling**: Use `fmt.Errorf("context: %w", err)` for error wrapping.
- **Styling**: TUI styling is centralized in `internal/tui/styles.go` using Lipgloss.
- **Testing**:
  - Unit tests are located alongside source files (`*_test.go`).
  - Integration tests are found in `cmd/integration_test.go`, `internal/sync/syncer_integration_test.go`, and `internal/store/integration_test.go`.

## Configuration & Data

- All configuration and data are stored in `~/.whooper/`:
  - `config.yaml`: API credentials.
  - `token.json`: OAuth2 tokens.
  - `whooper.db`: SQLite database.
