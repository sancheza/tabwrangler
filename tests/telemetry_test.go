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
			Metrics: types.Metrics{
				ProcessID: 1001,
				Scope:     types.MetricScopeUnavailable,
			},
		},
		{
			ID:      "chrome:1:2",
			Browser: types.BrowserChrome,
			Title:   "Tab 2 without PID",
			Metrics: types.Metrics{
				ProcessID: 0,
				Scope:     types.MetricScopeUnavailable,
			},
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
			{
				PID:        1000,
				Name:       "Google Chrome",
				Browser:    types.BrowserChrome,
				CPUPercent: 2.0,
				RAMBytes:   200 * 1024 * 1024,
				IsRenderer: false,
			},
		},
	}

	stats := telemetry.CalculateBrowserStats(procs)
	chromeStats := stats[types.BrowserChrome]
	if chromeStats.TotalRAMBytes != 600*1024*1024 {
		t.Errorf("Expected 600MB total browser RAM, got %d", chromeStats.TotalRAMBytes)
	}
	if chromeStats.ProcessCount != 2 {
		t.Errorf("Expected 2 processes, got %d", chromeStats.ProcessCount)
	}

	enriched := telemetry.EnrichTabsWithMetrics(tabs, procs)

	if len(enriched) != 2 {
		t.Fatalf("Expected 2 tabs, got %d", len(enriched))
	}

	// Tab 1 matched PID 1001
	if enriched[0].Metrics.Scope != types.MetricScopeTab {
		t.Errorf("Expected scope tab for tab 1, got %s", enriched[0].Metrics.Scope)
	}
	if enriched[0].Metrics.RAMBytes != 400*1024*1024 {
		t.Errorf("Expected 400MB RAM, got %d", enriched[0].Metrics.RAMBytes)
	}
	if enriched[0].Metrics.CPUPercent != 10.0 {
		t.Errorf("Expected 10.0%% CPU, got %f", enriched[0].Metrics.CPUPercent)
	}

	// Tab 2 shares available renderer with shared_process scope
	if enriched[1].Metrics.Scope != types.MetricScopeSharedProcess {
		t.Errorf("Expected scope shared_process for tab 2, got %s", enriched[1].Metrics.Scope)
	}
	if enriched[1].Metrics.RAMBytes == 0 {
		t.Errorf("Expected non-zero shared RAM for tab 2, got 0")
	}
}
