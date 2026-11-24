package bridge

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// TokenPath returns the location of the per-run authentication token file that
// belongs to the IPC server listening on sockPath.
func TokenPath(sockPath string) string {
	if isTCPAddr(sockPath) {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" || runtime.GOOS != "windows" {
			if dir, err := os.UserCacheDir(); err == nil {
				base = dir
			} else {
				base = os.TempDir()
			}
		}
		return filepath.Join(base, "TabWrangler", "bridge.token")
	}
	return sockPath + ".token"
}

// newToken returns a random 256-bit token encoded as hex.
func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed generating bridge token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// writeToken stores token at path with owner-only permissions, replacing any previous file.
func writeToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed creating token directory: %w", err)
	}
	_ = os.Remove(path)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed creating token file: %w", err)
	}
	if _, err := f.WriteString(token); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed writing token file: %w", err)
	}
	return f.Close()
}

// ReadToken returns the current per-run token for the server listening on sockPath.
func ReadToken(sockPath string) (string, error) {
	data, err := os.ReadFile(TokenPath(sockPath))
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", errors.New("bridge token file is empty")
	}
	return token, nil
}

// tokenMatches compares two tokens in constant time.
func tokenMatches(expected, got string) bool {
	if expected == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}
