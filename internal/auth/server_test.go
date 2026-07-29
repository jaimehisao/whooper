package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestValidateRedirectURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		ok   bool
	}{
		{name: "localhost valid", raw: "http://localhost:8484/callback", ok: true},
		{name: "loopback valid", raw: "http://127.0.0.1:8484/callback", ok: true},
		{name: "https rejected", raw: "https://localhost:8484/callback", ok: false},
		{name: "wrong host", raw: "http://evil.example:8484/callback", ok: false},
		{name: "wrong port", raw: "http://localhost:9090/callback", ok: false},
		{name: "wrong path", raw: "http://localhost:8484/other", ok: false},
		{name: "invalid", raw: "://", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRedirectURL(tt.raw)
			if tt.ok && err != nil {
				t.Fatalf("validateRedirectURL(%q) error = %v", tt.raw, err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("validateRedirectURL(%q) expected error", tt.raw)
			}
		})
	}
}

func TestRandomState(t *testing.T) {
	a, err := randomState()
	if err != nil {
		t.Fatalf("randomState error = %v", err)
	}
	b, err := randomState()
	if err != nil {
		t.Fatalf("randomState error = %v", err)
	}
	if len(a) != 32 || len(b) != 32 {
		t.Fatalf("random state length should be 32 hex chars, got %d and %d", len(a), len(b))
	}
	if a == b {
		t.Fatalf("random states should differ")
	}
}

func TestOpenBrowserValidation(t *testing.T) {
	if err := openBrowser("://bad-url"); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestOpenBrowserUnsupportedPlatform(t *testing.T) {
	orig := runtimeGOOS
	runtimeGOOS = "plan9"
	t.Cleanup(func() { runtimeGOOS = orig })

	err := openBrowser("http://localhost:8484/callback")
	if err == nil {
		t.Fatal("expected unsupported platform error")
	}
}

func TestOpenBrowserLinuxMissingBinary(t *testing.T) {
	orig := runtimeGOOS
	runtimeGOOS = "linux"
	t.Cleanup(func() { runtimeGOOS = orig })

	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })

	err := openBrowser("http://localhost:8484/callback")
	if err == nil {
		t.Fatal("expected command start error when xdg-open is missing")
	}
}

func TestHandleOAuthCallbackStateMismatch(t *testing.T) {
	ch := make(chan oauthResult, 1)
	req := httptest.NewRequest(http.MethodGet, "/callback?state=wrong&code=abc", nil)
	rr := httptest.NewRecorder()

	handleOAuthCallback(rr, req, "expected", "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		t.Fatal("exchange should not be called")
		return nil, nil
	}, ch)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	select {
	case res := <-ch:
		t.Fatalf("unexpected channel result: %v", res)
	default:
	}
}

func TestHandleOAuthCallbackMissingCode(t *testing.T) {
	ch := make(chan oauthResult, 1)
	req := httptest.NewRequest(http.MethodGet, "/callback?state=expected", nil)
	rr := httptest.NewRecorder()

	handleOAuthCallback(rr, req, "expected", "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		t.Fatal("exchange should not be called")
		return nil, nil
	}, ch)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	select {
	case res := <-ch:
		t.Fatalf("unexpected channel result: %v", res)
	default:
	}
}

func TestHandleOAuthCallbackExchangeFailure(t *testing.T) {
	ch := make(chan oauthResult, 1)
	req := httptest.NewRequest(http.MethodGet, "/callback?state=expected&code=abc", nil)
	rr := httptest.NewRecorder()

	handleOAuthCallback(rr, req, "expected", "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		return nil, errors.New("exchange failed")
	}, ch)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if got := (<-ch).err; got == nil || !strings.Contains(got.Error(), "exchanging code") {
		t.Fatalf("unexpected channel error: %v", got)
	}
}

func TestHandleOAuthCallbackSuccess(t *testing.T) {
	ch := make(chan oauthResult, 1)
	req := httptest.NewRequest(http.MethodGet, "/callback?state=expected&code=abc", nil)
	rr := httptest.NewRecorder()
	want := &oauth2.Token{AccessToken: "token123"}

	handleOAuthCallback(rr, req, "expected", "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		return want, nil
	}, ch)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	res := <-ch
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if res.token == nil || res.token.AccessToken != want.AccessToken {
		t.Fatalf("unexpected token: %#v", res.token)
	}
}

func TestHandleOAuthCallbackSecondCallbackDoesNotBlock(t *testing.T) {
	ch := make(chan oauthResult, 1)
	firstReq := httptest.NewRequest(http.MethodGet, "/callback?state=expected&code=one", nil)
	firstRR := httptest.NewRecorder()

	handleOAuthCallback(firstRR, firstReq, "expected", "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "one"}, nil
	}, ch)

	secondReq := httptest.NewRequest(http.MethodGet, "/callback?state=expected&code=two", nil)
	secondRR := httptest.NewRecorder()
	handleOAuthCallback(secondRR, secondReq, "expected", "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "two"}, nil
	}, ch)

	res := <-ch
	if res.token == nil || res.token.AccessToken != "one" {
		t.Fatalf("expected first callback result, got %#v", res)
	}
}

func TestHandleOAuthCallbackRepeatedMismatchedState(t *testing.T) {
	ch := make(chan oauthResult, 1)
	state := "expected"

	// First callback: mismatched state
	req1 := httptest.NewRequest(http.MethodGet, "/callback?state=wrong&code=abc", nil)
	rr1 := httptest.NewRecorder()
	handleOAuthCallback(rr1, req1, state, "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		t.Fatal("exchange should not be called")
		return nil, nil
	}, ch)

	if rr1.Code != http.StatusBadRequest {
		t.Fatalf("first status = %d, want %d", rr1.Code, http.StatusBadRequest)
	}
	select {
	case res := <-ch:
		t.Fatalf("unexpected channel result after mismatch: %v", res)
	default:
	}

	// Second callback: mismatched state again
	req2 := httptest.NewRequest(http.MethodGet, "/callback?state=wrong2&code=abc", nil)
	rr2 := httptest.NewRecorder()
	handleOAuthCallback(rr2, req2, state, "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		t.Fatal("exchange should not be called")
		return nil, nil
	}, ch)

	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("second status = %d, want %d", rr2.Code, http.StatusBadRequest)
	}
}

func TestHandleOAuthCallbackWrongMethod(t *testing.T) {
	ch := make(chan oauthResult, 1)
	req := httptest.NewRequest(http.MethodPost, "/callback?state=expected&code=abc", nil)
	rr := httptest.NewRecorder()

	handleOAuthCallback(rr, req, "expected", "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		t.Fatal("exchange should not be called")
		return nil, nil
	}, ch)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	select {
	case res := <-ch:
		t.Fatalf("unexpected channel result: %v", res)
	default:
		// Success: nothing sent to channel for wrong method
	}
}

func TestHandleOAuthCallbackInvalidThenValid(t *testing.T) {
	ch := make(chan oauthResult, 1)
	state := "expected"

	// First callback: missing code (invalid)
	req1 := httptest.NewRequest(http.MethodGet, "/callback?state=expected", nil)
	rr1 := httptest.NewRecorder()
	handleOAuthCallback(rr1, req1, state, "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		t.Fatal("exchange should not be called")
		return nil, nil
	}, ch)

	if rr1.Code != http.StatusBadRequest {
		t.Fatalf("first status = %d, want %d", rr1.Code, http.StatusBadRequest)
	}
	select {
	case res := <-ch:
		t.Fatalf("unexpected channel result after missing code: %v", res)
	default:
	}

	// Second callback: valid
	req2 := httptest.NewRequest(http.MethodGet, "/callback?state=expected&code=abc", nil)
	rr2 := httptest.NewRecorder()
	want := &oauth2.Token{AccessToken: "token123"}
	handleOAuthCallback(rr2, req2, state, "verifier", func(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
		return want, nil
	}, ch)

	if rr2.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", rr2.Code, http.StatusOK)
	}
	res2 := <-ch
	if res2.err != nil {
		t.Fatalf("second result error = %v, want nil", res2.err)
	}
	if res2.token == nil || res2.token.AccessToken != want.AccessToken {
		t.Fatalf("second result token = %v, want %v", res2.token, want)
	}
}

func TestHandleOAuthCallbackMalformedQueries(t *testing.T) {
	state := "expected"
	verifier := "verifier"

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "missing both",
			query:      "",
			wantStatus: http.StatusBadRequest,
			wantErr:    "", // ignored — does not complete the flow
		},
		{
			name:       "duplicate code",
			query:      "state=expected&code=abc&code=def",
			wantStatus: http.StatusOK,
			wantErr:    "",
		},
		{
			name:       "unexpected params",
			query:      "state=expected&code=abc&foo=bar",
			wantStatus: http.StatusOK,
			wantErr:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan oauthResult, 1)
			req := httptest.NewRequest(http.MethodGet, "/callback?"+tt.query, nil)
			rr := httptest.NewRecorder()

			handleOAuthCallback(rr, req, state, verifier, func(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
				if tt.wantErr != "" {
					t.Fatal("exchange should not be called")
				}
				return &oauth2.Token{AccessToken: "abc"}, nil
			}, ch)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantErr != "" {
				res := <-ch
				if res.err == nil || !strings.Contains(res.err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %s", res.err, tt.wantErr)
				}
			} else if tt.wantStatus != http.StatusOK {
				select {
				case res := <-ch:
					t.Fatalf("unexpected channel result: %v", res)
				default:
				}
			}
		})
	}
}

func TestValidateRedirectURLExported(t *testing.T) {
	if err := ValidateRedirectURL("http://localhost:8484/callback"); err != nil {
		t.Fatalf("ValidateRedirectURL error = %v", err)
	}
}

func TestRunOAuthFlow_Timeout(t *testing.T) {
	origTimeout := oauthFlowTimeout
	oauthFlowTimeout = 10 * time.Millisecond
	defer func() { oauthFlowTimeout = origTimeout }()

	origOpenBrowser := openBrowserFunc
	openBrowserFunc = func(string) error { return nil }
	defer func() { openBrowserFunc = origOpenBrowser }()

	conf := &oauth2.Config{RedirectURL: "http://localhost:8484/callback"}
	_, err := RunOAuthFlow(conf)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("unexpected error: %v", err)
	}
}
