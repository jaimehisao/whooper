package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

var (
	serveAllowRemote bool
	serveToken       string
)

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		return false
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip != nil {
		return ip.IsLoopback()
	}
	return host == "localhost"
}

func resolveServeToken() string {
	if serveToken != "" {
		return serveToken
	}
	return os.Getenv("WHOOPER_SERVE_TOKEN")
}

func validateServeBind(addr string, allowRemote bool, token string) error {
	if isLoopbackAddr(addr) {
		return nil
	}
	if !allowRemote {
		return fmt.Errorf("refusing non-loopback bind %q without --allow-remote", addr)
	}
	if token == "" {
		return fmt.Errorf("non-loopback bind requires --token or WHOOPER_SERVE_TOKEN")
	}
	return nil
}

func bearerAuthMiddleware(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) || auth[len(prefix):] != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
