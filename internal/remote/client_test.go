package remote

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestGetJSONEmptyBaseURL(t *testing.T) {
	c := New("", "")
	err := c.GetJSON("/x", nil, nil)
	var re *Error
	if !asError(err, &re) || re.Kind != KindUnreachable {
		t.Fatalf("got %v, want unreachable empty URL", err)
	}
}

func TestGetJSONInvalidBaseURL(t *testing.T) {
	c := New("not-a-url", "")
	err := c.GetJSON("/x", nil, nil)
	var re *Error
	if !asError(err, &re) || re.Kind != KindUnreachable {
		t.Fatalf("got %v, want unreachable invalid URL", err)
	}
}

func TestGetJSONHTTPErrorAndDecode(t *testing.T) {
	t.Run("http_500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer srv.Close()
		c := New(srv.URL, "t")
		err := c.GetJSON("/x", nil, &map[string]any{})
		var re *Error
		if !asError(err, &re) || re.Kind != KindHTTP || re.StatusCode != 500 {
			t.Fatalf("got %#v", err)
		}
	})
	t.Run("bad_json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-json"))
		}))
		defer srv.Close()
		c := New(srv.URL, "")
		err := c.GetJSON("/x", nil, &map[string]any{})
		var re *Error
		if !asError(err, &re) || re.Kind != KindDecode {
			t.Fatalf("got %#v", err)
		}
	})
	t.Run("query_params", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("from") != "2024-01-01" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode([]any{})
		}))
		defer srv.Close()
		c := New(srv.URL, "")
		q := url.Values{"from": {"2024-01-01"}}
		var dest []any
		if err := c.GetJSON("/api/recovery", q, &dest); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("nil_dest_ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()
		c := New(srv.URL, "")
		if err := c.GetJSON("/x", nil, nil); err != nil {
			t.Fatal(err)
		}
	})
}

func TestErrorUnwrapAndFormat(t *testing.T) {
	inner := errors.New("dial")
	e := &Error{Kind: KindUnreachable, Message: "remote backend unreachable", Err: inner}
	if e.Error() == "" || e.Unwrap() != inner {
		t.Fatalf("unwrap/format failed: %v", e)
	}
	e2 := &Error{Kind: KindHTTP, Message: "no wrap"}
	if !strings.Contains(e2.Error(), "http_error") {
		t.Fatalf("error = %s", e2.Error())
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
