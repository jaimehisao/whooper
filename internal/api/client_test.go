package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type okTokenSource struct{}

func (okTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "abc123", TokenType: "Bearer"}, nil
}

type errTokenSource struct{}

func (errTokenSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("token source failed")
}

func TestNewClientConfigAndAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer abc123" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer abc123")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	c := NewClient(okTokenSource{})
	c.R.SetBaseURL(server.URL)

	resp, err := c.R.R().Get("/")
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode())
	}
}

func TestNewClientRetryAfterParsing(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{name: "valid", header: "10", want: 10 * time.Second},
		{name: "invalid", header: "abc", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					w.Header().Set("Retry-After", tt.header)
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{}"))
			}))
			defer server.Close()

			c := NewClient(okTokenSource{})
			c.R.SetBaseURL(server.URL)
			c.R.SetRetryCount(1)

			start := time.Now()
			_, err := c.R.R().Get("/")
			if err != nil {
				t.Fatalf("request error = %v", err)
			}
			elapsed := time.Since(start)

			if calls != 2 {
				t.Fatalf("server calls = %d, want 2", calls)
			}

			if tt.want > 0 {
				if elapsed < tt.want-2*time.Second {
					t.Fatalf("elapsed %v shorter than expected wait %v", elapsed, tt.want)
				}
			} else {
				if elapsed > 3*time.Second {
					t.Fatalf("elapsed %v unexpectedly long for header %q", elapsed, tt.header)
				}
			}
		})
	}
}

func TestNewClientBeforeRequestTokenError(t *testing.T) {
	c := NewClient(errTokenSource{})
	c.R.SetBaseURL("http://127.0.0.1:" + strconv.Itoa(1))
	c.R.SetRetryCount(0)

	_, err := c.R.R().Get("/")
	if err == nil {
		t.Fatal("expected error from token source")
	}
	if err.Error() != "token source failed" {
		t.Fatalf("unexpected error = %v", err)
	}
}
