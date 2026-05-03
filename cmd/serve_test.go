package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func TestServeHealthz(t *testing.T) {
	handler := newServeHandler(func() statusReport { return statusReport{} })

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("GET /healthz body = %q", rec.Body.String())
	}
}

func TestServeStatus(t *testing.T) {
	want := statusReport{
		ConfigPath:             "/tmp/config.yaml",
		DBPath:                 "/tmp/whooper.db",
		TokenPath:              "/tmp/token.json",
		ClientIDConfigured:     true,
		ClientSecretConfigured: true,
		RedirectURL:            "http://localhost:8484/callback",
		TokenPresent:           true,
		DBOpen:                 true,
		RecordCounts:           map[string]int{"cycles": 2},
		LastSync:               map[string]string{"cycles": "2024-01-02T00:00:00Z"},
	}
	handler := newServeHandler(func() statusReport { return want })

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /status status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got statusReport
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("GET /status JSON decode: %v\n%s", err, rec.Body.String())
	}
	if !got.ClientIDConfigured || !got.ClientSecretConfigured || !got.TokenPresent || !got.DBOpen {
		t.Fatalf("GET /status decoded flags = %+v", got)
	}
	if got.RecordCounts["cycles"] != 2 {
		t.Fatalf("GET /status cycles count = %d, want 2", got.RecordCounts["cycles"])
	}
}

func TestServeMetricsFromStatusReport(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)
	if err := config.Save(&config.Config{ClientID: "id", ClientSecret: "secret"}); err != nil {
		t.Fatalf("Save config: %v", err)
	}
	if err := auth.SaveToken(config.TokenPath(), &oauth2.Token{AccessToken: "token"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	if err := db.SaveCycles([]models.Cycle{{ID: 1, UserID: 1, Start: "2024-01-01T00:00:00Z"}}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}
	if err := db.SetSyncState("cycles", "2024-01-02T00:00:00Z"); err != nil {
		t.Fatalf("SetSyncState: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close DB: %v", err)
	}

	handler := newServeHandler(buildServeStatusReport)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"whooper_db_open 1",
		"whooper_token_present 1",
		"whooper_client_id_configured 1",
		"whooper_client_secret_configured 1",
		`whooper_records_total{entity="cycles"} 1`,
		`whooper_last_sync_timestamp_seconds{entity="cycles"} 1.7041536e+09`,
		`whooper_records_total{entity="recoveries"} 0`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /metrics missing %q:\n%s", want, body)
		}
	}
}

func TestBuildServeStatusReportDoesNotCreateDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "missing.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)

	report := buildServeStatusReport()
	if report.DBOpen {
		t.Fatalf("DBOpen = true for missing read-only database: %+v", report)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("serve status should not create database, stat err = %v", err)
	}
}

func TestServeCommandUsesAddrFlag(t *testing.T) {
	origAddr := serveAddr
	origListenAndServe := serveListenAndServe
	defer func() {
		serveAddr = origAddr
		serveListenAndServe = origListenAndServe
		_ = serveCmd.Flags().Set("addr", origAddr)
		serveCmd.SetOut(nil)
	}()

	var gotAddr string
	serveListenAndServe = func(addr string, handler http.Handler) error {
		gotAddr = addr
		if handler == nil {
			t.Fatal("serve handler is nil")
		}
		return http.ErrServerClosed
	}
	if err := serveCmd.Flags().Set("addr", "127.0.0.1:9999"); err != nil {
		t.Fatalf("set addr flag: %v", err)
	}

	rec := httptest.NewRecorder()
	serveCmd.SetOut(rec)
	if err := serveCmd.RunE(serveCmd, nil); err != nil {
		t.Fatalf("serveCmd.RunE error = %v", err)
	}
	if gotAddr != "127.0.0.1:9999" {
		t.Fatalf("serve addr = %q, want 127.0.0.1:9999", gotAddr)
	}
	if !strings.Contains(rec.Body.String(), "Listening on http://127.0.0.1:9999") {
		t.Fatalf("serve output = %q", rec.Body.String())
	}
}

func TestServeCommandReturnsListenError(t *testing.T) {
	origListenAndServe := serveListenAndServe
	defer func() {
		serveListenAndServe = origListenAndServe
	}()

	wantErr := errors.New("bind failed")
	serveListenAndServe = func(string, http.Handler) error {
		return wantErr
	}
	if err := serveCmd.RunE(serveCmd, nil); !errors.Is(err, wantErr) {
		t.Fatalf("serveCmd.RunE error = %v, want %v", err, wantErr)
	}
}
