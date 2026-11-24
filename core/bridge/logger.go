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

var (
	logMu   sync.Mutex
	logPath string
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
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	_, _ = fmt.Fprintf(f, "[%s] [PID %d] %s\n", timestamp, os.Getpid(), msg)
}
