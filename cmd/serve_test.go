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

func TestServeAPIEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)
	if err := config.Save(&config.Config{ClientID: "id", ClientSecret: "secret"}); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	if err := db.SaveRecoveries([]models.Recovery{{
		CycleID: 1, UserID: 1, CreatedAt: "2024-01-02T07:00:00Z", ScoreState: "SCORED",
		Score: &models.RecoveryScore{RecoveryScore: 81, HRVRmssd: 45, RestingHeartRate: 55},
	}}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}
	if err := db.SaveSleeps([]models.Sleep{{
		ID: "sleep-1", UserID: 1, Start: "2024-01-02T00:00:00Z", ScoreState: "SCORED",
		Score: &models.SleepScore{
			StageSummary: models.SleepStageSummary{
				TotalInBedTimeMilli: 8 * 3600 * 1000,
				TotalAwakeTimeMilli: 30 * 60 * 1000,
			},
			SleepNeeded:        models.SleepNeeded{BaselineMilli: 8 * 3600 * 1000},
			SleepEfficiencyPct: 94,
		},
	}}); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}
	if err := db.SaveCycles([]models.Cycle{{
		ID: 1, UserID: 1, Start: "2024-01-02T00:00:00Z", ScoreState: "SCORED",
		Score: &models.CycleScore{Strain: 12.4},
	}}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}
	if err := db.SaveWorkouts([]models.Workout{{
		ID: "workout-1", UserID: 1, Start: "2024-01-02T17:00:00Z", End: "2024-01-02T18:00:00Z", SportID: 0, ScoreState: "SCORED",
		Score: &models.WorkoutScore{Strain: 9.1, AverageHeartRate: 140, MaxHeartRate: 178, DistanceMeter: 5000},
	}}); err != nil {
		t.Fatalf("SaveWorkouts: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close DB: %v", err)
	}

	handler := newServeHandler(buildServeStatusReport)
	tests := map[string]string{
		"/api/summary":  `"latest_health"`,
		"/api/recovery": `"recovery_score": 81`,
		"/api/sleep":    `"actual_hours": 7.5`,
		"/api/strain":   `"strain": 12.4`,
		"/api/workouts": `"distance_km": 5`,
	}
	for path, want := range tests {
		req := httptest.NewRequest(http.MethodGet, path+"?limit=1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body:\n%s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("GET %s missing %q:\n%s", path, want, rec.Body.String())
		}
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
	if err := db.SaveRecoveries([]models.Recovery{{
		CycleID: 1, UserID: 1, CreatedAt: "2024-01-02T07:00:00Z", ScoreState: "SCORED",
		Score: &models.RecoveryScore{RecoveryScore: 81, HRVRmssd: 45, RestingHeartRate: 55},
	}}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}
	if err := db.SaveSleeps([]models.Sleep{{
		ID: "sleep-1", UserID: 1, Start: "2024-01-02T00:00:00Z", ScoreState: "SCORED",
		Score: &models.SleepScore{
			StageSummary: models.SleepStageSummary{
				TotalInBedTimeMilli: 8 * 3600 * 1000,
				TotalAwakeTimeMilli: 30 * 60 * 1000,
			},
			SleepNeeded: models.SleepNeeded{
				BaselineMilli: 8 * 3600 * 1000,
			},
			SleepEfficiencyPct:  94,
			SleepPerformancePct: 88,
			SleepConsistencyPct: 76,
		},
	}}); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}
	if err := db.SaveCycles([]models.Cycle{{
		ID: 2, UserID: 1, Start: "2024-01-02T00:00:00Z", ScoreState: "SCORED",
		Score: &models.CycleScore{Strain: 12.4},
	}}); err != nil {
		t.Fatalf("SaveCycles scored: %v", err)
	}
	if err := db.SaveWorkouts([]models.Workout{{
		ID: "workout-1", UserID: 1, Start: "2024-01-02T17:00:00Z", End: "2024-01-02T18:00:00Z", ScoreState: "SCORED",
		Score: &models.WorkoutScore{Strain: 9.1, AverageHeartRate: 140, MaxHeartRate: 178, DistanceMeter: 5000},
	}}); err != nil {
		t.Fatalf("SaveWorkouts: %v", err)
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
		`whooper_records_total{entity="cycles"} 2`,
		`whooper_last_sync_timestamp_seconds{entity="cycles"} 1.7041536e+09`,
		`whooper_records_total{entity="recoveries"} 1`,
		`whooper_latest_health_metric{metric="recovery_score"} 81`,
		`whooper_latest_health_metric{metric="hrv_rmssd"} 45`,
		`whooper_latest_health_metric{metric="sleep_actual_hours"} 7.5`,
		`whooper_latest_health_metric{metric="sleep_need_gap_hours"} -0.5`,
		`whooper_latest_health_metric{metric="day_strain"} 12.4`,
		`whooper_latest_health_metric{metric="workout_distance_km"} 5`,
		`whooper_latest_health_timestamp_seconds{entity="sleep"} 1.7041536e+09`,
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
