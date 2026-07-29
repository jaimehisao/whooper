package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/remote"
)

// remoteNewClient is a test seam for constructing remote HTTP clients.
var remoteNewClient = func(baseURL, token string) *remote.Client {
	return remote.New(baseURL, token)
}

// remoteHTTPClient is injected onto clients in tests (optional).
var remoteHTTPClient *http.Client

type remoteBackend struct {
	URL    string
	Token  string
	Client *remote.Client
}

// resolveRemoteBackend returns a configured remote client when remote mode is
// enabled (URL from config or WHOOPER_REMOTE_URL). ok is false for local mode.
func resolveRemoteBackend() (remoteBackend, bool, error) {
	cfg, err := config.Load()
	if err != nil {
		return remoteBackend{}, false, fmt.Errorf("load config: %w", err)
	}
	baseURL := config.ResolvedRemoteURL(cfg)
	if baseURL == "" {
		return remoteBackend{}, false, nil
	}
	token := config.ResolvedRemoteToken(cfg)
	client := remoteNewClient(baseURL, token)
	if remoteHTTPClient != nil {
		client.HTTPClient = remoteHTTPClient
	}
	return remoteBackend{URL: baseURL, Token: token, Client: client}, true, nil
}

func formatRemoteError(err error) error {
	if err == nil {
		return nil
	}
	var re *remote.Error
	if errors.As(err, &re) {
		switch re.Kind {
		case remote.KindMissingToken:
			return fmt.Errorf("remote missing_token: %s", re.Message)
		case remote.KindUnauthorized:
			return fmt.Errorf("remote unauthorized: %s", re.Message)
		case remote.KindUnreachable:
			return fmt.Errorf("remote unreachable: %s", re.Message)
		default:
			return fmt.Errorf("remote %s: %s", re.Kind, re.Message)
		}
	}
	return fmt.Errorf("remote: %w", err)
}

// entityAPIPath maps export entity names to serve HTTP paths.
// cycles use /api/strain (daily strain view); recoveries/sleeps/workouts match views.
func entityAPIPath(entity string) (string, error) {
	switch entity {
	case "recoveries":
		return "/api/recovery", nil
	case "sleeps":
		return "/api/sleep", nil
	case "cycles", "strain":
		return "/api/strain", nil
	case "workouts":
		return "/api/workouts", nil
	default:
		return "", fmt.Errorf("unknown entity %q (valid: cycles, recoveries, sleeps, workouts)", entity)
	}
}

func remoteQuery(from, to string, limit int) url.Values {
	q := url.Values{}
	if from != "" {
		q.Set("from", from)
	}
	if to != "" {
		q.Set("to", to)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	return q
}

// remoteLocalOnlyHint is appended when a local-only command is used while remote is configured.
func remoteLocalOnlyHint(command string) string {
	return fmt.Sprintf(
		"%s uses local Whoop credentials and SQLite; it does not call the remote backend.\nHint: unset remote-url / WHOOPER_REMOTE_URL for a fully local workflow, or run %s on the server host.",
		command, command,
	)
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}
