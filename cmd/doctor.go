package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type doctorDB interface {
	Ping() error
	Close() error
}

type doctorDeps struct {
	loadConfig       func() (*config.Config, error)
	validateRedirect func(string) error
	loadToken        func(string) (*oauth2.Token, error)
	tokenPath        func() string
	openDB           func(string) (doctorDB, error)
	dbPath           func() string
	apiCheck         func(*config.Config, *oauth2.Token) error
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type doctorReport struct {
	OK       bool          `json:"ok"`
	Failures int           `json:"failures"`
	Checks   []doctorCheck `json:"checks"`
}

func defaultDoctorDeps() doctorDeps {
	return doctorDeps{
		loadConfig:       config.Load,
		validateRedirect: auth.ValidateRedirectURL,
		loadToken:        auth.LoadToken,
		tokenPath:        config.TokenPath,
		openDB: func(path string) (doctorDB, error) {
			return store.OpenReadOnly(path)
		},
		dbPath: config.DBPath,
		apiCheck: func(cfg *config.Config, token *oauth2.Token) error {
			oauthCfg := auth.OAuthConfig(cfg)
			tokenSource := auth.PersistingTokenSource(
				config.TokenPath(),
				oauthCfg.TokenSource(context.Background(), token),
			)
			client := api.NewClient(tokenSource)
			_, err := client.GetProfile()
			return err
		},
	}
}

func runDoctor(out io.Writer, deps doctorDeps, skipAPI bool) error {
	return runDoctorWithFormat(out, deps, skipAPI, false)
}

func runDoctorWithFormat(out io.Writer, deps doctorDeps, skipAPI, jsonOutput bool) error {
	report := buildDoctorReport(deps, skipAPI)
	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	} else {
		writeDoctorText(out, report)
	}

	if !report.OK {
		return fmt.Errorf("doctor found %d issue(s)", report.Failures)
	}
	return nil
}

func buildDoctorReport(deps doctorDeps, skipAPI bool) doctorReport {
	report := doctorReport{OK: true}
	failures := 0
	add := func(ok bool, name string, err error) {
		check := doctorCheck{Name: name}
		if ok {
			check.Status = "ok"
		} else {
			failures++
			check.Status = "fail"
			check.Error = err.Error()
		}
		report.Checks = append(report.Checks, check)
	}
	skip := func(name string) {
		report.Checks = append(report.Checks, doctorCheck{Name: name, Status: "skip"})
	}

	cfg, err := deps.loadConfig()
	if err != nil {
		add(false, "load config", fmt.Errorf("%w\nHint: make sure config file exists and is valid", err))
		report.OK = false
		report.Failures = failures
		return report
	}
	add(true, "load config", nil)

	if skipAPI {
		skip("client-id configured")
	} else {
		if cfg.ClientID == "" {
			add(false, "client-id configured", fmt.Errorf("client_id is empty\nHint: run 'whooper config set client-id <id>'"))
		} else {
			add(true, "client-id configured", nil)
		}
	}

	if skipAPI {
		skip("client-secret configured")
	} else {
		if cfg.ClientSecret == "" {
			add(false, "client-secret configured", fmt.Errorf("client_secret is empty\nHint: run 'whooper config set client-secret <secret>'"))
		} else {
			add(true, "client-secret configured", nil)
		}
	}

	if err := deps.validateRedirect(cfg.RedirectURL); err != nil {
		add(false, "redirect URL", fmt.Errorf("%w\nHint: run 'whooper config set redirect-url <url>'", err))
	} else {
		add(true, "redirect URL", nil)
	}

	token, err := deps.loadToken(deps.tokenPath())
	if skipAPI {
		skip("load token")
	} else if err != nil {
		add(false, "load token", fmt.Errorf("%w\nHint: run 'whooper login' to authenticate", err))
	} else {
		add(true, "load token", nil)
	}

	db, err := deps.openDB(deps.dbPath())
	if err != nil {
		add(false, "open database", fmt.Errorf("%w\nHint: run 'whooper sync' or 'whooper login' to initialize the database", err))
	} else {
		if pingErr := db.Ping(); pingErr != nil {
			add(false, "database ping", fmt.Errorf("%w\nHint: the database file may be corrupt or inaccessible", pingErr))
		} else {
			add(true, "database ping", nil)
		}
		_ = db.Close()
	}

	if skipAPI {
		skip("Whoop API reachability")
	} else if token == nil {
		add(false, "Whoop API reachability", fmt.Errorf("token unavailable\nHint: run 'whooper login'"))
	} else {
		if err := deps.apiCheck(cfg, token); err != nil {
			add(false, "Whoop API reachability", fmt.Errorf("%w\nHint: check your internet connection and API credentials", err))
		} else {
			add(true, "Whoop API reachability", nil)
		}
	}

	report.Failures = failures
	report.OK = failures == 0
	return report
}

func writeDoctorText(out io.Writer, report doctorReport) {
	for _, check := range report.Checks {
		switch check.Status {
		case "ok":
			fmt.Fprintf(out, "[ok]   %s\n", check.Name)
		case "skip":
			fmt.Fprintf(out, "[skip] %s\n", check.Name)
		default:
			fmt.Fprintf(out, "[fail] %s: %s\n", check.Name, check.Error)
		}
	}

	if report.OK {
		fmt.Fprintln(out, "Doctor checks passed.")
	}
}

var doctorSkipAPI bool
var doctorJSON bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run production-readiness smoke checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctorWithFormat(cmd.OutOrStdout(), defaultDoctorDeps(), doctorSkipAPI, doctorJSON)
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorSkipAPI, "skip-api", false, "Skip Whoop API reachability check")
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output doctor results as JSON")
	rootCmd.AddCommand(doctorCmd)
}
