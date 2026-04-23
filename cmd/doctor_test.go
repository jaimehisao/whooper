package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"golang.org/x/oauth2"
)

type fakeDoctorDB struct {
	pingErr error
	closed  bool
}

func (f *fakeDoctorDB) Ping() error { return f.pingErr }
func (f *fakeDoctorDB) Close() error {
	f.closed = true
	return nil
}

func TestRunDoctorSuccess(t *testing.T) {
	db := &fakeDoctorDB{}
	deps := doctorDeps{
		loadConfig: func() (*config.Config, error) {
			return &config.Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost:8484/callback"}, nil
		},
		validateRedirect: func(string) error { return nil },
		loadToken: func(string) (*oauth2.Token, error) {
			return &oauth2.Token{AccessToken: "abc"}, nil
		},
		tokenPath: func() string { return "token.json" },
		openDB: func(string) (doctorDB, error) {
			return db, nil
		},
		dbPath:   func() string { return "whooper.db" },
		apiCheck: func(*config.Config, *oauth2.Token) error { return nil },
	}

	var out bytes.Buffer
	err := runDoctor(&out, deps, false)
	if err != nil {
		t.Fatalf("runDoctor error = %v", err)
	}

	if !strings.Contains(out.String(), "Doctor checks passed.") {
		t.Fatalf("expected success message, got: %s", out.String())
	}
	if !db.closed {
		t.Fatal("expected database to be closed")
	}
}

func TestRunDoctorFailuresAreReported(t *testing.T) {
	deps := doctorDeps{
		loadConfig: func() (*config.Config, error) {
			return &config.Config{RedirectURL: "bad://url"}, nil
		},
		validateRedirect: func(string) error { return errors.New("bad redirect") },
		loadToken:        func(string) (*oauth2.Token, error) { return nil, errors.New("missing token") },
		tokenPath:        func() string { return "token.json" },
		openDB:           func(string) (doctorDB, error) { return nil, errors.New("db open failed") },
		dbPath:           func() string { return "whooper.db" },
		apiCheck:         func(*config.Config, *oauth2.Token) error { return nil },
	}

	var out bytes.Buffer
	err := runDoctor(&out, deps, false)
	if err == nil {
		t.Fatal("expected error from runDoctor")
	}
	if !strings.Contains(err.Error(), "doctor found") {
		t.Fatalf("unexpected error: %v", err)
	}

	text := out.String()
	if !strings.Contains(text, "[fail] client-id configured") {
		t.Fatalf("expected client-id failure in output, got: %s", text)
	}
	if !strings.Contains(text, "[fail] Whoop API reachability") {
		t.Fatalf("expected api failure in output, got: %s", text)
	}
}

func TestRunDoctorSkipAPI(t *testing.T) {
	db := &fakeDoctorDB{}
	calledAPI := false
	deps := doctorDeps{
		loadConfig: func() (*config.Config, error) {
			return &config.Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost:8484/callback"}, nil
		},
		validateRedirect: func(string) error { return nil },
		loadToken: func(string) (*oauth2.Token, error) {
			return &oauth2.Token{AccessToken: "abc"}, nil
		},
		tokenPath: func() string { return "token.json" },
		openDB: func(string) (doctorDB, error) {
			return db, nil
		},
		dbPath: func() string { return "whooper.db" },
		apiCheck: func(*config.Config, *oauth2.Token) error {
			calledAPI = true
			return nil
		},
	}

	var out bytes.Buffer
	err := runDoctor(&out, deps, true)
	if err != nil {
		t.Fatalf("runDoctor error = %v", err)
	}
	if calledAPI {
		t.Fatal("api check should not run when --skip-api is set")
	}
	if !strings.Contains(out.String(), "[skip] Whoop API reachability") {
		t.Fatalf("expected skip message, got: %s", out.String())
	}
}

func TestRunDoctorLoadConfigFailure(t *testing.T) {
	deps := doctorDeps{
		loadConfig:       func() (*config.Config, error) { return nil, errors.New("boom") },
		validateRedirect: func(string) error { return nil },
		loadToken:        func(string) (*oauth2.Token, error) { return nil, nil },
		tokenPath:        func() string { return "token.json" },
		openDB:           func(string) (doctorDB, error) { return &fakeDoctorDB{}, nil },
		dbPath:           func() string { return "whooper.db" },
		apiCheck:         func(*config.Config, *oauth2.Token) error { return nil },
	}

	err := runDoctor(&bytes.Buffer{}, deps, false)
	if err == nil {
		t.Fatal("expected error from runDoctor")
	}
}
