package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

// testItem is a simple record type for testing FetchAll.
type testItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func newTestClient(baseURL string) *Client {
	r := resty.New().SetBaseURL(baseURL)
	return &Client{R: r}
}

func TestFetchAll_SinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := paginatedResponse[testItem]{
			Records: []testItem{
				{ID: 1, Name: "alpha"},
				{ID: 2, Name: "beta"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	items, err := FetchAll[testItem](client, "/items", nil)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != 1 || items[1].Name != "beta" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestFetchAll_MultiPage(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Query().Get("nextToken") == "" {
			// First page
			resp := paginatedResponse[testItem]{
				Records:   []testItem{{ID: 1, Name: "first"}},
				NextToken: "page2",
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			// Second page
			resp := paginatedResponse[testItem]{
				Records: []testItem{{ID: 2, Name: "second"}},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	items, err := FetchAll[testItem](client, "/items", nil)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
	if items[0].Name != "first" || items[1].Name != "second" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestFetchAll_EmptyRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := paginatedResponse[testItem]{
			Records: []testItem{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	items, err := FetchAll[testItem](client, "/items", nil)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestFetchAll_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := FetchAll[testItem](client, "/items", nil)
	if err == nil {
		t.Fatal("expected error for server error response, got nil")
	}
}
