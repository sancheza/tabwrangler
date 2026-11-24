//go:build darwin

// Package safari implements the Safari browser adapter using macOS AppleScript / JXA automation.
package safari

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sancheza/tabwrangler/core/types"
)

type safariTabRecord struct {
	WindowID string `json:"window_id"`
	TabIndex int    `json:"tab_index"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	IsActive bool   `json:"is_active"`
}

// Adapter provides Safari automation on macOS.
type Adapter struct{}

// NewAdapter constructs a macOS Safari adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Browser returns types.BrowserSafari.
func (a *Adapter) Browser() types.BrowserType {
	return types.BrowserSafari
}

// IsAvailable checks whether Safari is installed and accessible.
func (a *Adapter) IsAvailable() bool {
	// Verify osascript exists
	_, err := exec.LookPath("osascript")
	return err == nil
}

// ListTabs queries Safari for all open windows and tabs via JXA.
func (a *Adapter) ListTabs(ctx context.Context) ([]types.Tab, error) {
	script := `
(function() {
    try {
        const Safari = Application('Safari');
        if (!Safari.running()) {
            return JSON.stringify([]);
        }
        const results = [];
        const windows = Safari.windows();
        for (let w = 0; w < windows.length; w++) {
            const win = windows[w];
            let winId = "";
            try { winId = String(win.id()); } catch(e) { winId = String(w + 1); }
            const tabs = win.tabs();
            let currentTabIdx = 1;
            try {
                const cur = win.currentTab();
                if (cur) currentTabIdx = cur.index();
            } catch(e) {}

            for (let t = 0; t < tabs.length; t++) {
                const tab = tabs[t];
                let title = "";
                let url = "";
                let idx = t + 1;
                try { title = tab.name() || ""; } catch(e) {}
                try { url = tab.url() || ""; } catch(e) {}
                try { idx = tab.index(); } catch(e) {}

                results.push({
                    window_id: winId,
                    tab_index: idx,
                    title: title,
                    url: url,
                    is_active: (idx === currentTabIdx)
                });
            }
        }
        return JSON.stringify(results);
    } catch(err) {
        return JSON.stringify({error: err.toString()});
    }
})()
`
	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("safari osascript error: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}

	// Check if returned error object
	if strings.Contains(trimmed, `"error":`) {
		var errObj struct {
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(trimmed), &errObj) == nil && errObj.Error != "" {
			return nil, fmt.Errorf("safari automation error: %s", errObj.Error)
		}
	}

	var records []safariTabRecord
	if err := json.Unmarshal([]byte(trimmed), &records); err != nil {
		return nil, fmt.Errorf("failed to parse safari tabs output: %w", err)
	}

	tabs := make([]types.Tab, 0, len(records))
	for _, rec := range records {
		tabIDStr := strconv.Itoa(rec.TabIndex)
		domain := types.ExtractDomain(rec.URL)
		tabs = append(tabs, types.Tab{
			ID:                 fmt.Sprintf("safari:%s:%s", rec.WindowID, tabIDStr),
			Browser:            types.BrowserSafari,
			Profile:            "Default",
			WindowID:           rec.WindowID,
			TabID:              tabIDStr,
			Title:              rec.Title,
			URL:                rec.URL,
			Domain:             domain,
			LastAccessedUnixMs: 0,
			IsActive:           rec.IsActive,
			IsPinned:           false,
			IsDiscarded:        false,
			IsSpecial:          types.IsInternalOrSpecialURL(rec.URL),
			Metrics: types.Metrics{
				CPUPercent: 0.0,
				RAMBytes:   0,
				Scope:      types.MetricScopeUnavailable,
			},
			Capabilities: types.Capabilities{
				CanFocus:   true,
				CanDiscard: false, // Discard unsupported on Safari
				CanClose:   true,
			},
		})
	}

	return tabs, nil
}

// FocusTab raises Safari, sets the active window, and focuses the target tab.
func (a *Adapter) FocusTab(ctx context.Context, tab types.Tab) error {
	tabIdx, err := strconv.Atoi(tab.TabID)
	if err != nil {
		return fmt.Errorf("invalid safari tab index: %w", err)
	}

	script := fmt.Sprintf(`
(function() {
    const Safari = Application('Safari');
    Safari.activate();
    let win = null;
    try {
        win = Safari.windows.byId(%s);
    } catch(e) {}
    if (!win) {
        const wins = Safari.windows();
        if (wins.length > 0) win = wins[0];
    }
    if (win) {
        const tabs = win.tabs();
        const targetIdx = %d - 1;
        if (targetIdx >= 0 && targetIdx < tabs.length) {
            win.currentTab = tabs[targetIdx];
        }
        win.index = 1;
    }
})()
`, tab.WindowID, tabIdx)

	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
	return cmd.Run()
}

// DiscardTabs returns skipped since Safari does not support discarding tabs.
func (a *Adapter) DiscardTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	return 0, len(tabs), nil
}

// CloseTabs closes the specified Safari tabs in reverse index order to preserve validity.
func (a *Adapter) CloseTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	closed := 0
	failed := 0

	for _, t := range tabs {
		tabIdx, err := strconv.Atoi(t.TabID)
		if err != nil {
			failed++
			continue
		}

		script := fmt.Sprintf(`
(function() {
    const Safari = Application('Safari');
    let win = null;
    try { win = Safari.windows.byId(%s); } catch(e) {}
    if (!win) {
        const wins = Safari.windows();
        if (wins.length > 0) win = wins[0];
    }
    if (win) {
        const tabs = win.tabs();
        const targetIdx = %d - 1;
        if (targetIdx >= 0 && targetIdx < tabs.length) {
            tabs[targetIdx].close();
            return "ok";
        }
    }
    return "not_found";
})()
`, t.WindowID, tabIdx)

		cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) == "ok" {
			closed++
		} else {
			failed++
		}
	}

	return closed, failed, nil
}

// ReopenTabs reopens previously closed tabs in Safari by creating new tabs with the specified URLs.
func (a *Adapter) ReopenTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	opened := 0
	failed := 0

	for _, t := range tabs {
		if t.URL == "" {
			failed++
			continue
		}

		script := fmt.Sprintf(`
(function() {
    const Safari = Application('Safari');
    let win = null;
    try { win = Safari.windows.byId(%s); } catch(e) {}
    if (!win) {
        const wins = Safari.windows();
        if (wins.length > 0) win = wins[0];
    }
    if (win) {
        const newTab = Safari.Tab({url: %q});
        win.tabs.push(newTab);
        return "ok";
    }
    return "no_window";
})()
`, t.WindowID, t.URL)

		cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) == "ok" {
			opened++
		} else {
			failed++
		}
	}

	return opened, failed, nil
}

// WakeTabs is a no-op for Safari since Safari does not discard tabs via background APIs.
func (a *Adapter) WakeTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	return len(tabs), 0, nil
}
