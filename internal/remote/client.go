// Package remote implements an HTTP client for a Whooper serve/service backend.
package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Failure classes for actionable CLI errors.
const (
	KindMissingToken = "missing_token"
	KindUnauthorized = "unauthorized"
	KindUnreachable  = "unreachable"
	KindHTTP         = "http_error"
	KindDecode       = "decode_error"
)

// Error is a classified remote-client failure.
type Error struct {
	Kind       string
	StatusCode int
	Message    string
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// Client talks to a Whooper HTTP API (serve/service).
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New returns a client for baseURL (trailing slash stripped).
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetJSON performs GET path (absolute path like /api/summary) with optional query
// and decodes a successful JSON body into dest.
func (c *Client) GetJSON(path string, query url.Values, dest any) error {
	if c.BaseURL == "" {
		return &Error{Kind: KindUnreachable, Message: "remote backend URL is empty"}
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return &Error{Kind: KindUnreachable, Message: fmt.Sprintf("invalid remote backend URL %q", c.BaseURL), Err: err}
	}
	ref, err := url.Parse(path)
	if err != nil {
		return &Error{Kind: KindHTTP, Message: fmt.Sprintf("invalid path %q", path), Err: err}
	}
	full := u.ResolveReference(ref)
	if query != nil {
		full.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, full.String(), nil)
	if err != nil {
		return &Error{Kind: KindHTTP, Message: "build request", Err: err}
	}
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return classifyTransport(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return &Error{Kind: KindHTTP, Message: "read response body", Err: err}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if dest == nil {
			return nil
		}
		if err := json.Unmarshal(body, dest); err != nil {
			return &Error{Kind: KindDecode, Message: "decode JSON response", Err: err}
		}
		return nil
	case http.StatusUnauthorized:
		if c.Token == "" {
			return &Error{
				Kind:       KindMissingToken,
				StatusCode: http.StatusUnauthorized,
				Message:    "remote backend requires a bearer token; set remote-token, WHOOPER_REMOTE_TOKEN, or WHOOPER_SERVE_TOKEN",
			}
		}
		return &Error{
			Kind:       KindUnauthorized,
			StatusCode: http.StatusUnauthorized,
			Message:    "remote backend rejected credentials (unauthorized)",
		}
	default:
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return &Error{
			Kind:       KindHTTP,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("remote backend returned HTTP %d: %s", resp.StatusCode, truncate(msg, 200)),
		}
	}
}

func classifyTransport(err error) error {
	if err == nil {
		return nil
	}
	var opErr *net.OpError
	var urlErr *url.Error
	switch {
	case errors.As(err, &urlErr):
		if urlErr.Timeout() || isConnRefused(urlErr.Err) || errors.As(urlErr.Err, &opErr) {
			return &Error{
				Kind:    KindUnreachable,
				Message: "remote backend unreachable or connection refused",
				Err:     err,
			}
		}
	case errors.As(err, &opErr):
		return &Error{
			Kind:    KindUnreachable,
			Message: "remote backend unreachable or connection refused",
			Err:     err,
		}
	case isConnRefused(err):
		return &Error{
			Kind:    KindUnreachable,
			Message: "remote backend unreachable or connection refused",
			Err:     err,
		}
	}
	return &Error{
		Kind:    KindUnreachable,
		Message: "remote backend request failed",
		Err:     err,
	}
}

func isConnRefused(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "connection refused")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
