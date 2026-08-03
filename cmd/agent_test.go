package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/remote"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"github.com/spf13/cobra"
)

func setupAgentEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	config.SetTestPaths(tmp, filepath.Join(tmp, "config.yaml"), filepath.Join(tmp, "whooper.db"))
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")
	// Reset agent flags
	agentFrom, agentTo = "", ""
	agentLimit = defaultAPILimit
	agentDoctorAPI = false
	return tmp
}

func decodeAgentResponse(t *testing.T, raw []byte) agentResponse {
	t.Helper()
	var resp agentResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode agent JSON: %v\n%s", err, raw)
	}
	return resp
}

func runAgentNamed(t *testing.T, name string) (string, error) {
	t.Helper()
	var target *cobra.Command
	for _, c := range agentCmd.Commands() {
		if c.Name() == name {
			target = c
			break
		}
	}
	if target == nil && name == "" {
		target = agentCmd
	}
	if target == nil {
		t.Fatalf("agent subcommand %q not found", name)
	}
	var buf bytes.Buffer
	target.SetOut(&buf)
	target.SetErr(&bytes.Buffer{})
	err := target.RunE(target, nil)
	return buf.String(), err
}

func TestAgentSchema(t *testing.T) {
	setupAgentEnv(t)
	out, err := runAgentNamed(t, "schema")
	if err != nil {
		t.Fatalf("schema: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK || resp.Command != "schema" {
		t.Fatalf("resp = %+v", resp)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok || data["read_only"] != true {
		t.Fatalf("expected read_only true: %#v", resp.Data)
	}
	if _, ok := data["commands"]; !ok {
		t.Fatal("missing commands catalog")
	}
}

func TestAgentSummaryLocal(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveRecoveries([]models.Recovery{{
		CycleID:    1,
		CreatedAt:  "2024-06-01T08:00:00Z",
		ScoreState: "SCORED",
		Score:      &models.RecoveryScore{RecoveryScore: 77, HRVRmssd: 60, RestingHeartRate: 50},
	}}); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, err := runAgentNamed(t, "summary")
	if err != nil {
		t.Fatalf("summary: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK || resp.Source != agentSourceLocal {
		t.Fatalf("resp = %+v", resp)
	}
	raw, _ := json.Marshal(resp.Data)
	if !strings.Contains(string(raw), "recovery_score") {
		t.Fatalf("missing recovery in data: %s", raw)
	}
}

func TestAgentSummaryMissingDB(t *testing.T) {
	setupAgentEnv(t)
	out, err := runAgentNamed(t, "summary")
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *agentExitError
	if !errors.As(err, &ae) || ae.code != 1 {
		t.Fatalf("exit err = %v", err)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if resp.OK || resp.Error == nil || resp.Error.Class != agentClassMissingDB {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentRecoveryLocalWithLimit(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	recs := []models.Recovery{
		{CycleID: 1, CreatedAt: "2024-06-01T08:00:00Z", ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 80}},
		{CycleID: 2, CreatedAt: "2024-06-02T08:00:00Z", ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 70}},
		{CycleID: 3, CreatedAt: "2024-06-03T08:00:00Z", ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 60}},
	}
	if err := db.SaveRecoveries(recs); err != nil {
		t.Fatal(err)
	}
	db.Close()

	agentFrom, agentTo = "2024-06-01", "2024-06-03"
	agentLimit = 2
	out, err := runAgentNamed(t, "recovery")
	if err != nil {
		t.Fatalf("recovery: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK {
		t.Fatalf("resp = %+v", resp)
	}
	data := resp.Data.(map[string]any)
	if int(data["count"].(float64)) != 2 {
		t.Fatalf("count = %v want 2 (limit)", data["count"])
	}
}

func TestAgentInvalidDate(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	agentFrom = "not-a-date"
	agentTo = ""
	agentLimit = 10
	out, err := runAgentNamed(t, "sleep")
	if err == nil {
		t.Fatal("expected invalid date error")
	}
	resp := decodeAgentResponse(t, []byte(out))
	if resp.OK || resp.Error == nil || resp.Error.Class != agentClassInvalidArgs {
		t.Fatalf("resp = %+v", resp)
	}
	var ae *agentExitError
	if !errors.As(err, &ae) || ae.code != 2 {
		t.Fatalf("want exit 2, got %v", err)
	}
}

func TestAgentMissingSubcommand(t *testing.T) {
	setupAgentEnv(t)
	var buf bytes.Buffer
	agentCmd.SetOut(&buf)
	err := agentCmd.RunE(agentCmd, nil)
	if err == nil {
		t.Fatal("expected missing subcommand")
	}
	resp := decodeAgentResponse(t, buf.Bytes())
	if resp.OK || resp.Error == nil || resp.Error.Class != agentClassInvalidArgs {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentRemoteSummary(t *testing.T) {
	_, _ = startRemoteBackend(t, remoteTestToken)
	out, err := runAgentNamed(t, "summary")
	if err != nil {
		t.Fatalf("remote summary: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK || resp.Source != agentSourceRemote {
		t.Fatalf("resp = %+v\n%s", resp, out)
	}
	raw, _ := json.Marshal(resp.Data)
	if !strings.Contains(string(raw), "recovery_score") {
		t.Fatalf("data = %s", raw)
	}
}

func TestAgentRemoteUnauthorized(t *testing.T) {
	baseURL, clientHome := startRemoteBackend(t, remoteTestToken)
	if err := config.Save(&config.Config{RemoteURL: baseURL, RemoteToken: "wrong"}); err != nil {
		t.Fatal(err)
	}
	config.SetTestPaths(clientHome, filepath.Join(clientHome, "config.yaml"), filepath.Join(clientHome, "missing.db"))
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	out, err := runAgentNamed(t, "status")
	if err == nil {
		t.Fatal("expected unauthorized")
	}
	resp := decodeAgentResponse(t, []byte(out))
	if resp.OK || resp.Error == nil || resp.Error.Class != agentClassUnauthorized {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentDoctorLocal(t *testing.T) {
	setupAgentEnv(t)
	if err := config.Save(&config.Config{ClientID: "id", ClientSecret: "secret"}); err != nil {
		t.Fatal(err)
	}
	out, err := runAgentNamed(t, "doctor")
	resp := decodeAgentResponse(t, []byte(out))
	if resp.Command != "doctor" {
		t.Fatalf("command = %s", resp.Command)
	}
	// With credentials and skip-API, open DB should succeed (creates empty schema).
	if err != nil {
		t.Logf("doctor err (may be ok if report has failures): %v\n%s", err, out)
	}
	if resp.Error != nil && resp.Error.Class == agentClassInvalidArgs {
		t.Fatalf("unexpected invalid_args: %s", out)
	}
}

func TestClassifyAgentError(t *testing.T) {
	cases := []struct {
		err   error
		class string
	}{
		{&remote.Error{Kind: remote.KindMissingToken, Message: "m"}, agentClassMissingToken},
		{&remote.Error{Kind: remote.KindUnauthorized, Message: "m"}, agentClassUnauthorized},
		{&remote.Error{Kind: remote.KindUnreachable, Message: "m"}, agentClassUnreachable},
		{fmt.Errorf("open database: boom"), agentClassMissingDB},
		{fmt.Errorf("invalid limit"), agentClassInvalidArgs},
	}
	for _, tc := range cases {
		class, _ := classifyAgentError(tc.err)
		if class != tc.class {
			t.Fatalf("err %v: class %s want %s", tc.err, class, tc.class)
		}
	}
}

func TestLimitAnySlice(t *testing.T) {
	rows := []models.Recovery{{CycleID: 1}, {CycleID: 2}, {CycleID: 3}}
	got := limitAnySlice(rows, 2).([]models.Recovery)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if anySliceLen(got) != 2 {
		t.Fatal("anySliceLen")
	}
}

func TestAgentRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("agent command not registered")
	}
}

func TestAgentWorkoutsEntityName(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	out, err := runAgentNamed(t, "workouts")
	// empty rows is success
	if err != nil {
		// missing data is ok; open succeeded
		resp := decodeAgentResponse(t, []byte(out))
		if resp.Error != nil && resp.Error.Class == agentClassMissingDB {
			t.Fatalf("unexpected missing_db after Open: %s", out)
		}
	} else {
		resp := decodeAgentResponse(t, []byte(out))
		if !resp.OK {
			t.Fatalf("resp = %+v", resp)
		}
		data := resp.Data.(map[string]any)
		if data["entity"] != "workouts" {
			t.Fatalf("entity = %v", data["entity"])
		}
	}
}
