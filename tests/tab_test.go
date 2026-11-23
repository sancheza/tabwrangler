package tests

import (
	"testing"
	"time"

	"github.com/sancheza/tabwrangler/core/types"
)

func TestExtractDomain(t *testing.T) {
	cases := []struct {
		url      string
		expected string
	}{
		{"https://github.com/charmbracelet/bubbletea", "github.com"},
		{"https://www.google.com/search?q=go", "google.com"},
		{"http://localhost:8080/dashboard", "localhost:8080"},
		{"chrome://extensions", "chrome://extensions"},
		{"about:blank", "about:blank"},
		{"github.com/sancheza", "github.com"},
	}

	for _, c := range cases {
		actual := types.ExtractDomain(c.url)
		if actual != c.expected {
			t.Errorf("ExtractDomain(%q) = %q; expected %q", c.url, actual, c.expected)
		}
	}
}

func TestIsInternalOrSpecialURL(t *testing.T) {
	cases := []struct {
		url      string
		expected bool
	}{
		{"chrome://extensions", true},
		{"about:support", true},
		{"about:blank", true},
		{"file:///Users/test/document.pdf", true},
		{"chrome-extension://abcdef/popup.html", true},
		{"https://github.com", false},
		{"https://google.com", false},
	}

	for _, c := range cases {
		actual := types.IsInternalOrSpecialURL(c.url)
		if actual != c.expected {
			t.Errorf("IsInternalOrSpecialURL(%q) = %v; expected %v", c.url, actual, c.expected)
		}
	}
}

func TestFormattedMetrics(t *testing.T) {
	tab := types.Tab{
		Metrics: types.Metrics{
			RAMBytes:   500 * 1024 * 1024,
			CPUPercent: 4.5,
			Scope:      types.MetricScopeTab,
		},
	}

	if tab.FormattedRAM() != "500 MB" {
		t.Errorf("Expected 500 MB, got %s", tab.FormattedRAM())
	}
	if tab.FormattedCPU() != "4.5%" {
		t.Errorf("Expected 4.5%%, got %s", tab.FormattedCPU())
	}

	// Shared process indicator
	tab.Metrics.Scope = types.MetricScopeSharedProcess
	if tab.FormattedRAM() != "~500 MB" {
		t.Errorf("Expected ~500 MB, got %s", tab.FormattedRAM())
	}
	if tab.FormattedCPU() != "~4.5%" {
		t.Errorf("Expected ~4.5%%, got %s", tab.FormattedCPU())
	}

	// Unavailable scope
	tab.Metrics.Scope = types.MetricScopeUnavailable
	if tab.FormattedRAM() != "—" {
		t.Errorf("Expected —, got %s", tab.FormattedRAM())
	}
	if tab.FormattedCPU() != "—" {
		t.Errorf("Expected —, got %s", tab.FormattedCPU())
	}
}

func TestFormattedLastUsed(t *testing.T) {
	nowMs := time.Now().UnixMilli()

	tab := types.Tab{
		LastAccessedUnixMs: nowMs - (10 * 1000), // 10s ago
	}
	if tab.FormattedLastUsed() != "now" {
		t.Errorf("Expected 'now', got %s", tab.FormattedLastUsed())
	}

	tab.LastAccessedUnixMs = nowMs - (15 * 60 * 1000) // 15m ago
	if tab.FormattedLastUsed() != "15m" {
		t.Errorf("Expected '15m', got %s", tab.FormattedLastUsed())
	}

	tab.LastAccessedUnixMs = nowMs - (3 * 3600 * 1000) // 3h ago
	if tab.FormattedLastUsed() != "3h" {
		t.Errorf("Expected '3h', got %s", tab.FormattedLastUsed())
	}

	tab.LastAccessedUnixMs = 0
	if tab.FormattedLastUsed() != "—" {
		t.Errorf("Expected '—', got %s", tab.FormattedLastUsed())
	}
}
