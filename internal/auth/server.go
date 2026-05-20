package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

var oauthFlowTimeout = 5 * time.Minute

const oauthServerAddr = "127.0.0.1:8484"
const callbackPath = "/callback"

var runtimeGOOS = runtime.GOOS
var openBrowserFunc = openBrowser

type oauthResult struct {
	token *oauth2.Token
	err   error
}

type tokenExchanger func(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)

// RunOAuthFlow performs the full OAuth2 authorization-code flow with PKCE:
//  1. Generates a random state parameter and PKCE verifier.
//  2. Starts a local HTTP server on :8484 to receive the callback.
//  3. Opens the user's browser to the authorization URL.
//  4. Waits for the callback, exchanges the code for a token.
//  5. Shuts down the server and returns the token.
func RunOAuthFlow(oauthCfg *oauth2.Config) (*oauth2.Token, error) {
	return RunOAuthFlowWithBrowser(oauthCfg, true)
}

// RunOAuthFlowWithBrowser performs the OAuth2 authorization-code flow and
// optionally opens the user's browser to the authorization URL.
func RunOAuthFlowWithBrowser(oauthCfg *oauth2.Config, openBrowser bool) (*oauth2.Token, error) {
	if err := validateRedirectURL(oauthCfg.RedirectURL); err != nil {
		return nil, fmt.Errorf("invalid redirect URL: %w", err)
	}

	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	verifier := oauth2.GenerateVerifier()

	ch := make(chan oauthResult, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:              oauthServerAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		handleOAuthCallback(w, r, state, verifier, oauthCfg.Exchange, ch)
	})

	// Start the server in a goroutine.
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sendOAuthResult(ch, oauthResult{err: fmt.Errorf("callback server: %w", err)})
		}
	}()

	// Open the browser to the authorization URL with PKCE.
	authURL := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	fmt.Printf("Waiting for authorization on http://%s%s...\n", oauthServerAddr, callbackPath)
	fmt.Printf("Open this URL in your browser:\n%s\n", authURL)
	if openBrowser {
		if err := openBrowserFunc(authURL); err != nil {
			fmt.Printf("Could not open browser automatically: %v\n", err)
		}
	} else {
		fmt.Println("Automatic browser opening disabled.")
	}

	// Wait for the callback result with a timeout.
	var res oauthResult
	select {
	case res = <-ch:
	case <-time.After(oauthFlowTimeout):
		res = oauthResult{err: errors.New("authorization timed out after 5 minutes")}
	}

	// Shut down the server.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return nil, fmt.Errorf("shutting down callback server: %w", err)
	}

	return res.token, res.err
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request, state, verifier string, exchange tokenExchanger, ch chan oauthResult) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Query().Get("state") != state {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		sendOAuthResult(ch, oauthResult{err: errors.New("state mismatch")})
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		sendOAuthResult(ch, oauthResult{err: errors.New("missing authorization code")})
		return
	}

	token, err := exchange(r.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		sendOAuthResult(ch, oauthResult{err: fmt.Errorf("exchanging code: %w", err)})
		return
	}

	fmt.Fprint(w, "<html><body><h1>Authorization successful!</h1><p>You may close this window.</p></body></html>")
	sendOAuthResult(ch, oauthResult{token: token})
}

func sendOAuthResult(ch chan oauthResult, res oauthResult) {
	select {
	case ch <- res:
	default:
	}
}

func ValidateRedirectURL(raw string) error {
	return validateRedirectURL(raw)
}

func validateRedirectURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" {
		return fmt.Errorf("redirect URL must use http")
	}
	host := u.Hostname()
	if host != "localhost" && host != "127.0.0.1" {
		return fmt.Errorf("redirect host must be localhost or 127.0.0.1")
	}
	port := u.Port()
	if port == "" {
		port = "80"
	}
	if port != "8484" {
		return fmt.Errorf("redirect port must be 8484")
	}
	if u.Path != callbackPath {
		return fmt.Errorf("redirect path must be %s", callbackPath)
	}
	return nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(rawURL string) error {
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	var cmd *exec.Cmd
	switch runtimeGOOS {
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", runtimeGOOS)
	}
	return cmd.Start()
}
