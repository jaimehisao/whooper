package cmd

import (
	"bytes"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	gosync "git.infra.hisao.org/hisao/whooper/internal/sync"
	"golang.org/x/oauth2"
)

func TestServiceRunE_StartsServerAndSyncs(t *testing.T) {
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
	origListenAndServe := serveListenAndServe
	origSleep := syncSleep
	origIterations := syncLoopIterations

	// Sync globals mapped by service
	origSyncFull := syncFull
	origSyncSince := syncSince
	origSyncDebug := syncDebug

	// Flags
	origAddr := serviceAddr
	origInterval := serviceInterval
	origSince := serviceSince
	origFull := serviceFull
	origDebug := serviceDebug

	defer func() {
		syncLoadConfig = origLoadConfig
		syncLoadToken = origLoadToken
		syncOpenDB = origOpenDB
		syncNewClient = origNewClient
		syncNewSyncer = origNewSyncer
		serveListenAndServe = origListenAndServe
		syncSleep = origSleep
		syncLoopIterations = origIterations

		syncFull = origSyncFull
		syncSince = origSyncSince
		syncDebug = origSyncDebug

		serviceAddr = origAddr
		serviceInterval = origInterval
		serviceSince = origSince
		serviceFull = origFull
		serviceDebug = origDebug
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

	serverStarted := make(chan string, 1)
	var capturedHandler http.Handler
	serveListenAndServe = func(addr string, handler http.Handler) error {
		capturedHandler = handler
		serverStarted <- addr
		return http.ErrServerClosed
	}

	syncSleep = func(time.Duration) {}
	syncLoopIterations = 2

	serviceAddr = "127.0.0.1:1234"
	serviceInterval = 100 * time.Millisecond
	serviceSince = ""
	serviceFull = false
	serviceDebug = false

	var out bytes.Buffer
	serviceCmd.SetOut(&out)
	defer serviceCmd.SetOut(nil)

	if err := serviceCmd.RunE(serviceCmd, nil); err != nil {
		t.Fatalf("serviceCmd.RunE error = %v", err)
	}

	select {
	case addr := <-serverStarted:
		if addr != "127.0.0.1:1234" {
			t.Errorf("server started on %s, want 127.0.0.1:1234", addr)
		}
		if capturedHandler == nil {
			t.Error("server started with nil handler")
		}
	case <-time.After(time.Second):
		t.Fatal("server did not start")
	}

	if fake.calls != 2 {
		t.Errorf("sync calls = %d, want 2", fake.calls)
	}
}

func TestServiceRunE_SyncErrorDoesNotStopServer(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	fake := &fakeSyncFromRunner{err: errors.New("sync failed")}

	origLoadConfig := syncLoadConfig
	origLoadToken := syncLoadToken
	origOpenDB := syncOpenDB
	origNewSyncer := syncNewSyncer
	origListenAndServe := serveListenAndServe
	origSleep := syncSleep
	origIterations := syncLoopIterations

	origSyncFull := syncFull
	origSyncSince := syncSince
	origSyncDebug := syncDebug

	origAddr := serviceAddr
	origInterval := serviceInterval
	origSince := serviceSince
	origFull := serviceFull
	origDebug := serviceDebug

	defer func() {
		syncLoadConfig = origLoadConfig
		syncLoadToken = origLoadToken
		syncOpenDB = origOpenDB
		syncNewSyncer = origNewSyncer
		serveListenAndServe = origListenAndServe
		syncSleep = origSleep
		syncLoopIterations = origIterations

		syncFull = origSyncFull
		syncSince = origSyncSince
		syncDebug = origSyncDebug

		serviceAddr = origAddr
		serviceInterval = origInterval
		serviceSince = origSince
		serviceFull = origFull
		serviceDebug = origDebug
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
	syncNewSyncer = func(*api.Client, *store.DB, gosync.ProgressFunc) syncRunner {
		return fake
	}

	serveListenAndServe = func(string, http.Handler) error {
		time.Sleep(10 * time.Millisecond)
		return http.ErrServerClosed
	}

	syncSleep = func(time.Duration) {}
	syncLoopIterations = 2
	serviceInterval = time.Millisecond

	var out bytes.Buffer
	serviceCmd.SetOut(&out)
	defer serviceCmd.SetOut(nil)

	if err := serviceCmd.RunE(serviceCmd, nil); err != nil {
		t.Fatalf("serviceCmd.RunE error = %v", err)
	}

	// Update assertion to be less brittle: check for "Sync error:" and "sync failed"
	got := out.String()
	if !strings.Contains(got, "Sync error:") || !strings.Contains(got, "sync failed") {
		t.Errorf("expected sync error message in output, got:\n%s", got)
	}
	if fake.calls != 2 {
		t.Errorf("sync calls = %d, want 2", fake.calls)
	}
}

func TestServiceRunE_ServerErrorReturns(t *testing.T) {
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
	origNewSyncer := syncNewSyncer
	origListenAndServe := serveListenAndServe
	origSleep := syncSleep
	origIterations := syncLoopIterations

	origSyncFull := syncFull
	origSyncSince := syncSince
	origSyncDebug := syncDebug

	origAddr := serviceAddr
	origInterval := serviceInterval
	origSince := serviceSince
	origFull := serviceFull
	origDebug := serviceDebug

	defer func() {
		syncLoadConfig = origLoadConfig
		syncLoadToken = origLoadToken
		syncOpenDB = origOpenDB
		syncNewSyncer = origNewSyncer
		serveListenAndServe = origListenAndServe
		syncSleep = origSleep
		syncLoopIterations = origIterations

		syncFull = origSyncFull
		syncSince = origSyncSince
		syncDebug = origSyncDebug

		serviceAddr = origAddr
		serviceInterval = origInterval
		serviceSince = origSince
		serviceFull = origFull
		serviceDebug = origDebug
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
	syncNewSyncer = func(*api.Client, *store.DB, gosync.ProgressFunc) syncRunner {
		return fake
	}

	serverErr := errors.New("bind failed")
	serveListenAndServe = func(string, http.Handler) error {
		return serverErr
	}

	syncSleep = func(time.Duration) {}
	syncLoopIterations = 0 // loop forever
	serviceInterval = time.Millisecond

	err = serviceCmd.RunE(serviceCmd, nil)
	if !errors.Is(err, serverErr) {
		t.Fatalf("expected server error %v, got %v", serverErr, err)
	}
}

func TestServiceRunE_Validation(t *testing.T) {
	origSince := serviceSince
	origInterval := serviceInterval
	origSyncFull := syncFull
	origSyncSince := syncSince
	origSyncDebug := syncDebug

	defer func() {
		serviceSince = origSince
		serviceInterval = origInterval
		syncFull = origSyncFull
		syncSince = origSyncSince
		syncDebug = origSyncDebug
	}()

	t.Run("InvalidSince", func(t *testing.T) {
		serviceSince = "invalid"
		err := serviceCmd.RunE(serviceCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "must be YYYY-MM-DD") {
			t.Errorf("expected YYYY-MM-DD validation error, got %v", err)
		}
	})

	t.Run("ZeroInterval", func(t *testing.T) {
		serviceSince = ""
		serviceInterval = 0
		err := serviceCmd.RunE(serviceCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "interval must be greater than zero") {
			t.Errorf("expected interval validation error, got %v", err)
		}
	})
}
