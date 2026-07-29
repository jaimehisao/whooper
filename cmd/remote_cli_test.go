package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

const remoteTestToken = "test-remote-token-xyz"

// seedRemoteFixtureDB creates a server-side DB with known health data.
func seedRemoteFixtureDB(t *testing.T, dbPath string) {
	t.Helper()
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	defer db.Close()

	if err := db.SaveRecoveries([]models.Recovery{{
		CycleID:    99,
		CreatedAt:  "2024-06-01T08:00:00Z",
		ScoreState: "SCORED",
		Score: &models.RecoveryScore{
			RecoveryScore:    88,
			HRVRmssd:         72,
			RestingHeartRate: 48,
		},
	}}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}
	if err := db.SaveCycles([]models.Cycle{{
		ID:         99,
		Start:      "2024-06-01T00:00:00Z",
		End:        "2024-06-02T00:00:00Z",
		ScoreState: "SCORED",
		Score: &models.CycleScore{
			Strain:           14.2,
			Kilojoule:        1000,
			AverageHeartRate: 70,
			MaxHeartRate:     120,
		},
	}}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}
	if err := db.SaveSleeps([]models.Sleep{{
		ID:         "s-remote-1",
		Start:      "2024-05-31T22:00:00Z",
		End:        "2024-06-01T06:00:00Z",
		Nap:        false,
		ScoreState: "SCORED",
		Score: &models.SleepScore{
			StageSummary: models.SleepStageSummary{
				TotalInBedTimeMilli: 28800000,
				TotalAwakeTimeMilli: 1800000,
			},
			SleepNeeded: models.SleepNeeded{
				BaselineMilli: 28800000,
			},
			SleepEfficiencyPct:  93,
			SleepPerformancePct: 90,
		},
	}}); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}
	if err := db.SaveWorkouts([]models.Workout{{
		ID:         "w-remote-1",
		Start:      "2024-06-01T12:00:00Z",
		End:        "2024-06-01T13:00:00Z",
		SportID:    1,
		ScoreState: "SCORED",
		Score: &models.WorkoutScore{
			Strain:           11.5,
			AverageHeartRate: 140,
			MaxHeartRate:     170,
			DistanceMeter:    models.FloatPtr(20000),
		},
	}}); err != nil {
		t.Fatalf("SaveWorkouts: %v", err)
	}
	if err := db.SetSyncState("recoveries", "2024-06-01T09:00:00Z"); err != nil {
		t.Fatalf("SetSyncState: %v", err)
	}
}

// startRemoteBackend boots the real serve handler against a seeded DB with bearer auth.
func startRemoteBackend(t *testing.T, token string) (baseURL string, clientHome string) {
	t.Helper()

	serverDir := t.TempDir()
	serverDB := filepath.Join(serverDir, "whooper.db")
	seedRemoteFixtureDB(t, serverDB)

	// Point serve seams at the server fixture for the lifetime of this test.
	prevOpen, prevPath := serveOpenDB, serveDBPath
	serveOpenDB = store.OpenReadOnly
	serveDBPath = func() string { return serverDB }
	t.Cleanup(func() {
		serveOpenDB = prevOpen
		serveDBPath = prevPath
	})

	// Client home has config pointing remote, and deliberately NO usable local DB.
	clientHome = t.TempDir()
	clientCfg := filepath.Join(clientHome, "config.yaml")
	clientDB := filepath.Join(clientHome, "missing.db") // does not exist
	prevDir, prevCfg, prevDB := config.Dir(), config.Path(), config.DBPath()
	config.SetTestPaths(clientHome, clientCfg, clientDB)
	t.Cleanup(func() {
		config.SetTestPaths(prevDir, prevCfg, prevDB)
	})

	// Status reporter must open the server DB even while client paths are active.
	handler := bearerAuthMiddleware(token, newServeHandler(func() statusReport {
		return buildStatusReportWithOpenDB(func(string) (*store.DB, error) {
			return store.OpenReadOnly(serverDB)
		})
	}))
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// Persist remote config for the client (no local Whoop credentials required).
	if err := config.Save(&config.Config{
		RemoteURL:   srv.URL,
		RemoteToken: token,
	}); err != nil {
		t.Fatalf("Save remote config: %v", err)
	}

	// Ensure env does not override unless a test sets it.
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	return srv.URL, clientHome
}

func TestRemoteSummarySuccessJSON(t *testing.T) {
	_, _ = startRemoteBackend(t, remoteTestToken)

	// Confirm local DB is absent so local mode would fail.
	if _, err := os.Stat(config.DBPath()); !os.IsNotExist(err) {
		t.Fatalf("expected missing local db at %s, err=%v", config.DBPath(), err)
	}

	var buf bytes.Buffer
	summaryJSON = true
	summaryCmd.SetOut(&buf)
	summaryCmd.SetErr(&buf)
	t.Cleanup(func() {
		summaryJSON = false
		summaryCmd.SetOut(nil)
		summaryCmd.SetErr(nil)
	})

	if err := summaryCmd.RunE(summaryCmd, nil); err != nil {
		t.Fatalf("summary remote: %v\n%s", err, buf.String())
	}
	out := buf.String()
	t.Logf("remote summary --json output:\n%s", out)
	if !strings.Contains(out, "88") {
		t.Fatalf("expected recovery_score 88 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "recovery_score") {
		t.Fatalf("expected recovery_score key, got:\n%s", out)
	}
	// Ensure we actually decoded structured JSON with fixture HRV.
	var parsed struct {
		LatestHealth *healthReport `json:"latest_health"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("parse summary json: %v\n%s", err, out)
	}
	if parsed.LatestHealth == nil || parsed.LatestHealth.Values["recovery_score"] != 88 {
		t.Fatalf("latest_health = %#v", parsed.LatestHealth)
	}
	if parsed.LatestHealth.Values["hrv_rmssd"] != 72 {
		t.Fatalf("hrv = %v", parsed.LatestHealth.Values["hrv_rmssd"])
	}
}

func TestRemoteExportRecoveriesContainsFixture(t *testing.T) {
	_, _ = startRemoteBackend(t, remoteTestToken)

	prevEntity, prevFormat := exportEntity, exportFormat
	exportEntity = "recoveries"
	exportFormat = "json"
	exportFrom, exportTo, exportOutput = "", "", ""
	t.Cleanup(func() {
		exportEntity, exportFormat = prevEntity, prevFormat
		exportFrom, exportTo, exportOutput = "", "", ""
	})

	var buf bytes.Buffer
	exportCmd.SetOut(&buf)
	exportCmd.SetErr(&buf)
	t.Cleanup(func() {
		exportCmd.SetOut(nil)
		exportCmd.SetErr(nil)
	})

	if err := exportCmd.RunE(exportCmd, nil); err != nil {
		t.Fatalf("export remote: %v\n%s", err, buf.String())
	}
	out := buf.String()
	t.Logf("remote export recoveries output:\n%s", out)
	if !strings.Contains(out, "88") && !strings.Contains(out, "88.0") {
		// recovery_score from view
		t.Fatalf("expected recovery score in export, got:\n%s", out)
	}
	if !strings.Contains(out, "recovery_score") {
		t.Fatalf("expected recovery_score column, got:\n%s", out)
	}
}

func TestRemoteStatusSuccess(t *testing.T) {
	baseURL, _ := startRemoteBackend(t, remoteTestToken)

	var buf bytes.Buffer
	statusJSON = false
	statusCmd.SetOut(&buf)
	t.Cleanup(func() {
		statusCmd.SetOut(nil)
	})

	if err := statusCmd.RunE(statusCmd, nil); err != nil {
		t.Fatalf("status remote: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Remote backend:") {
		t.Fatalf("expected remote backend line, got:\n%s", out)
	}
	if !strings.Contains(out, baseURL) {
		t.Fatalf("expected %s in output:\n%s", baseURL, out)
	}
	if !strings.Contains(out, "Database open: true") {
		t.Fatalf("expected remote db open, got:\n%s", out)
	}
}

func TestRemoteAuthRejectedWrongToken(t *testing.T) {
	baseURL, clientHome := startRemoteBackend(t, remoteTestToken)
	// Overwrite client token with wrong value.
	if err := config.Save(&config.Config{
		RemoteURL:   baseURL,
		RemoteToken: "wrong-token",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	config.SetTestPaths(clientHome, filepath.Join(clientHome, "config.yaml"), filepath.Join(clientHome, "missing.db"))

	var buf bytes.Buffer
	summaryJSON = true
	summaryCmd.SetOut(&buf)
	summaryCmd.SetErr(&buf)
	t.Cleanup(func() {
		summaryJSON = false
		summaryCmd.SetOut(nil)
		summaryCmd.SetErr(nil)
	})

	err := summaryCmd.RunE(summaryCmd, nil)
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	msg := err.Error()
	t.Logf("wrong token error: %s", msg)
	if !strings.Contains(msg, "unauthorized") {
		t.Fatalf("expected unauthorized in error, got: %v", err)
	}
}

func TestRemoteAuthMissingToken(t *testing.T) {
	baseURL, clientHome := startRemoteBackend(t, remoteTestToken)
	if err := config.Save(&config.Config{
		RemoteURL:   baseURL,
		RemoteToken: "",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	config.SetTestPaths(clientHome, filepath.Join(clientHome, "config.yaml"), filepath.Join(clientHome, "missing.db"))
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	err := summaryCmd.RunE(summaryCmd, nil)
	if err == nil {
		t.Fatal("expected missing_token error")
	}
	msg := err.Error()
	t.Logf("missing token error: %s", msg)
	if !strings.Contains(msg, "missing_token") && !strings.Contains(msg, "token") {
		t.Fatalf("expected missing token messaging, got: %v", err)
	}
}

func TestRemoteUnreachable(t *testing.T) {
	clientHome := t.TempDir()
	config.SetTestPaths(clientHome, filepath.Join(clientHome, "config.yaml"), filepath.Join(clientHome, "missing.db"))
	if err := config.Save(&config.Config{
		RemoteURL:   "http://127.0.0.1:1",
		RemoteToken: "x",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	err := summaryCmd.RunE(summaryCmd, nil)
	if err == nil {
		t.Fatal("expected unreachable error")
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("expected unreachable, got: %v", err)
	}
}

func TestLocalModeWhenRemoteUnset(t *testing.T) {
	// Local empty DB still works (existing empty summary path).
	tmp := t.TempDir()
	config.SetTestPaths(tmp, filepath.Join(tmp, "config.yaml"), filepath.Join(tmp, "whooper.db"))
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.Close()

	var buf bytes.Buffer
	summaryJSON = false
	summaryCmd.SetOut(&buf)
	t.Cleanup(func() {
		summaryCmd.SetOut(nil)
	})
	if err := summaryCmd.RunE(summaryCmd, nil); err != nil {
		t.Fatalf("local summary: %v", err)
	}
	if !strings.Contains(buf.String(), "No local health data found.") {
		t.Fatalf("expected local empty message, got:\n%s", buf.String())
	}
}

func TestLocalModeMissingDBStillErrors(t *testing.T) {
	tmp := t.TempDir()
	config.SetTestPaths(tmp, filepath.Join(tmp, "config.yaml"), filepath.Join(tmp, "nope.db"))
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")
	_ = config.Save(&config.Config{})

	err := summaryCmd.RunE(summaryCmd, nil)
	if err == nil {
		t.Fatal("expected open database error in local mode")
	}
	t.Logf("local missing DB error: %s", err.Error())
	if !strings.Contains(err.Error(), "open database") {
		t.Fatalf("expected open database error, got: %v", err)
	}
}

func TestConfigSetAndShowRemoteNoSecretLeak(t *testing.T) {
	tmp := t.TempDir()
	config.SetTestPaths(tmp, filepath.Join(tmp, "config.yaml"), filepath.Join(tmp, "whooper.db"))
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	if err := configSetCmd.RunE(configSetCmd, []string{"remote-url", "http://backend:9464"}); err != nil {
		t.Fatalf("set remote-url: %v", err)
	}
	if err := configSetCmd.RunE(configSetCmd, []string{"remote-token", "super-secret-token-value"}); err != nil {
		t.Fatalf("set remote-token: %v", err)
	}

	var buf bytes.Buffer
	configCmd.SetOut(&buf)
	t.Cleanup(func() { configCmd.SetOut(nil) })
	if err := configCmd.RunE(configCmd, nil); err != nil {
		t.Fatalf("config show: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "http://backend:9464") {
		t.Fatalf("expected remote url, got:\n%s", out)
	}
	if !strings.Contains(out, "remote_mode:   enabled") {
		t.Fatalf("expected remote mode enabled, got:\n%s", out)
	}
	if strings.Contains(out, "super-secret-token-value") {
		t.Fatalf("token leaked in config show:\n%s", out)
	}
	if !strings.Contains(out, "****") {
		t.Fatalf("expected masked token, got:\n%s", out)
	}
}

func TestRemoteEnvOverridesFileConfig(t *testing.T) {
	baseURL, clientHome := startRemoteBackend(t, remoteTestToken)
	// File points somewhere else; env should win.
	if err := config.Save(&config.Config{
		RemoteURL:   "http://127.0.0.1:1",
		RemoteToken: "wrong",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	config.SetTestPaths(clientHome, filepath.Join(clientHome, "config.yaml"), filepath.Join(clientHome, "missing.db"))
	t.Setenv(config.EnvRemoteURL, baseURL)
	t.Setenv(config.EnvRemoteToken, remoteTestToken)

	var buf bytes.Buffer
	summaryJSON = true
	summaryCmd.SetOut(&buf)
	t.Cleanup(func() {
		summaryJSON = false
		summaryCmd.SetOut(nil)
	})
	if err := summaryCmd.RunE(summaryCmd, nil); err != nil {
		t.Fatalf("env override summary: %v", err)
	}
	var parsed struct {
		LatestHealth *healthReport `json:"latest_health"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("parse: %v\n%s", err, buf.String())
	}
	if parsed.LatestHealth == nil || parsed.LatestHealth.Values["recovery_score"] != 88 {
		t.Fatalf("expected recovery 88, got %#v\n%s", parsed.LatestHealth, buf.String())
	}
}

// Ensure bearer middleware is wired the same way production serve uses it.
func TestRemoteUsesRealServeHandlerContract(t *testing.T) {
	_, _ = startRemoteBackend(t, remoteTestToken)
	backend, ok, err := resolveRemoteBackend()
	if err != nil || !ok {
		t.Fatalf("resolve: ok=%v err=%v", ok, err)
	}
	req, _ := http.NewRequest(http.MethodGet, backend.URL+"/healthz", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status %d", resp.StatusCode)
	}
}
