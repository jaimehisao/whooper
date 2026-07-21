package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func TestGetProfile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/user/profile/basic" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		profile := map[string]interface{}{
			"user_id": 123,
			"email":   "test@example.com",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	profile, err := client.GetProfile()
	if err != nil {
		t.Fatalf("GetProfile error = %v", err)
	}
	if profile.UserID != 123 {
		t.Errorf("UserID = %d, want 123", profile.UserID)
	}
}

func TestGetProfile_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetProfile()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsUnauthorized(err) {
		t.Fatalf("expected unauthorized StatusError, got %v", err)
	}
}

func TestGetProfile_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetProfile()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetCycles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"records": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetCycles("2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetCycles error = %v", err)
	}
}

func TestGetRecoveries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"records": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetRecoveries("", "")
	if err != nil {
		t.Fatalf("GetRecoveries error = %v", err)
	}
}

func TestGetSleeps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"records": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetSleeps("", "")
	if err != nil {
		t.Fatalf("GetSleeps error = %v", err)
	}
}

func TestGetWorkouts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"records": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetWorkouts("", "")
	if err != nil {
		t.Fatalf("GetWorkouts error = %v", err)
	}
}

func TestForEachEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		record := map[string]interface{}{"id": 1}
		if r.URL.Path == "/v2/activity/sleep" || r.URL.Path == "/v2/activity/workout" {
			record["id"] = "1"
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"records": []interface{}{record},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	t.Run("Cycles", func(t *testing.T) {
		err := client.ForEachCycle("", "", func(records []models.Cycle) error { return nil })
		if err != nil {
			t.Errorf("ForEachCycle error = %v", err)
		}
	})
	t.Run("Recoveries", func(t *testing.T) {
		err := client.ForEachRecovery("", "", func(records []models.Recovery) error { return nil })
		if err != nil {
			t.Errorf("ForEachRecovery error = %v", err)
		}
	})
	t.Run("Sleeps", func(t *testing.T) {
		err := client.ForEachSleep("", "", func(records []models.Sleep) error { return nil })
		if err != nil {
			t.Errorf("ForEachSleep error = %v", err)
		}
	})
	t.Run("Workouts", func(t *testing.T) {
		err := client.ForEachWorkout("", "", func(records []models.Workout) error { return nil })
		if err != nil {
			t.Errorf("ForEachWorkout error = %v", err)
		}
	})
}

func TestEndpointParams(t *testing.T) {
	tests := []struct {
		name string
		got  map[string]string
	}{
		{name: "recovery", got: recoveryParams("2024-01-01", "2024-01-31")},
		{name: "sleep", got: sleepParams("2024-01-01", "2024-01-31")},
		{name: "workout", got: workoutParams("2024-01-01", "2024-01-31")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got["limit"] != "25" {
				t.Fatalf("limit = %q, want 25", tt.got["limit"])
			}
			if tt.got["start"] != "2024-01-01" {
				t.Fatalf("start = %q, want 2024-01-01", tt.got["start"])
			}
			if tt.got["end"] != "2024-01-31" {
				t.Fatalf("end = %q, want 2024-01-31", tt.got["end"])
			}
		})
	}
}
