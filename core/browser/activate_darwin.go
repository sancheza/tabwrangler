//go:build darwin

// Package browser provides cross-platform window activation and host automation utilities.
package browser

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/sancheza/tabwrangler/core/types"
)

// ActivateBrowserApplication raises the specified browser application to the very front on macOS,
// ensuring it appears in front of the terminal window.
func ActivateBrowserApplication(b types.BrowserType) {
	if os.Getenv("CI") != "" {
		return
	}

	var appName string
	switch b {
	case types.BrowserFirefox:
		appName = "Firefox"
	case types.BrowserChrome:
		appName = "Google Chrome"
	case types.BrowserSafari:
		appName = "Safari"
	default:
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	script := `
tell application "System Events"
    set isRunning to (exists application process "` + appName + `")
end tell
if isRunning then
    tell application "` + appName + `" to activate
end if
`
	_ = exec.CommandContext(ctx, "osascript", "-e", script).Run()
}
