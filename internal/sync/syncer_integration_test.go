package sync

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"golang.org/x/oauth2"
)

type staticTokenSource struct{}

func (staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "test-token", TokenType: "Bearer"}, nil
}

func newSyncTestClient(baseURL string) *api.Client {
	c := api.NewClient(staticTokenSource{})
	c.R.SetBaseURL(baseURL)
	c.R.SetRetryCount(0)
	return c
}

func newSyncTestDB(t *testing.T) *store.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "syncer.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSyncFromFullSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/user/profile/basic":
			_, _ = w.Write([]byte(`{"user_id":123,"email":"a@b.com","first_name":"A","last_name":"B"}`))
		case "/v1/cycle":
			_, _ = w.Write([]byte(`{"records":[{"id":1,"user_id":123,"created_at":"2024-01-15T00:00:00Z","updated_at":"2024-01-15T00:00:00Z","start":"2024-01-15T00:00:00Z","end":"2024-01-16T00:00:00Z","days":1,"score_state":"SCORED","score":{"strain":10.1,"kilojoule":1000,"average_heart_rate":70,"max_heart_rate":150}}]}`))
		case "/v1/recovery":
			_, _ = w.Write([]byte(`{"records":[{"cycle_id":1,"sleep_id":11,"user_id":123,"created_at":"2024-01-15T06:00:00Z","updated_at":"2024-01-15T06:00:00Z","score_state":"SCORED","score":{"user_calibrating":false,"recovery_score":75,"resting_heart_rate":55,"hrv_rmssd_milli":45,"spo2_percentage":97,"skin_temp_celsius":33.5}}]}`))
		case "/v1/activity/sleep":
			_, _ = w.Write([]byte(`{"records":[{"id":11,"user_id":123,"created_at":"2024-01-14T22:00:00Z","updated_at":"2024-01-15T06:00:00Z","start":"2024-01-14T22:00:00Z","end":"2024-01-15T06:00:00Z","nap":false,"score_state":"SCORED","score":{"stage_summary":{"total_in_bed_time_milli":28800000,"total_awake_time_milli":3600000,"total_no_data_time_milli":0,"total_light_sleep_time_milli":10000000,"total_slow_wave_sleep_time_milli":7000000,"total_rem_sleep_time_milli":7000000,"sleep_cycle_count":4,"disturbance_count":1},"sleep_needed":{"baseline_sleep_needed_milli":28800000,"need_from_sleep_debt_milli":0,"need_from_recent_strain_milli":0,"need_from_recent_nap_milli":0},"respiratory_rate":15.2,"sleep_performance_percentage":88,"sleep_consistency_percentage":85,"sleep_efficiency_percentage":87}}]}`))
		case "/v1/activity/workout":
			_, _ = w.Write([]byte(`{"records":[{"id":99,"user_id":123,"created_at":"2024-01-15T07:00:00Z","updated_at":"2024-01-15T08:00:00Z","start":"2024-01-15T07:00:00Z","end":"2024-01-15T08:00:00Z","sport_id":0,"score_state":"SCORED","score":{"strain":11.1,"average_heart_rate":140,"max_heart_rate":170,"kilojoule":800,"percent_recorded":100,"distance_meter":5000,"altitude_gain_meter":10,"altitude_change_meter":5,"zone_duration":{"zone_zero_milli":1,"zone_one_milli":2,"zone_two_milli":3,"zone_three_milli":4,"zone_four_milli":5,"zone_five_milli":6}}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	db := newSyncTestDB(t)
	s := New(newSyncTestClient(server.URL), db, nil)

	if err := s.SyncFrom("full"); err != nil {
		t.Fatalf("SyncFrom(full) error = %v", err)
	}

	if _, err := db.GetProfile(); err != nil {
		t.Fatalf("GetProfile error = %v", err)
	}

	if cycles, err := db.ListCycles("", ""); err != nil || len(cycles) != 1 {
		t.Fatalf("ListCycles len = %d, err = %v, want 1", len(cycles), err)
	}
	if recs, err := db.ListRecoveries("", ""); err != nil || len(recs) != 1 {
		t.Fatalf("ListRecoveries len = %d, err = %v, want 1", len(recs), err)
	}
	if sleeps, err := db.ListSleeps("", "", false); err != nil || len(sleeps) != 1 {
		t.Fatalf("ListSleeps len = %d, err = %v, want 1", len(sleeps), err)
	}
	if workouts, err := db.ListWorkouts("", ""); err != nil || len(workouts) != 1 {
		t.Fatalf("ListWorkouts len = %d, err = %v, want 1", len(workouts), err)
	}

	for _, entity := range []string{"cycles", "recoveries", "sleeps", "workouts"} {
		got, err := db.GetSyncState(entity)
		if err != nil {
			t.Fatalf("GetSyncState(%s) error = %v", entity, err)
		}
		if got == "" {
			t.Fatalf("GetSyncState(%s) should not be empty", entity)
		}
	}
}

func TestSyncFromUsesOverlapStartForIncrementalSync(t *testing.T) {
	var mu sync.Mutex
	starts := map[string]string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if start := r.URL.Query().Get("start"); start != "" {
			starts[r.URL.Path] = start
		}
		mu.Unlock()

		switch r.URL.Path {
		case "/v1/user/profile/basic":
			_, _ = w.Write([]byte(`{"user_id":1,"email":"x@y.com","first_name":"X","last_name":"Y"}`))
		case "/v1/cycle", "/v1/recovery", "/v1/activity/sleep", "/v1/activity/workout":
			_, _ = w.Write([]byte(`{"records":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	db := newSyncTestDB(t)
	if err := db.SetSyncState("cycles", "2024-01-15T00:00:00Z"); err != nil {
		t.Fatalf("SetSyncState error = %v", err)
	}

	s := New(newSyncTestClient(server.URL), db, nil)
	if err := s.SyncFrom(""); err != nil {
		t.Fatalf("SyncFrom(empty) error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	expected := "2024-01-14T00:00:00Z"
	if starts["/v1/cycle"] != expected {
		t.Fatalf("cycle start query = %q, want %q", starts["/v1/cycle"], expected)
	}
}

func TestSyncFromAggregatesEntityErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/user/profile/basic":
			_, _ = w.Write([]byte(`{"user_id":1,"email":"x@y.com","first_name":"X","last_name":"Y"}`))
		case "/v1/recovery":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		case "/v1/cycle", "/v1/activity/sleep", "/v1/activity/workout":
			_, _ = w.Write([]byte(`{"records":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	db := newSyncTestDB(t)
	s := New(newSyncTestClient(server.URL), db, nil)

	err := s.SyncFrom("full")
	if err == nil {
		t.Fatal("expected sync error, got nil")
	}
	if !strings.Contains(err.Error(), "recoveries") {
		t.Fatalf("error %q should mention recoveries", err)
	}

	got, stateErr := db.GetSyncState("cycles")
	if stateErr != nil {
		t.Fatalf("GetSyncState error = %v", stateErr)
	}
	if got != "" {
		t.Fatalf("sync state should remain empty on aggregate failure, got %q", got)
	}
}

func TestSyncAllCallsSyncFrom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/user/profile/basic":
			_, _ = w.Write([]byte(`{"user_id":1,"email":"x@y.com","first_name":"X","last_name":"Y"}`))
		case "/v1/cycle", "/v1/recovery", "/v1/activity/sleep", "/v1/activity/workout":
			_, _ = w.Write([]byte(`{"records":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	db := newSyncTestDB(t)
	s := New(newSyncTestClient(server.URL), db, nil)

	if err := s.SyncAll(); err != nil {
		t.Fatalf("SyncAll error = %v", err)
	}
}
