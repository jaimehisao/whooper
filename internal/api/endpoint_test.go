package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProfile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/user/profile/basic" {
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
