package tests

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/sancheza/tabwrangler/core/bridge"
	"github.com/sancheza/tabwrangler/core/browser/safari"
	"github.com/sancheza/tabwrangler/core/collector"
	"github.com/sancheza/tabwrangler/core/telemetry"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestCorrelationWithProcesses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bridgeServer := bridge.NewServer()
	_ = bridgeServer.Start(ctx)
	defer bridgeServer.Close()

	coll := collector.NewCollector()
	coll.RegisterAdapter(bridge.NewExtensionAdapter(types.BrowserFirefox, bridgeServer))
	coll.RegisterAdapter(safari.NewAdapter())

	tabs, errs := coll.FetchAll(ctx, 3*time.Second)
	if len(tabs) == 0 {
		t.Fatalf("Expected tabs, got 0; errs: %v", errs)
	}

	procColl := telemetry.NewOSCollector()
	procs, err := procColl.CollectBrowserProcesses()
	if err != nil {
		t.Fatalf("CollectBrowserProcesses error: %v", err)
	}

	enriched := telemetry.EnrichTabsWithMetrics(tabs, procs)

	// Sort tabs by RAM descending to inspect top memory consumers
	sort.Slice(enriched, func(i, j int) bool {
		return enriched[i].Metrics.RAMBytes > enriched[j].Metrics.RAMBytes
	})

	t.Logf("Total tabs: %d", len(enriched))
	nonZeroRAM := 0
	nonZeroCPU := 0
	for _, tab := range enriched {
		if tab.Metrics.RAMBytes > 0 {
			nonZeroRAM++
		}
		if tab.Metrics.CPUPercent > 0 {
			nonZeroCPU++
		}
	}
	t.Logf("Tabs with non-zero RAM: %d / %d", nonZeroRAM, len(enriched))
	t.Logf("Tabs with non-zero CPU: %d / %d", nonZeroCPU, len(enriched))

	t.Logf("Top 10 Tabs by RAM:")
	for i := 0; i < min(10, len(enriched)); i++ {
		tab := enriched[i]
		t.Logf("  [%d] RAM=%s, CPU=%s, Scope=%s | %s (%s)",
			i, tab.FormattedRAM(), tab.FormattedCPU(), tab.Metrics.Scope, tab.Title, tab.Domain)
	}
}
