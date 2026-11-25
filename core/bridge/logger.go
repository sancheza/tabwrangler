// Package bridge provides logging and IPC messaging.
package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	// maxLogBytes is the size at which the log file is moved to "<log>.1".
	maxLogBytes = 5 << 20
	// maxRotatedAge is how old "<log>.1" may be before it is deleted.
	maxRotatedAge = 72 * time.Hour
)

var (
	logMu     sync.Mutex
	logPath   string
	pruneOnce sync.Once
)

// LogPath returns the path to the persistent log file.
func LogPath() string {
	if logPath != "" {
		return logPath
	}
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			home, _ := os.UserHomeDir()
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		logPath = filepath.Join(localAppData, "TabWrangler", "tabwrangler.log")
	} else {
		uid := os.Getuid()
		logPath = fmt.Sprintf("/tmp/tabwrangler-%d.log", uid)
	}
	return logPath
}

// Log writes a timestamped, PID-tagged log entry to the log file.
func Log(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()

	p := LogPath()
	_ = os.MkdirAll(filepath.Dir(p), 0700)
	pruneOnce.Do(func() { pruneRotated(p) })
	rotateIfLarge(p)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_ = f.Chmod(0600)

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	_, _ = fmt.Fprintf(f, "[%s] [PID %d] %s\n", timestamp, os.Getpid(), msg)
}

// rotateIfLarge moves the log to "<path>.1" when it has reached maxLogBytes,
// replacing any earlier "<path>.1".
func rotateIfLarge(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() < maxLogBytes {
		return
	}
	_ = os.Rename(path, path+".1")
}

// pruneRotated deletes "<path>.1" when it is older than maxRotatedAge.
func pruneRotated(path string) {
	info, err := os.Stat(path + ".1")
	if err != nil || time.Since(info.ModTime()) < maxRotatedAge {
		return
	}
	_ = os.Remove(path + ".1")
}
