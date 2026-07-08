//go:build unix

package auth

import (
	"os"
	"path/filepath"
	"syscall"
)

// lockTokenFile acquires an exclusive flock on a sibling lock file so concurrent
// processes serialize token read/write. Returns an unlock function.
func lockTokenFile(tokenPath string) (func(), error) {
	lockPath := tokenPath + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
