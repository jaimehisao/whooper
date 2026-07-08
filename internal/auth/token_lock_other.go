//go:build !unix

package auth

// lockTokenFile is a no-op on non-Unix platforms (in-process mutex still applies).
func lockTokenFile(tokenPath string) (func(), error) {
	return func() {}, nil
}
