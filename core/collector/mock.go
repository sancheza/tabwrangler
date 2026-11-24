// Package collector provides a mock browser adapter for testing and isolated TUI execution.
package collector

import (
	"context"
	"sync"
	"time"

	"github.com/sancheza/tabwrangler/core/types"
)

// MockAdapter simulates browser state in-memory for testing and standalone execution.
type MockAdapter struct {
	mu      sync.Mutex
	browser types.BrowserType
	tabs    []types.Tab
}

// NewMockAdapter constructs a populated MockAdapter.
func NewMockAdapter(browser types.BrowserType) *MockAdapter {
	m := &MockAdapter{
		browser: browser,
	}
	m.tabs = m.generateDefaultTabs()
	return m
}

func (m *MockAdapter) generateDefaultTabs() []types.Tab {
	nowMs := time.Now().UnixMilli()

	switch m.browser {
	case types.BrowserChrome:
		return []types.Tab{
			{
				ID:                 "chrome:1:101",
				Browser:            types.BrowserChrome,
				Profile:            "Default",
				WindowID:           "1",
				TabID:              "101",
				Title:              "Charmbracelet Bubble Tea v2 Architecture",
				URL:                "https://github.com/charmbracelet/bubbletea",
				Domain:             "github.com",
				LastAccessedUnixMs: nowMs - (2 * 60 * 1000), // 2m ago
				IsActive:           true,
				IsPinned:           false,
				IsDiscarded:        false,
				IsSpecial:          false,
				Metrics: types.Metrics{
					CPUPercent: 4.2,
					RAMBytes:   412 * 1024 * 1024,
					Scope:      types.MetricScopeTab,
					ProcessID:  48102,
				},
				Capabilities: types.Capabilities{
					CanFocus:   true,
					CanDiscard: true,
					CanClose:   true,
				},
			},
			{
				ID:                 "chrome:1:102",
				Browser:            types.BrowserChrome,
				Profile:            "Default",
				WindowID:           "1",
				TabID:              "102",
				Title:              "Google Cloud Monitoring Dashboard",
				URL:                "https://console.cloud.google.com/monitoring",
				Domain:             "console.cloud.google.com",
				LastAccessedUnixMs: nowMs - (35 * 60 * 1000), // 35m ago
				IsActive:           false,
				IsPinned:           true,
				IsDiscarded:        false,
				IsSpecial:          false,
				Metrics: types.Metrics{
					CPUPercent: 8.5,
					RAMBytes:   780 * 1024 * 1024,
					Scope:      types.MetricScopeSharedProcess,
					ProcessID:  48105,
				},
				Capabilities: types.Capabilities{
					CanFocus:   true,
					CanDiscard: true,
					CanClose:   true,
				},
			},
			{
				ID:                 "chrome:1:103",
				Browser:            types.BrowserChrome,
				Profile:            "Default",
				WindowID:           "1",
				TabID:              "103",
				Title:              "Settings — Extensions",
				URL:                "chrome://extensions",
				Domain:             "chrome://extensions",
				LastAccessedUnixMs: nowMs - (4 * 3600 * 1000), // 4h ago
				IsActive:           false,
				IsPinned:           false,
				IsDiscarded:        false,
				IsSpecial:          true,
				Metrics: types.Metrics{
					CPUPercent: 0.0,
					RAMBytes:   48 * 1024 * 1024,
					Scope:      types.MetricScopeTab,
					ProcessID:  48109,
				},
				Capabilities: types.Capabilities{
					CanFocus:   true,
					CanDiscard: false,
					CanClose:   true,
				},
			},
			{
				ID:                 "chrome:1:104",
				Browser:            types.BrowserChrome,
				Profile:            "Default",
				WindowID:           "1",
				TabID:              "104",
				Title:              "Go 1.24 Release Notes & Standard Library Changes",
				URL:                "https://go.dev/doc/go1.24",
				Domain:             "go.dev",
				LastAccessedUnixMs: nowMs - (80 * 60 * 1000), // 1h ago
				IsActive:           false,
				IsPinned:           false,
				IsDiscarded:        true,
				IsSpecial:          false,
				Metrics: types.Metrics{
					CPUPercent: 0.0,
					RAMBytes:   18 * 1024 * 1024,
					Scope:      types.MetricScopeTab,
					ProcessID:  0,
				},
				Capabilities: types.Capabilities{
					CanFocus:   true,
					CanDiscard: false,
					CanClose:   true,
				},
			},
		}

	case types.BrowserFirefox:
		return []types.Tab{
			{
				ID:                 "firefox:2:201",
				Browser:            types.BrowserFirefox,
				Profile:            "default-release",
				WindowID:           "2",
				TabID:              "201",
				Title:              "MDN Web Docs: Web Extensions Native Messaging API",
				URL:                "https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/Native_messaging",
				Domain:             "developer.mozilla.org",
				LastAccessedUnixMs: nowMs - (12 * 60 * 1000), // 12m ago
				IsActive:           true,
				IsPinned:           false,
				IsDiscarded:        false,
				IsSpecial:          false,
				Metrics: types.Metrics{
					CPUPercent: 1.8,
					RAMBytes:   295 * 1024 * 1024,
					Scope:      types.MetricScopeTab,
					ProcessID:  51201,
				},
				Capabilities: types.Capabilities{
					CanFocus:   true,
					CanDiscard: true,
					CanClose:   true,
				},
			},
			{
				ID:                 "firefox:2:202",
				Browser:            types.BrowserFirefox,
				Profile:            "default-release",
				WindowID:           "2",
				TabID:              "202",
				Title:              "Troubleshooting Information",
				URL:                "about:support",
				Domain:             "about:support",
				LastAccessedUnixMs: nowMs - (5 * 3600 * 1000),
				IsActive:           false,
				IsPinned:           false,
				IsDiscarded:        false,
				IsSpecial:          true,
				Metrics: types.Metrics{
					CPUPercent: 0.1,
					RAMBytes:   65 * 1024 * 1024,
					Scope:      types.MetricScopeTab,
					ProcessID:  51204,
				},
				Capabilities: types.Capabilities{
					CanFocus:   true,
					CanDiscard: false,
					CanClose:   true,
				},
			},
			{
				ID:                 "firefox:2:203",
				Browser:            types.BrowserFirefox,
				Profile:            "default-release",
				WindowID:           "2",
				TabID:              "203",
				Title:              "Hacker News — Technology & Startups",
				URL:                "https://news.ycombinator.com",
				Domain:             "news.ycombinator.com",
				LastAccessedUnixMs: nowMs - (150 * 60 * 1000),
				IsActive:           false,
				IsPinned:           false,
				IsDiscarded:        false,
				IsSpecial:          false,
				Metrics: types.Metrics{
					CPUPercent: 0.3,
					RAMBytes:   142 * 1024 * 1024,
					Scope:      types.MetricScopeSharedProcess,
					ProcessID:  51208,
				},
				Capabilities: types.Capabilities{
					CanFocus:   true,
					CanDiscard: true,
					CanClose:   true,
				},
			},
		}

	case types.BrowserSafari:
		return []types.Tab{
			{
				ID:                 "safari:1:1",
				Browser:            types.BrowserSafari,
				Profile:            "Personal",
				WindowID:           "1",
				TabID:              "1",
				Title:              "Apple Developer Documentation — Scripting & Automation",
				URL:                "https://developer.apple.com/documentation",
				Domain:             "developer.apple.com",
				LastAccessedUnixMs: 0, // Safari does not expose tab lastAccessed
				IsActive:           true,
				IsPinned:           false,
				IsDiscarded:        false,
				IsSpecial:          false,
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
			},
			{
				ID:                 "safari:1:2",
				Browser:            types.BrowserSafari,
				Profile:            "Personal",
				WindowID:           "1",
				TabID:              "2",
				Title:              "Wikipedia: Terminal User Interface",
				URL:                "https://en.wikipedia.org/wiki/Text-based_user_interface",
				Domain:             "en.wikipedia.org",
				LastAccessedUnixMs: 0,
				IsActive:           false,
				IsPinned:           false,
				IsDiscarded:        false,
				IsSpecial:          false,
				Metrics: types.Metrics{
					CPUPercent: 0.0,
					RAMBytes:   0,
					Scope:      types.MetricScopeUnavailable,
				},
				Capabilities: types.Capabilities{
					CanFocus:   true,
					CanDiscard: false,
					CanClose:   true,
				},
			},
		}
	}

	return nil
}

func (m *MockAdapter) Browser() types.BrowserType {
	return m.browser
}

func (m *MockAdapter) IsAvailable() bool {
	return true
}

func (m *MockAdapter) ListTabs(ctx context.Context) ([]types.Tab, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]types.Tab, len(m.tabs))
	copy(res, m.tabs)
	return res, nil
}

func (m *MockAdapter) FocusTab(ctx context.Context, tab types.Tab) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.tabs {
		if m.tabs[i].ID == tab.ID {
			m.tabs[i].IsActive = true
			m.tabs[i].LastAccessedUnixMs = time.Now().UnixMilli()
		} else if m.tabs[i].WindowID == tab.WindowID {
			m.tabs[i].IsActive = false
		}
	}
	return nil
}

func (m *MockAdapter) DiscardTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	discarded := 0
	skipped := 0

	ids := make(map[string]bool)
	for _, t := range tabs {
		ids[t.ID] = true
	}

	for i := range m.tabs {
		if ids[m.tabs[i].ID] {
			if !m.tabs[i].Capabilities.CanDiscard || m.tabs[i].IsDiscarded {
				skipped++
			} else {
				m.tabs[i].IsDiscarded = true
				m.tabs[i].Metrics.RAMBytes = 15 * 1024 * 1024 // Reclaimed
				m.tabs[i].Metrics.CPUPercent = 0.0
				discarded++
			}
		}
	}
	return discarded, skipped, nil
}

func (m *MockAdapter) CloseTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make(map[string]bool)
	for _, t := range tabs {
		ids[t.ID] = true
	}

	var remaining []types.Tab
	closed := 0
	for _, t := range m.tabs {
		if ids[t.ID] {
			closed++
		} else {
			remaining = append(remaining, t)
		}
	}
	m.tabs = remaining
	return closed, 0, nil
}

func (m *MockAdapter) ReopenTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	opened := 0
	for _, t := range tabs {
		t.IsActive = false
		m.tabs = append(m.tabs, t)
		opened++
	}
	return opened, 0, nil
}

func (m *MockAdapter) WakeTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make(map[string]bool)
	for _, t := range tabs {
		ids[t.ID] = true
	}

	woken := 0
	for i := range m.tabs {
		if ids[m.tabs[i].ID] {
			m.tabs[i].IsDiscarded = false
			m.tabs[i].Metrics.RAMBytes = 180 * 1024 * 1024
			woken++
		}
	}
	return woken, 0, nil
}
