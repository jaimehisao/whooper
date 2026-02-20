package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

const oauthFlowTimeout = 5 * time.Minute

// RunOAuthFlow performs the full OAuth2 authorization-code flow with PKCE:
//  1. Generates a random state parameter and PKCE verifier.
//  2. Starts a local HTTP server on :8484 to receive the callback.
//  3. Opens the user's browser to the authorization URL.
//  4. Waits for the callback, exchanges the code for a token.
//  5. Shuts down the server and returns the token.
func RunOAuthFlow(oauthCfg *oauth2.Config) (*oauth2.Token, error) {
	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	verifier := oauth2.GenerateVerifier()

	type result struct {
		token *oauth2.Token
		err   error
	}
	ch := make(chan result, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:              ":8484",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "invalid state parameter", http.StatusBadRequest)
			ch <- result{err: errors.New("state mismatch")}
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code parameter", http.StatusBadRequest)
			ch <- result{err: errors.New("missing authorization code")}
			return
		}

		token, err := oauthCfg.Exchange(r.Context(), code, oauth2.VerifierOption(verifier))
		if err != nil {
			http.Error(w, "token exchange failed", http.StatusInternalServerError)
			ch <- result{err: fmt.Errorf("exchanging code: %w", err)}
			return
		}

		fmt.Fprint(w, "<html><body><h1>Authorization successful!</h1><p>You may close this window.</p></body></html>")
		ch <- result{token: token}
	})

	// Start the server in a goroutine.
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- result{err: fmt.Errorf("callback server: %w", err)}
		}
	}()

	// Open the browser to the authorization URL with PKCE.
	authURL := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	fmt.Printf("Waiting for authorization on http://localhost:8484/callback...\n")
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Open this URL in your browser:\n%s\n", authURL)
	}

	// Wait for the callback result with a timeout.
	var res result
	select {
	case res = <-ch:
	case <-time.After(oauthFlowTimeout):
		res = result{err: errors.New("authorization timed out after 5 minutes")}
	}

	// Shut down the server.
	if err := srv.Shutdown(context.Background()); err != nil {
		return nil, fmt.Errorf("shutting down callback server: %w", err)
	}

	return res.token, res.err
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
