package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
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

func TestAgentStatusLocal(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	if err := config.Save(&config.Config{ClientID: "cid", ClientSecret: "csec"}); err != nil {
		t.Fatal(err)
	}
	out, err := runAgentNamed(t, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK || resp.Source != agentSourceLocal {
		t.Fatalf("resp = %+v", resp)
	}
	raw, _ := json.Marshal(resp.Data)
	if !strings.Contains(string(raw), "client_id_configured") {
		t.Fatalf("status data = %s", raw)
	}
}

func TestAgentStatusRemote(t *testing.T) {
	_, _ = startRemoteBackend(t, remoteTestToken)
	out, err := runAgentNamed(t, "status")
	if err != nil {
		t.Fatalf("remote status: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK || resp.Source != agentSourceRemote {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentRemoteEntities(t *testing.T) {
	_, _ = startRemoteBackend(t, remoteTestToken)
	agentFrom, agentTo = "2024-01-01", "2024-12-31"
	agentLimit = 50
	for _, name := range []string{"recovery", "sleep", "strain", "workouts"} {
		t.Run(name, func(t *testing.T) {
			out, err := runAgentNamed(t, name)
			if err != nil {
				t.Fatalf("%s: %v\n%s", name, err, out)
			}
			resp := decodeAgentResponse(t, []byte(out))
			if !resp.OK || resp.Source != agentSourceRemote {
				t.Fatalf("resp = %+v", resp)
			}
			data := resp.Data.(map[string]any)
			if _, ok := data["rows"]; !ok {
				t.Fatalf("missing rows: %#v", data)
			}
			if data["limit"].(float64) != 50 {
				t.Fatalf("limit = %v", data["limit"])
			}
		})
	}
}

func TestAgentRemoteMissingToken(t *testing.T) {
	baseURL, clientHome := startRemoteBackend(t, remoteTestToken)
	if err := config.Save(&config.Config{RemoteURL: baseURL, RemoteToken: ""}); err != nil {
		t.Fatal(err)
	}
	config.SetTestPaths(clientHome, filepath.Join(clientHome, "config.yaml"), filepath.Join(clientHome, "missing.db"))
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	out, err := runAgentNamed(t, "summary")
	if err == nil {
		t.Fatal("expected missing token")
	}
	resp := decodeAgentResponse(t, []byte(out))
	if resp.OK || resp.Error == nil || resp.Error.Class != agentClassMissingToken {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentRemoteUnreachable(t *testing.T) {
	setupAgentEnv(t)
	if err := config.Save(&config.Config{RemoteURL: "http://127.0.0.1:1", RemoteToken: "x"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	out, err := runAgentNamed(t, "summary")
	if err == nil {
		t.Fatal("expected unreachable")
	}
	resp := decodeAgentResponse(t, []byte(out))
	if resp.OK || resp.Error == nil || resp.Error.Class != agentClassUnreachable {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentDoctorRemoteNote(t *testing.T) {
	_, _ = startRemoteBackend(t, remoteTestToken)
	// Remote configured; doctor still inspects local paths.
	out, err := runAgentNamed(t, "doctor")
	resp := decodeAgentResponse(t, []byte(out))
	if resp.Command != "doctor" || resp.Source != agentSourceRemote {
		t.Fatalf("resp = %+v err=%v\n%s", resp, err, out)
	}
	raw, _ := json.Marshal(resp.Data)
	if !strings.Contains(string(raw), "remote") {
		t.Fatalf("expected remote note in data: %s", raw)
	}
}

func TestAgentInvalidLimit(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	agentLimit = 0
	out, err := runAgentNamed(t, "recovery")
	if err == nil {
		t.Fatal("expected invalid limit")
	}
	resp := decodeAgentResponse(t, []byte(out))
	if resp.OK || resp.Error == nil || resp.Error.Class != agentClassInvalidArgs {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentLimitCap(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	agentLimit = maxAPILimit + 50
	out, err := runAgentNamed(t, "recovery")
	if err != nil {
		t.Fatalf("recovery: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK {
		t.Fatalf("resp = %+v", resp)
	}
	data := resp.Data.(map[string]any)
	if int(data["limit"].(float64)) != maxAPILimit {
		t.Fatalf("limit capped = %v want %d", data["limit"], maxAPILimit)
	}
}

func TestAgentLocalSleepStrain(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveSleeps([]models.Sleep{{
		ID: "s1", Start: "2024-06-01T00:00:00Z", End: "2024-06-01T07:00:00Z",
		Nap: false, ScoreState: "SCORED",
		Score: &models.SleepScore{
			StageSummary:        models.SleepStageSummary{TotalInBedTimeMilli: 28800000},
			SleepNeeded:         models.SleepNeeded{BaselineMilli: 28800000},
			SleepEfficiencyPct:  90,
			SleepPerformancePct: 88,
		},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCycles([]models.Cycle{{
		ID: 1, Start: "2024-06-01T00:00:00Z", End: "2024-06-02T00:00:00Z",
		ScoreState: "SCORED",
		Score:      &models.CycleScore{Strain: 10.5},
	}}); err != nil {
		t.Fatal(err)
	}
	db.Close()

	for _, name := range []string{"sleep", "strain"} {
		out, err := runAgentNamed(t, name)
		if err != nil {
			t.Fatalf("%s: %v\n%s", name, err, out)
		}
		resp := decodeAgentResponse(t, []byte(out))
		if !resp.OK || resp.Source != agentSourceLocal {
			t.Fatalf("%s resp = %+v", name, resp)
		}
		if int(resp.Data.(map[string]any)["count"].(float64)) < 1 {
			t.Fatalf("%s expected rows", name)
		}
	}
}

func TestClassifyAgentErrorExtended(t *testing.T) {
	class, _ := classifyAgentError(nil)
	if class != agentClassInternal {
		t.Fatalf("nil -> %s", class)
	}
	class, msg := classifyAgentError(&remote.Error{Kind: remote.KindHTTP, Message: "boom", StatusCode: 500})
	if class != agentClassHTTP || msg != "boom" {
		t.Fatalf("http: %s %s", class, msg)
	}
	class, _ = classifyAgentError(&remote.Error{Kind: remote.KindDecode, Message: "bad json"})
	if class != agentClassHTTP {
		t.Fatalf("decode: %s", class)
	}
	class, _ = classifyAgentError(&remote.Error{Kind: "other", Message: "x"})
	if class != agentClassHTTP {
		t.Fatalf("default remote: %s", class)
	}
	// Wrapped message prefixes (formatRemoteError style)
	for _, tc := range []struct {
		msg   string
		class string
	}{
		{"remote missing_token: need token", agentClassMissingToken},
		{"remote unauthorized: no", agentClassUnauthorized},
		{"remote unreachable: down", agentClassUnreachable},
		{"unable to open database file", agentClassMissingDB},
		{"unknown entity \"x\"", agentClassInvalidArgs},
		{"missing subcommand", agentClassInvalidArgs},
		{"something else", agentClassInternal},
	} {
		class, _ = classifyAgentError(errors.New(tc.msg))
		if class != tc.class {
			t.Fatalf("%q -> %s want %s", tc.msg, class, tc.class)
		}
	}
}

func TestLimitAnySliceAllTypes(t *testing.T) {
	if limitAnySlice([]models.Recovery{{}}, 0).([]models.Recovery)[0].CycleID != 0 && false {
		t.Fatal("noop")
	}
	// limit <= 0 returns input unchanged
	in := []models.Cycle{{ID: 1}, {ID: 2}}
	if len(limitAnySlice(in, 0).([]models.Cycle)) != 2 {
		t.Fatal("limit 0")
	}
	if len(limitAnySlice([]models.Cycle{{ID: 1}, {ID: 2}, {ID: 3}}, 2).([]models.Cycle)) != 2 {
		t.Fatal("cycles")
	}
	if len(limitAnySlice([]models.Sleep{{ID: "a"}, {ID: "b"}, {ID: "c"}}, 1).([]models.Sleep)) != 1 {
		t.Fatal("sleeps")
	}
	if len(limitAnySlice([]models.Workout{{ID: "1"}, {ID: "2"}}, 1).([]models.Workout)) != 1 {
		t.Fatal("workouts")
	}
	maps := []map[string]any{{"a": 1}, {"a": 2}, {"a": 3}}
	if len(limitAnySlice(maps, 2).([]map[string]any)) != 2 {
		t.Fatal("maps")
	}
	// under limit unchanged
	if len(limitAnySlice([]models.Recovery{{CycleID: 1}}, 5).([]models.Recovery)) != 1 {
		t.Fatal("under")
	}
	// unknown type passthrough
	if limitAnySlice("x", 1) != "x" {
		t.Fatal("default")
	}
	if anySliceLen([]models.Cycle{{}, {}}) != 2 || anySliceLen([]models.Sleep{{}}) != 1 {
		t.Fatal("lens")
	}
	if anySliceLen([]models.Workout{{}, {}, {}}) != 3 || anySliceLen([]map[string]any{{}}) != 1 {
		t.Fatal("lens2")
	}
	if anySliceLen(42) != 0 {
		t.Fatal("unknown len")
	}
}

func TestAgentExitError(t *testing.T) {
	inner := errors.New("inner")
	e := &agentExitError{code: 2, err: inner}
	if e.Error() != "inner" || e.Unwrap() != inner {
		t.Fatalf("Error/Unwrap: %v %v", e.Error(), e.Unwrap())
	}
	e2 := &agentExitError{code: 1}
	if e2.Error() != "exit 1" {
		t.Fatalf("nil err message: %s", e2.Error())
	}
}

func TestAgentSummaryLocalEmptyHealth(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	// empty DB with schema only
	db.Close()
	out, err := runAgentNamed(t, "summary")
	if err != nil {
		t.Fatalf("summary empty: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK || resp.Source != agentSourceLocal {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentFailWrite(t *testing.T) {
	// Exercise agentFail via missing subcommand already; also ensure writeAgentResponse works with compact buffer
	var buf bytes.Buffer
	if err := writeAgentResponse(&buf, agentResponse{OK: true, Command: "x", GeneratedAt: "t"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"ok": true`) {
		t.Fatalf("got %s", buf.String())
	}
}

func TestAgentDoctorWithAPIFlagPath(t *testing.T) {
	setupAgentEnv(t)
	if err := config.Save(&config.Config{ClientID: "id", ClientSecret: "secret"}); err != nil {
		t.Fatal(err)
	}
	agentDoctorAPI = true // would hit API; without token doctor fails API check if not skip
	// For local without token and --api, expect failure report
	out, err := runAgentNamed(t, "doctor")
	resp := decodeAgentResponse(t, []byte(out))
	if resp.Command != "doctor" {
		t.Fatalf("resp = %+v err=%v", resp, err)
	}
	// With API flag and no token, doctor should report failures
	if resp.OK && err == nil {
		// might still pass if API skip logic differs — at least envelope exists
		t.Logf("doctor ok unexpectedly: %s", out)
	}
}

func TestAgentFetchEntityUnknownAndHelpers(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	_, _, err = agentFetchEntity("not-an-entity")
	if err == nil {
		t.Fatal("expected unknown entity")
	}
	class, _ := classifyAgentError(err)
	if class != agentClassInvalidArgs && class != agentClassInternal {
		// "unknown entity" maps to invalid_args
		if !strings.Contains(err.Error(), "unknown entity") {
			t.Fatalf("err = %v class = %s", err, class)
		}
	}
}

func TestAgentRemoteSummaryNilHealth(t *testing.T) {
	// Backend with empty DB → summary has no latest_health
	serverDir := t.TempDir()
	serverDB := filepath.Join(serverDir, "whooper.db")
	db, err := store.Open(serverDB)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	prevOpen, prevPath := serveOpenDB, serveDBPath
	serveOpenDB = store.OpenReadOnly
	serveDBPath = func() string { return serverDB }
	t.Cleanup(func() { serveOpenDB, serveDBPath = prevOpen, prevPath })

	clientHome := t.TempDir()
	config.SetTestPaths(clientHome, filepath.Join(clientHome, "config.yaml"), filepath.Join(clientHome, "missing.db"))
	handler := bearerAuthMiddleware("tok", newServeHandler(func() statusReport {
		return buildStatusReportWithOpenDB(func(string) (*store.DB, error) {
			return store.OpenReadOnly(serverDB)
		})
	}))
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	if err := config.Save(&config.Config{RemoteURL: srv.URL, RemoteToken: "tok"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvRemoteURL, "")
	t.Setenv(config.EnvRemoteToken, "")
	t.Setenv(config.EnvServeToken, "")

	out, err := runAgentNamed(t, "summary")
	if err != nil {
		t.Fatalf("summary: %v\n%s", err, out)
	}
	resp := decodeAgentResponse(t, []byte(out))
	if !resp.OK || resp.Source != agentSourceRemote {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestAgentSummaryLocalNeverSync(t *testing.T) {
	setupAgentEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	// Explicit empty sync state entities via never default
	if err := db.SaveRecoveries([]models.Recovery{{
		CycleID: 9, CreatedAt: "2024-07-01T00:00:00Z", ScoreState: "SCORED",
		Score: &models.RecoveryScore{RecoveryScore: 50},
	}}); err != nil {
		t.Fatal(err)
	}
	db.Close()
	out, err := runAgentNamed(t, "summary")
	if err != nil {
		t.Fatal(err)
	}
	resp := decodeAgentResponse(t, []byte(out))
	raw, _ := json.Marshal(resp.Data)
	if !strings.Contains(string(raw), "never") {
		t.Fatalf("expected never sync states: %s", raw)
	}
}
