package browser

import (
	"strings"

	"github.com/sancheza/tabwrangler/core/types"
)

// Debug receives diagnostic messages about window activation. The default
// discards them; the program points it at its log. Messages never contain tab
// titles or URLs.
var Debug = func(format string, args ...any) {}

// WindowCandidate describes one visible top-level window.
type WindowCandidate struct {
	Handle uintptr
	Class  string
	Title  string
	Exe    string // lower-case executable file name; empty when unknown
}

var chromiumExes = map[string]bool{
	"chrome.exe": true, "msedge.exe": true, "brave.exe": true,
	"chromium.exe": true, "vivaldi.exe": true,
}

// ownsWindow reports whether the candidate belongs to the given browser.
// The executable name decides when it is known. Without it, the window class
// decides, which is less precise because other applications share classes
// (every Electron app uses the Chromium window class).
func ownsWindow(b types.BrowserType, c WindowCandidate) bool {
	class := strings.ToLower(c.Class)
	switch b {
	case types.BrowserChrome:
		if c.Exe != "" {
			return chromiumExes[c.Exe]
		}
		return class == "chrome_widgetwin_1"
	case types.BrowserFirefox:
		if c.Exe != "" {
			return c.Exe == "firefox.exe"
		}
		return class == "mozillawindowclass"
	}
	return false
}

// ChooseWindow picks the browser window to raise from candidates listed
// front to back. It returns the index, or -1 when none belongs to the browser.
//
// A window whose title contains titleHint wins: a browser window's title
// starts with the title of its active tab, so this finds the window that
// holds the tab just focused. Otherwise the frontmost browser window wins.
func ChooseWindow(b types.BrowserType, titleHint string, cands []WindowCandidate) int {
	first, hintMatch := -1, -1
	hint := strings.ToLower(strings.TrimSpace(titleHint))
	for i, c := range cands {
		if strings.TrimSpace(c.Title) == "" || !ownsWindow(b, c) {
			continue
		}
		if first < 0 {
			first = i
		}
		if hint != "" && hintMatch < 0 && strings.Contains(strings.ToLower(c.Title), hint) {
			hintMatch = i
		}
	}
	if hintMatch >= 0 {
		return hintMatch
	}
	return first
}

// ActivateBrowserWindow raises the browser window that holds the tab with the
// given title. titleHint may be empty, which raises the frontmost browser window.
func ActivateBrowserWindow(b types.BrowserType, titleHint string) {
	activateWindow(b, titleHint)
}
