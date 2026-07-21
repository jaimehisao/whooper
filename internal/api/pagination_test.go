package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestFetchPaginated(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Query().Get("nextToken") == "" {
			resp := paginatedResponse[testItem]{
				Records:   []testItem{{ID: 1, Name: "p1"}},
				NextToken: "p2",
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			resp := paginatedResponse[testItem]{
				Records: []testItem{{ID: 2, Name: "p2"}},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	var captured []testItem
	err := FetchPaginated[testItem](client, "/items", nil, func(records []testItem) error {
		captured = append(captured, records...)
		return nil
	})

	if err != nil {
		t.Fatalf("FetchPaginated: %v", err)
	}
	if len(captured) != 2 {
		t.Errorf("expected 2 items, got %d", len(captured))
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
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
	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error type = %T, want *StatusError", err)
	}
	if statusErr.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", statusErr.Status)
	}
}

func TestIsUnauthorized(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &StatusError{Endpoint: "/items", Status: http.StatusUnauthorized})
	if !IsUnauthorized(err) {
		t.Fatal("expected wrapped 401 status error to be unauthorized")
	}
	if IsUnauthorized(&StatusError{Endpoint: "/items", Status: http.StatusForbidden}) {
		t.Fatal("403 should not be unauthorized")
	}
}

func TestFetchPaginated_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := FetchPaginated[testItem](client, "/items", nil, func(records []testItem) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestFetchPaginated_CallbackError(t *testing.T) {
	callbackErr := errors.New("stop")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := paginatedResponse[testItem]{Records: []testItem{{ID: 1}}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := FetchPaginated[testItem](client, "/items", nil, func(records []testItem) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("FetchPaginated error = %v, want callback error", err)
	}
}

func TestFetchPaginated_RepeatedNextTokenErrors(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := paginatedResponse[testItem]{
			Records:   []testItem{{ID: callCount}},
			NextToken: "same-token",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := FetchPaginated[testItem](client, "/items", nil, func(records []testItem) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "repeated next_token") {
		t.Fatalf("FetchPaginated error = %v, want repeated next_token", err)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2", callCount)
	}
}

func TestFetchPaginated_Property(t *testing.T) {
	tests := []struct {
		name       string
		tokens     []string // tokens to return in sequence
		wantCalls  int
		wantItems  int
		wantErr    bool
		wantParams map[string]string
	}{
		{
			name:      "Empty token stops",
			tokens:    []string{""},
			wantCalls: 1,
			wantItems: 1,
		},
		{
			name:      "Repeated token errors",
			tokens:    []string{"a", "a"},
			wantCalls: 2,
			wantItems: 2,
			wantErr:   true,
		},
		{
			name:      "Three page chain",
			tokens:    []string{"p2", "p3", ""},
			wantCalls: 3,
			wantItems: 3,
		},
		{
			name:      "Preserves params",
			tokens:    []string{""},
			wantCalls: 1,
			wantItems: 1,
			wantParams: map[string]string{"foo": "bar", "limit": "20"},
		},
		{
			name:      "Unusual token strings",
			tokens:    []string{"!@#$%^&*()", "   ", "{\"json\":true}", ""},
			wantCalls: 4,
			wantItems: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify query params are preserved
				for k, v := range tt.wantParams {
					if r.URL.Query().Get(k) != v {
						t.Errorf("call %d: missing param %s=%s", callCount, k, v)
					}
				}

				// Verify nextToken is NOT sent on first page
				if callCount == 0 && r.URL.Query().Get("nextToken") != "" {
					t.Errorf("call 0: should not have nextToken, got %q", r.URL.Query().Get("nextToken"))
				}
				// Verify nextToken IS sent on subsequent pages
				if callCount > 0 && r.URL.Query().Get("nextToken") != tt.tokens[callCount-1] {
					t.Errorf("call %d: want nextToken %q, got %q", callCount, tt.tokens[callCount-1], r.URL.Query().Get("nextToken"))
				}

				token := ""
				if callCount < len(tt.tokens) {
					token = tt.tokens[callCount]
				}

				resp := paginatedResponse[testItem]{
					Records:   []testItem{{ID: callCount}},
					NextToken: token,
				}
				callCount++
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := newTestClient(server.URL)
			var captured []testItem
			err := FetchPaginated[testItem](client, "/items", tt.wantParams, func(records []testItem) error {
				captured = append(captured, records...)
				return nil
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected pagination error, got nil")
				}
			} else if err != nil {
				t.Fatalf("FetchPaginated error = %v", err)
			}
			if callCount != tt.wantCalls {
				t.Errorf("callCount = %d, want %d", callCount, tt.wantCalls)
			}
			if len(captured) != tt.wantItems {
				t.Errorf("len(captured) = %d, want %d", len(captured), tt.wantItems)
			}
		})
	}
}
