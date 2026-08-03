package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/remote"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"github.com/spf13/cobra"
)

// Agent CLI is a read-only, JSON-only interface for automation and LLM agents.
// It never runs login, sync, config mutation, alerts mutation, serve, or TUI.

const (
	agentSourceLocal  = "local"
	agentSourceRemote = "remote"
	agentSourceNone   = ""

	agentClassInvalidArgs  = "invalid_args"
	agentClassMissingDB    = "missing_db"
	agentClassMissingToken = "missing_token"
	agentClassUnauthorized = "unauthorized"
	agentClassUnreachable  = "unreachable"
	agentClassHTTP         = "http_error"
	agentClassInternal     = "internal"
)

// agentResponse is the stable JSON envelope for all agent subcommands.
type agentResponse struct {
	OK          bool             `json:"ok"`
	Command     string           `json:"command"`
	Source      string           `json:"source,omitempty"`
	GeneratedAt string           `json:"generated_at"`
	Data        any              `json:"data,omitempty"`
	Error       *agentErrorBody  `json:"error,omitempty"`
}

type agentErrorBody struct {
	Class   string `json:"class"`
	Message string `json:"message"`
}

// agentExitError carries a process exit code without relying on cobra printing usage.
type agentExitError struct {
	code int
	err  error
}

func (e *agentExitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit %d", e.code)
}

func (e *agentExitError) Unwrap() error { return e.err }

var (
	agentFrom  string
	agentTo    string
	agentLimit int
	// agentDoctorSkipAPI defaults true; --api flips it off.
	agentDoctorAPI bool
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Read-only JSON CLI for agents and automation",
	Long: `Read-only JSON interface for agents.

Always prints a single JSON object to stdout (success or failure). Does not
run login, sync, config set, alerts mutation, serve, service, or the TUI.

Uses the local SQLite cache by default, or a remote Whooper backend when
remote-url / WHOOPER_REMOTE_URL is configured.

Subcommands: summary, status, recovery, sleep, strain, workouts, doctor, schema.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return agentFail(cmd, "agent", agentSourceNone, agentClassInvalidArgs,
			errors.New("missing subcommand (try: whooper agent schema)"))
	},
}

func init() {
	agentCmd.PersistentFlags().StringVar(&agentFrom, "from", "", "Start date (YYYY-MM-DD)")
	agentCmd.PersistentFlags().StringVar(&agentTo, "to", "", "End date (YYYY-MM-DD)")
	agentCmd.PersistentFlags().IntVar(&agentLimit, "limit", defaultAPILimit, "Max rows for list commands (1-1000)")

	agentCmd.AddCommand(newAgentSummaryCmd())
	agentCmd.AddCommand(newAgentStatusCmd())
	agentCmd.AddCommand(newAgentEntityCmd("recovery", "recoveries", "Daily recovery scores"))
	agentCmd.AddCommand(newAgentEntityCmd("sleep", "sleeps", "Daily sleep summaries"))
	agentCmd.AddCommand(newAgentEntityCmd("strain", "cycles", "Daily strain (physiological cycles)"))
	agentCmd.AddCommand(newAgentEntityCmd("workouts", "workouts", "Workout summaries"))
	agentCmd.AddCommand(newAgentDoctorCmd())
	agentCmd.AddCommand(newAgentSchemaCmd())

	rootCmd.AddCommand(agentCmd)
}

func newAgentSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "summary",
		Short:         "Latest health metrics and last sync state",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgent(cmd, "summary", func() (any, string, error) {
				return agentFetchSummary()
			})
		},
	}
}

func newAgentStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "status",
		Short:         "Config, database, and sync status",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgent(cmd, "status", func() (any, string, error) {
				return agentFetchStatus()
			})
		},
	}
}

func newAgentEntityCmd(name, exportEntity, short string) *cobra.Command {
	return &cobra.Command{
		Use:           name,
		Short:         short,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgent(cmd, name, func() (any, string, error) {
				return agentFetchEntity(exportEntity)
			})
		},
	}
}

func newAgentDoctorCmd() *cobra.Command {
	c := &cobra.Command{
		Use:           "doctor",
		Short:         "Readiness checks (JSON); skips Whoop API by default",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgent(cmd, "doctor", func() (any, string, error) {
				return agentFetchDoctor(!agentDoctorAPI)
			})
		},
	}
	c.Flags().BoolVar(&agentDoctorAPI, "api", false, "Also check Whoop API reachability (requires local token)")
	return c
}

func newAgentSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "schema",
		Short:         "Describe agent commands and the JSON envelope",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgent(cmd, "schema", func() (any, string, error) {
				return agentSchema(), agentSourceNone, nil
			})
		},
	}
}

func runAgent(cmd *cobra.Command, name string, fn func() (data any, source string, err error)) error {
	data, source, err := fn()
	if err != nil {
		class, msg := classifyAgentError(err)
		if werr := writeAgentResponse(cmd.OutOrStdout(), agentResponse{
			OK:          false,
			Command:     name,
			Source:      source,
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Error:       &agentErrorBody{Class: class, Message: msg},
		}); werr != nil {
			return werr
		}
		code := 1
		if class == agentClassInvalidArgs {
			code = 2
		}
		return &agentExitError{code: code, err: err}
	}
	return writeAgentResponse(cmd.OutOrStdout(), agentResponse{
		OK:          true,
		Command:     name,
		Source:      source,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Data:        data,
	})
}

func agentFail(cmd *cobra.Command, name, source, class string, err error) error {
	msg := err.Error()
	if werr := writeAgentResponse(cmd.OutOrStdout(), agentResponse{
		OK:          false,
		Command:     name,
		Source:      source,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Error:       &agentErrorBody{Class: class, Message: msg},
	}); werr != nil {
		return werr
	}
	code := 1
	if class == agentClassInvalidArgs {
		code = 2
	}
	return &agentExitError{code: code, err: err}
}

func writeAgentResponse(w io.Writer, resp agentResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func classifyAgentError(err error) (class, message string) {
	if err == nil {
		return agentClassInternal, ""
	}
	message = err.Error()

	var re *remote.Error
	if errors.As(err, &re) {
		switch re.Kind {
		case remote.KindMissingToken:
			return agentClassMissingToken, re.Message
		case remote.KindUnauthorized:
			return agentClassUnauthorized, re.Message
		case remote.KindUnreachable:
			return agentClassUnreachable, re.Message
		case remote.KindHTTP, remote.KindDecode:
			return agentClassHTTP, re.Message
		default:
			return agentClassHTTP, re.Message
		}
	}

	// formatRemoteError wraps remote.Error as plain fmt.Errorf — match message prefixes.
	msg := message
	switch {
	case strings.Contains(msg, "missing_token"):
		return agentClassMissingToken, message
	case strings.Contains(msg, "unauthorized"):
		return agentClassUnauthorized, message
	case strings.Contains(msg, "unreachable"):
		return agentClassUnreachable, message
	case strings.Contains(msg, "open database") || strings.Contains(msg, "unable to open database"):
		return agentClassMissingDB, message
	case strings.Contains(msg, "invalid") || strings.Contains(msg, "unknown entity") || strings.Contains(msg, "missing subcommand"):
		return agentClassInvalidArgs, message
	default:
		return agentClassInternal, message
	}
}

func agentFetchSummary() (any, string, error) {
	backend, remoteOK, err := resolveRemoteBackend()
	if err != nil {
		return nil, agentSourceNone, err
	}
	if remoteOK {
		var report statusReport
		if err := backend.Client.GetJSON("/api/summary", nil, &report); err != nil {
			return nil, agentSourceRemote, err
		}
		health := report.LatestHealth
		if health == nil {
			health = &healthReport{Values: map[string]float64{}, Timestamps: map[string]string{}}
		}
		syncStates := map[string]string{}
		for _, entity := range statusEntities() {
			last := ""
			if report.LastSync != nil {
				last = report.LastSync[entity]
			}
			if last == "" {
				last = "never"
			}
			syncStates[entity] = last
		}
		return map[string]any{
			"latest_health": health,
			"last_sync":     syncStates,
		}, agentSourceRemote, nil
	}

	db, err := store.OpenReadOnly(config.DBPath())
	if err != nil {
		return nil, agentSourceLocal, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	health, err := latestHealthReport(db)
	if err != nil {
		return nil, agentSourceLocal, err
	}
	syncStates := map[string]string{}
	for _, entity := range statusEntities() {
		last, err := db.GetSyncState(entity)
		if err != nil {
			continue
		}
		if last == "" {
			last = "never"
		}
		syncStates[entity] = last
	}
	return map[string]any{
		"latest_health": health,
		"last_sync":     syncStates,
	}, agentSourceLocal, nil
}

func agentFetchStatus() (any, string, error) {
	backend, remoteOK, err := resolveRemoteBackend()
	if err != nil {
		return nil, agentSourceNone, err
	}
	if remoteOK {
		var report statusReport
		if err := backend.Client.GetJSON("/status", nil, &report); err != nil {
			return nil, agentSourceRemote, err
		}
		return report, agentSourceRemote, nil
	}
	return buildStatusReport(), agentSourceLocal, nil
}

func agentFetchEntity(exportEntity string) (any, string, error) {
	if err := validateDateRange(agentFrom, agentTo); err != nil {
		return nil, agentSourceNone, err
	}
	limit := agentLimit
	if limit <= 0 {
		return nil, agentSourceNone, fmt.Errorf("invalid limit %d", limit)
	}
	if limit > maxAPILimit {
		limit = maxAPILimit
	}

	backend, remoteOK, err := resolveRemoteBackend()
	if err != nil {
		return nil, agentSourceNone, err
	}
	if remoteOK {
		path, err := entityAPIPath(exportEntity)
		if err != nil {
			return nil, agentSourceRemote, err
		}
		q := remoteQuery(agentFrom, agentTo, limit)
		var rows []map[string]any
		if err := backend.Client.GetJSON(path, q, &rows); err != nil {
			return nil, agentSourceRemote, err
		}
		return map[string]any{
			"entity": exportEntity,
			"from":   agentFrom,
			"to":     agentTo,
			"limit":  limit,
			"count":  len(rows),
			"rows":   rows,
		}, agentSourceRemote, nil
	}

	from, to, err := exportDateBounds(agentFrom, agentTo)
	if err != nil {
		return nil, agentSourceLocal, err
	}
	db, err := store.OpenReadOnly(config.DBPath())
	if err != nil {
		return nil, agentSourceLocal, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	var data any
	switch exportEntity {
	case "cycles":
		data, err = db.ListCycles(from, to)
	case "recoveries":
		data, err = db.ListRecoveries(from, to)
	case "sleeps":
		data, err = db.ListSleeps(from, to, false)
	case "workouts":
		data, err = db.ListWorkouts(from, to)
	default:
		return nil, agentSourceLocal, fmt.Errorf("unknown entity %q", exportEntity)
	}
	if err != nil {
		return nil, agentSourceLocal, err
	}

	// Apply limit client-side for local lists (views are not pre-limited).
	data = limitAnySlice(data, limit)
	count := anySliceLen(data)

	return map[string]any{
		"entity": exportEntity,
		"from":   agentFrom,
		"to":     agentTo,
		"limit":  limit,
		"count":  count,
		"rows":   data,
	}, agentSourceLocal, nil
}

func agentFetchDoctor(skipAPI bool) (any, string, error) {
	// Doctor always inspects local config/token/db paths (not remote Whoop data).
	backend, remoteOK, err := resolveRemoteBackend()
	if err != nil {
		return nil, agentSourceNone, err
	}
	source := agentSourceLocal
	if remoteOK {
		source = agentSourceRemote
		_ = backend
		// When remote is configured, skip Whoop API reachability by default semantics.
		skipAPI = true
	}
	report := buildDoctorReport(defaultDoctorDeps(), skipAPI)
	payload := any(report)
	if remoteOK {
		payload = map[string]any{
			"note":   "doctor checks local paths only; data commands use the remote backend",
			"remote": true,
			"report": report,
		}
	}
	if !report.OK {
		return payload, source, fmt.Errorf("doctor found %d issue(s)", report.Failures)
	}
	return payload, source, nil
}

func agentSchema() any {
	return map[string]any{
		"envelope": map[string]any{
			"ok":           "bool — overall success",
			"command":      "string — subcommand name",
			"source":       "string — local | remote (when applicable)",
			"generated_at": "RFC3339 UTC timestamp",
			"data":         "object — success payload",
			"error":        "object — {class, message} on failure",
		},
		"error_classes": []string{
			agentClassInvalidArgs,
			agentClassMissingDB,
			agentClassMissingToken,
			agentClassUnauthorized,
			agentClassUnreachable,
			agentClassHTTP,
			agentClassInternal,
		},
		"exit_codes": map[string]int{
			"ok":           0,
			"app_error":    1,
			"invalid_args": 2,
		},
		"commands": []map[string]any{
			{"name": "summary", "flags": []string{}, "description": "Latest health metrics and last sync"},
			{"name": "status", "flags": []string{}, "description": "Config, DB, and sync status"},
			{"name": "recovery", "flags": []string{"--from", "--to", "--limit"}, "description": "Daily recovery rows"},
			{"name": "sleep", "flags": []string{"--from", "--to", "--limit"}, "description": "Daily sleep rows"},
			{"name": "strain", "flags": []string{"--from", "--to", "--limit"}, "description": "Daily strain rows"},
			{"name": "workouts", "flags": []string{"--from", "--to", "--limit"}, "description": "Workout rows"},
			{"name": "doctor", "flags": []string{"--api"}, "description": "Readiness checks; skips Whoop API unless --api"},
			{"name": "schema", "flags": []string{}, "description": "This catalog"},
		},
		"read_only": true,
		"notes": []string{
			"stdout is always one JSON document; secrets are never printed",
			"remote-url / WHOOPER_REMOTE_URL selects remote HTTP backend for data commands",
			"login, sync, config set, alerts mutation, serve, and tui are not available under agent",
		},
	}
}

func limitAnySlice(data any, limit int) any {
	if limit <= 0 {
		return data
	}
	switch v := data.(type) {
	case []map[string]any:
		if len(v) > limit {
			return v[:limit]
		}
		return v
	case []models.Cycle:
		if len(v) > limit {
			return v[:limit]
		}
		return v
	case []models.Recovery:
		if len(v) > limit {
			return v[:limit]
		}
		return v
	case []models.Sleep:
		if len(v) > limit {
			return v[:limit]
		}
		return v
	case []models.Workout:
		if len(v) > limit {
			return v[:limit]
		}
		return v
	default:
		return data
	}
}

func anySliceLen(data any) int {
	switch v := data.(type) {
	case []map[string]any:
		return len(v)
	case []models.Cycle:
		return len(v)
	case []models.Recovery:
		return len(v)
	case []models.Sleep:
		return len(v)
	case []models.Workout:
		return len(v)
	default:
		return 0
	}
}
