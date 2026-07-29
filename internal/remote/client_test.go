package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetJSONSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/summary" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("auth = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "n": 42})
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	var dest map[string]any
	if err := c.GetJSON("/api/summary", nil, &dest); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if dest["ok"] != true {
		t.Fatalf("dest = %#v", dest)
	}
}

func TestGetJSONUnauthorizedMissingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	err := c.GetJSON("/status", nil, &map[string]any{})
	if err == nil {
		t.Fatal("expected error")
	}
	var re *Error
	if !asError(err, &re) || re.Kind != KindMissingToken {
		t.Fatalf("got %v, want missing_token", err)
	}
	if !strings.Contains(err.Error(), "missing_token") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetJSONUnauthorizedBadToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "wrong")
	err := c.GetJSON("/status", nil, &map[string]any{})
	if err == nil {
		t.Fatal("expected error")
	}
	var re *Error
	if !asError(err, &re) || re.Kind != KindUnauthorized {
		t.Fatalf("got %v, want unauthorized", err)
	}
}

func TestGetJSONUnreachable(t *testing.T) {
	c := New("http://127.0.0.1:1", "")
	c.HTTPClient = &http.Client{}
	err := c.GetJSON("/healthz", nil, nil)
	if err == nil {
		t.Fatal("expected unreachable error")
	}
	var re *Error
	if !asError(err, &re) || re.Kind != KindUnreachable {
		t.Fatalf("got %v, want unreachable", err)
	}
}

func asError(err error, target **Error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	*target = e
	return true
}
