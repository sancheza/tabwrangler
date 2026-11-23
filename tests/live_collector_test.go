package tests

import (
	"context"
	"testing"
	"time"

	"github.com/sancheza/tabwrangler/core/bridge"
	"github.com/sancheza/tabwrangler/core/browser/safari"
	"github.com/sancheza/tabwrangler/core/collector"
	"github.com/sancheza/tabwrangler/core/telemetry"
	"github.com/sancheza/tabwrangler/core/tui"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestLiveCollectorFetch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bridgeServer := bridge.NewServer()
	_ = bridgeServer.Start(ctx)
	defer bridgeServer.Close()

	coll := collector.NewCollector()
	coll.RegisterAdapter(bridge.NewExtensionAdapter(types.BrowserChrome, bridgeServer))
	coll.RegisterAdapter(bridge.NewExtensionAdapter(types.BrowserFirefox, bridgeServer))
	coll.RegisterAdapter(safari.NewAdapter())

	tabs, errs := coll.FetchAll(ctx, 3*time.Second)
	t.Logf("FetchAll completed: %d tabs, errs: %v", len(tabs), errs)

	procColl := telemetry.NewOSCollector()
	var stats map[types.BrowserType]telemetry.BrowserProcessStats
	if procs, err := procColl.CollectBrowserProcesses(); err == nil {
		tabs = telemetry.EnrichTabsWithMetrics(tabs, procs)
		stats = telemetry.CalculateBrowserStats(procs)
	}

	ffStats := stats[types.BrowserFirefox]
	t.Logf("Firefox process tree: Total RAM = %s (%d bytes), Total CPU = %.2f%% across %d processes (%d renderers)",
		tui.FormatBytes(ffStats.TotalRAMBytes), ffStats.TotalRAMBytes, ffStats.TotalCPUPercent, ffStats.ProcessCount, ffStats.RendererCount)

	firefoxCount := 0
	for _, tab := range tabs {
		if tab.Browser == types.BrowserFirefox {
			firefoxCount++
		}
	}

	for i, tab := range tabs[:5] {
		t.Logf("Tab %d: title=%q, cpu=%f (%s), ram=%d (%s), scope=%s",
			i, tab.Title, tab.Metrics.CPUPercent, tab.FormattedCPU(), tab.Metrics.RAMBytes, tab.FormattedRAM(), tab.Metrics.Scope)
	}

	t.Logf("Discovered %d Firefox tabs on live system", firefoxCount)
	if firefoxCount == 0 {
		t.Errorf("Expected to discover live open Firefox tabs from sessionstore fallback, got 0")
	}
}
