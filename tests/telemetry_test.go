package tests

import (
	"testing"

	"github.com/sancheza/tabwrangler/core/telemetry"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestIsRendererProcess(t *testing.T) {
	cases := []struct {
		name       string
		cmdline    string
		isRenderer bool
		browser    types.BrowserType
	}{
		{"Google Chrome Helper (Renderer)", "--type=renderer", true, types.BrowserChrome},
		{"Google Chrome", "/Applications/Google Chrome.app", false, types.BrowserChrome},
		{"firefox", "/Applications/Firefox.app -contentproc", true, types.BrowserFirefox},
		{"com.apple.WebKit.WebContent", "", true, types.BrowserSafari},
		{"zsh", "/bin/zsh", false, ""},
	}

	for _, c := range cases {
		isRenderer, browser := telemetry.IsRendererProcess(c.name, c.cmdline)
		if isRenderer != c.isRenderer || browser != c.browser {
			t.Errorf("IsRendererProcess(%q, %q) = (%v, %s); expected (%v, %s)",
				c.name, c.cmdline, isRenderer, browser, c.isRenderer, c.browser)
		}
	}
}

func TestEnrichTabsWithMetrics(t *testing.T) {
	tabs := []types.Tab{
		{
			ID:      "chrome:1:1",
			Browser: types.BrowserChrome,
			Title:   "Tab 1",
			Metrics: types.Metrics{Scope: types.MetricScopeUnavailable},
		},
		{
			ID:      "chrome:1:2",
			Browser: types.BrowserChrome,
			Title:   "Tab 2",
			Metrics: types.Metrics{Scope: types.MetricScopeUnavailable},
		},
	}

	procs := map[types.BrowserType][]telemetry.ProcessInfo{
		types.BrowserChrome: {
			{
				PID:        1001,
				Name:       "Google Chrome Helper (Renderer)",
				Browser:    types.BrowserChrome,
				CPUPercent: 10.0,
				RAMBytes:   400 * 1024 * 1024,
				IsRenderer: true,
			},
		},
	}

	enriched := telemetry.EnrichTabsWithMetrics(tabs, procs)

	if len(enriched) != 2 {
		t.Fatalf("Expected 2 tabs, got %d", len(enriched))
	}

	// Because 2 tabs share 1 renderer process, scope should be MetricScopeSharedProcess
	for _, tab := range enriched {
		if tab.Metrics.Scope != types.MetricScopeSharedProcess {
			t.Errorf("Expected scope shared_process, got %s", tab.Metrics.Scope)
		}
		if tab.Metrics.RAMBytes != 200*1024*1024 {
			t.Errorf("Expected 200MB shared RAM, got %d", tab.Metrics.RAMBytes)
		}
	}
}
