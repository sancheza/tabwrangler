package tests

import (
	"testing"

	"github.com/sancheza/tabwrangler/core/browser"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestChooseWindow(t *testing.T) {
	slack := browser.WindowCandidate{Handle: 1, Class: "Chrome_WidgetWin_1", Title: "Slack - Team", Exe: "slack.exe"}
	code := browser.WindowCandidate{Handle: 2, Class: "Chrome_WidgetWin_1", Title: "main.go - Code", Exe: "code.exe"}
	chromeA := browser.WindowCandidate{Handle: 3, Class: "Chrome_WidgetWin_1", Title: "Inbox - Google Chrome", Exe: "chrome.exe"}
	chromeB := browser.WindowCandidate{Handle: 4, Class: "Chrome_WidgetWin_1", Title: "Go Release Notes - Google Chrome", Exe: "chrome.exe"}
	edge := browser.WindowCandidate{Handle: 5, Class: "Chrome_WidgetWin_1", Title: "News - Microsoft Edge", Exe: "msedge.exe"}
	ff := browser.WindowCandidate{Handle: 6, Class: "MozillaWindowClass", Title: "Docs — Mozilla Firefox", Exe: "firefox.exe"}
	thunderbird := browser.WindowCandidate{Handle: 7, Class: "MozillaWindowClass", Title: "Inbox - Thunderbird", Exe: "thunderbird.exe"}
	untitled := browser.WindowCandidate{Handle: 8, Class: "Chrome_WidgetWin_1", Title: "", Exe: "chrome.exe"}
	noExe := browser.WindowCandidate{Handle: 9, Class: "Chrome_WidgetWin_1", Title: "Elevated - Google Chrome"}

	cases := []struct {
		name  string
		b     types.BrowserType
		hint  string
		cands []browser.WindowCandidate
		want  int
	}{
		{"skips other Chromium apps", types.BrowserChrome, "", []browser.WindowCandidate{slack, code, chromeA}, 2},
		{"frontmost browser window without a hint", types.BrowserChrome, "", []browser.WindowCandidate{chromeA, chromeB}, 0},
		{"hint picks the window with the tab", types.BrowserChrome, "Go Release Notes", []browser.WindowCandidate{chromeA, chromeB}, 1},
		{"hint match is case-insensitive", types.BrowserChrome, "go release notes", []browser.WindowCandidate{chromeA, chromeB}, 1},
		{"unmatched hint falls back to frontmost", types.BrowserChrome, "Nothing", []browser.WindowCandidate{chromeA, chromeB}, 0},
		{"chromium family counts as chrome", types.BrowserChrome, "", []browser.WindowCandidate{slack, edge}, 1},
		{"firefox ignores other mozilla apps", types.BrowserFirefox, "", []browser.WindowCandidate{thunderbird, ff}, 1},
		{"firefox ignores chromium windows", types.BrowserFirefox, "", []browser.WindowCandidate{chromeA, ff}, 1},
		{"untitled windows are skipped", types.BrowserChrome, "", []browser.WindowCandidate{untitled, chromeA}, 1},
		{"class decides when the executable is unknown", types.BrowserChrome, "", []browser.WindowCandidate{noExe}, 0},
		{"no browser window", types.BrowserChrome, "", []browser.WindowCandidate{slack, code}, -1},
		{"safari is not handled", types.BrowserSafari, "", []browser.WindowCandidate{chromeA}, -1},
		{"empty list", types.BrowserChrome, "x", nil, -1},
	}
	for _, c := range cases {
		if got := browser.ChooseWindow(c.b, c.hint, c.cands); got != c.want {
			t.Errorf("%s: ChooseWindow = %d, want %d", c.name, got, c.want)
		}
	}
}
