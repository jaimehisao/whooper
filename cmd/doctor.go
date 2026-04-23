package cmd

import (
	"context"
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

func defaultDoctorDeps() doctorDeps {
	return doctorDeps{
		loadConfig:       config.Load,
		validateRedirect: auth.ValidateRedirectURL,
		loadToken:        auth.LoadToken,
		tokenPath:        config.TokenPath,
		openDB: func(path string) (doctorDB, error) {
			return store.Open(path)
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
	failures := 0
	report := func(ok bool, name string, err error) {
		if ok {
			fmt.Fprintf(out, "[ok]   %s\n", name)
			return
		}
		failures++
		fmt.Fprintf(out, "[fail] %s: %v\n", name, err)
	}

	cfg, err := deps.loadConfig()
	if err != nil {
		report(false, "load config", err)
		return fmt.Errorf("doctor found %d issue(s)", failures)
	}
	report(true, "load config", nil)

	if skipAPI {
		fmt.Fprintln(out, "[skip] client-id configured (--skip-api)")
	} else {
		if cfg.ClientID == "" {
			report(false, "client-id configured", fmt.Errorf("client_id is empty"))
		} else {
			report(true, "client-id configured", nil)
		}
	}

	if skipAPI {
		fmt.Fprintln(out, "[skip] client-secret configured (--skip-api)")
	} else {
		if cfg.ClientSecret == "" {
			report(false, "client-secret configured", fmt.Errorf("client_secret is empty"))
		} else {
			report(true, "client-secret configured", nil)
		}
	}

	if err := deps.validateRedirect(cfg.RedirectURL); err != nil {
		report(false, "redirect URL", err)
	} else {
		report(true, "redirect URL", nil)
	}

	token, err := deps.loadToken(deps.tokenPath())
	if skipAPI {
		fmt.Fprintln(out, "[skip] load token (--skip-api)")
	} else if err != nil {
		report(false, "load token", err)
	} else {
		report(true, "load token", nil)
	}

	db, err := deps.openDB(deps.dbPath())
	if err != nil {
		report(false, "open database", err)
	} else {
		if pingErr := db.Ping(); pingErr != nil {
			report(false, "database ping", pingErr)
		} else {
			report(true, "database ping", nil)
		}
		_ = db.Close()
	}

	if skipAPI {
		fmt.Fprintln(out, "[skip] Whoop API reachability (--skip-api)")
	} else if token == nil {
		report(false, "Whoop API reachability", fmt.Errorf("token unavailable"))
	} else {
		if err := deps.apiCheck(cfg, token); err != nil {
			report(false, "Whoop API reachability", err)
		} else {
			report(true, "Whoop API reachability", nil)
		}
	}

	if failures > 0 {
		return fmt.Errorf("doctor found %d issue(s)", failures)
	}

	fmt.Fprintln(out, "Doctor checks passed.")
	return nil
}

var doctorSkipAPI bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run production-readiness smoke checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor(cmd.OutOrStdout(), defaultDoctorDeps(), doctorSkipAPI)
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorSkipAPI, "skip-api", false, "Skip Whoop API reachability check")
	rootCmd.AddCommand(doctorCmd)
}
