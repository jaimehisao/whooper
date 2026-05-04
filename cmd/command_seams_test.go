package cmd

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/oauth2"

	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	gosync "git.infra.hisao.org/hisao/whooper/internal/sync"
	"git.infra.hisao.org/hisao/whooper/internal/tui"
)

type fakeSyncFromRunner struct {
	start string
	err   error
	calls int
}

func (f *fakeSyncFromRunner) SyncFrom(start string) error {
	f.start = start
	f.calls++
	return f.err
}

type fakeSyncAllRunner struct {
	called bool
	err    error
}

func (f *fakeSyncAllRunner) SyncAll() error {
	f.called = true
	return f.err
}

func TestLoginRunE_SaveTokenError(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))
	if err := config.Save(&config.Config{ClientID: "id", ClientSecret: "secret"}); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	origOAuthFlow := oauthFlowFunc
	origSaveToken := saveTokenFunc
	defer func() {
		oauthFlowFunc = origOAuthFlow
		saveTokenFunc = origSaveToken
	}()

	oauthFlowFunc = func(*oauth2.Config) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "token"}, nil
	}
	saveErr := errors.New("disk full")
	saveTokenFunc = func(string, *oauth2.Token) error {
		return saveErr
	}

	err := loginCmd.RunE(loginCmd, nil)
	if !errors.Is(err, saveErr) {
		t.Fatalf("loginCmd.RunE error = %v, want save error", err)
	}
}

func TestSyncRunE_UsesInjectedSyncerAndSince(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	fake := &fakeSyncFromRunner{}
	origLoadConfig := syncLoadConfig
	origLoadToken := syncLoadToken
	origOpenDB := syncOpenDB
	origNewClient := syncNewClient
	origNewSyncer := syncNewSyncer
	origFull := syncFull
	origSince := syncSince
	origDebug := syncDebug
	defer func() {
		syncLoadConfig = origLoadConfig
		syncLoadToken = origLoadToken
		syncOpenDB = origOpenDB
		syncNewClient = origNewClient
		syncNewSyncer = origNewSyncer
		syncFull = origFull
		syncSince = origSince
		syncDebug = origDebug
	}()

	syncLoadConfig = func() (*config.Config, error) {
		return &config.Config{ClientID: "id", ClientSecret: "secret"}, nil
	}
	syncLoadToken = func(string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "token"}, nil
	}
	syncOpenDB = func(string) (*store.DB, error) {
		return db, nil
	}
	syncNewClient = func(oauth2.TokenSource) *api.Client {
		return &api.Client{}
	}
	syncNewSyncer = func(*api.Client, *store.DB, gosync.ProgressFunc) syncRunner {
		return fake
	}
	syncFull = false
	syncSince = "2024-01-15"
	syncDebug = true

	var out bytes.Buffer
	syncCmd.SetOut(&out)
	defer syncCmd.SetOut(nil)

	if err := syncCmd.RunE(syncCmd, nil); err != nil {
		t.Fatalf("syncCmd.RunE error = %v", err)
	}
	if fake.start != "2024-01-15T00:00:00.000Z" {
		t.Fatalf("SyncFrom start = %q, want since start", fake.start)
	}
	if !strings.Contains(out.String(), "[debug] sync start=") {
		t.Fatalf("expected debug sync output, got:\n%s", out.String())
	}
}

func TestSyncRunE_LoopRunsOnInterval(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	fake := &fakeSyncFromRunner{}
	origLoadConfig := syncLoadConfig
	origLoadToken := syncLoadToken
	origOpenDB := syncOpenDB
	origNewClient := syncNewClient
	origNewSyncer := syncNewSyncer
	origLoop := syncLoop
	origInterval := syncInterval
	origIterations := syncLoopIterations
	origSleep := syncSleep
	origFull := syncFull
	origSince := syncSince
	defer func() {
		syncLoadConfig = origLoadConfig
		syncLoadToken = origLoadToken
		syncOpenDB = origOpenDB
		syncNewClient = origNewClient
		syncNewSyncer = origNewSyncer
		syncLoop = origLoop
		syncInterval = origInterval
		syncLoopIterations = origIterations
		syncSleep = origSleep
		syncFull = origFull
		syncSince = origSince
	}()

	syncLoadConfig = func() (*config.Config, error) {
		return &config.Config{ClientID: "id", ClientSecret: "secret"}, nil
	}
	syncLoadToken = func(string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "token"}, nil
	}
	syncOpenDB = func(string) (*store.DB, error) {
		return db, nil
	}
	syncNewClient = func(oauth2.TokenSource) *api.Client {
		return &api.Client{}
	}
	syncNewSyncer = func(*api.Client, *store.DB, gosync.ProgressFunc) syncRunner {
		return fake
	}
	syncLoop = true
	syncInterval = time.Second
	syncLoopIterations = 2
	syncFull = false
	syncSince = ""

	sleepCalls := 0
	syncSleep = func(d time.Duration) {
		sleepCalls++
		if d != time.Second {
			t.Fatalf("sleep duration = %s, want 1s", d)
		}
	}

	var out bytes.Buffer
	syncCmd.SetOut(&out)
	defer syncCmd.SetOut(nil)

	if err := syncCmd.RunE(syncCmd, nil); err != nil {
		t.Fatalf("syncCmd.RunE error = %v", err)
	}
	if fake.calls != 2 {
		t.Fatalf("SyncFrom calls = %d, want 2", fake.calls)
	}
	if sleepCalls != 1 {
		t.Fatalf("sleep calls = %d, want 1", sleepCalls)
	}
	if !strings.Contains(out.String(), "Next sync in 1s") {
		t.Fatalf("expected loop output, got:\n%s", out.String())
	}
}

func TestSyncRunE_UnauthorizedSuggestsLogin(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	fake := &fakeSyncFromRunner{err: &api.StatusError{Endpoint: "/v2/cycle", Status: 401, Body: "unauthorized"}}
	origLoadConfig := syncLoadConfig
	origLoadToken := syncLoadToken
	origOpenDB := syncOpenDB
	origNewClient := syncNewClient
	origNewSyncer := syncNewSyncer
	origLoop := syncLoop
	defer func() {
		syncLoadConfig = origLoadConfig
		syncLoadToken = origLoadToken
		syncOpenDB = origOpenDB
		syncNewClient = origNewClient
		syncNewSyncer = origNewSyncer
		syncLoop = origLoop
	}()

	syncLoadConfig = func() (*config.Config, error) {
		return &config.Config{ClientID: "id", ClientSecret: "secret"}, nil
	}
	syncLoadToken = func(string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "token"}, nil
	}
	syncOpenDB = func(string) (*store.DB, error) {
		return db, nil
	}
	syncNewClient = func(oauth2.TokenSource) *api.Client {
		return &api.Client{}
	}
	syncNewSyncer = func(*api.Client, *store.DB, gosync.ProgressFunc) syncRunner {
		return fake
	}
	syncLoop = false

	err = syncCmd.RunE(syncCmd, nil)
	if err == nil {
		t.Fatal("expected unauthorized sync error")
	}
	if !strings.Contains(err.Error(), "run 'whooper login' again") {
		t.Fatalf("expected login hint, got: %v", err)
	}
}

func TestTuiRunE_UsesInjectedRunnerAndSyncFunction(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	fake := &fakeSyncAllRunner{}
	origOpenDB := tuiOpenDB
	origLoadConfig := tuiLoadConfig
	origLoadToken := tuiLoadToken
	origNewClient := tuiNewClient
	origNewSyncer := tuiNewSyncer
	origRunApp := tuiRunApp
	defer func() {
		tuiOpenDB = origOpenDB
		tuiLoadConfig = origLoadConfig
		tuiLoadToken = origLoadToken
		tuiNewClient = origNewClient
		tuiNewSyncer = origNewSyncer
		tuiRunApp = origRunApp
	}()

	tuiOpenDB = func(string) (*store.DB, error) {
		return db, nil
	}
	tuiLoadConfig = func() (*config.Config, error) {
		return &config.Config{ClientID: "id", ClientSecret: "secret"}, nil
	}
	tuiLoadToken = func(string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "token"}, nil
	}
	tuiNewClient = func(oauth2.TokenSource) *api.Client {
		return &api.Client{}
	}
	tuiNewSyncer = func(*api.Client, *store.DB) tuiSyncRunner {
		return fake
	}
	tuiRunApp = func(app *tui.App) error {
		_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
		if cmd == nil {
			t.Fatal("expected sync command")
		}
		cmd()
		return nil
	}

	if err := tuiCmd.RunE(tuiCmd, nil); err != nil {
		t.Fatalf("tuiCmd.RunE error = %v", err)
	}
	if !fake.called {
		t.Fatal("expected injected TUI syncer to run")
	}
}

func TestTuiRunE_RequiresSetupBeforeLaunching(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))

	origOpenDB := tuiOpenDB
	origLoadConfig := tuiLoadConfig
	origLoadToken := tuiLoadToken
	defer func() {
		tuiOpenDB = origOpenDB
		tuiLoadConfig = origLoadConfig
		tuiLoadToken = origLoadToken
	}()

	tuiOpenDB = func(string) (*store.DB, error) {
		t.Fatal("database should not open when setup is missing")
		return nil, nil
	}
	tuiLoadConfig = config.Load
	tuiLoadToken = auth.LoadToken

	err := tuiCmd.RunE(tuiCmd, nil)
	if err == nil {
		t.Fatal("expected setup error")
	}
	if !strings.Contains(err.Error(), "setup required before launching the dashboard") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootRunEDelegatesToTUI(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	origOpenDB := tuiOpenDB
	origLoadConfig := tuiLoadConfig
	origLoadToken := tuiLoadToken
	origNewClient := tuiNewClient
	origNewSyncer := tuiNewSyncer
	origRunApp := tuiRunApp
	defer func() {
		tuiOpenDB = origOpenDB
		tuiLoadConfig = origLoadConfig
		tuiLoadToken = origLoadToken
		tuiNewClient = origNewClient
		tuiNewSyncer = origNewSyncer
		tuiRunApp = origRunApp
	}()

	tuiOpenDB = func(string) (*store.DB, error) {
		return db, nil
	}
	tuiLoadConfig = func() (*config.Config, error) {
		return &config.Config{ClientID: "id", ClientSecret: "secret"}, nil
	}
	tuiLoadToken = func(string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "token"}, nil
	}
	tuiNewClient = func(oauth2.TokenSource) *api.Client {
		return &api.Client{}
	}
	tuiNewSyncer = func(*api.Client, *store.DB) tuiSyncRunner {
		return &fakeSyncAllRunner{}
	}
	called := false
	tuiRunApp = func(*tui.App) error {
		called = true
		return nil
	}

	if err := rootCmd.RunE(rootCmd, nil); err != nil {
		t.Fatalf("rootCmd.RunE error = %v", err)
	}
	if !called {
		t.Fatal("expected root command to delegate to TUI runner")
	}
}
