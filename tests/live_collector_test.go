package tests

import (
	"context"
	"testing"
	"time"

	"github.com/sancheza/tabwrangler/core/bridge"
	"github.com/sancheza/tabwrangler/core/browser/safari"
	"github.com/sancheza/tabwrangler/core/collector"
	"github.com/sancheza/tabwrangler/core/telemetry"
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
	if procs, err := procColl.CollectBrowserProcesses(); err == nil {
		tabs = telemetry.EnrichTabsWithMetrics(tabs, procs)
	}

	firefoxCount := 0
	for _, tab := range tabs {
		if tab.Browser == types.BrowserFirefox {
			firefoxCount++
		}
	}

	t.Logf("Discovered %d Firefox tabs on live system", firefoxCount)
	if firefoxCount == 0 {
		t.Errorf("Expected to discover live open Firefox tabs from sessionstore fallback, got 0")
	}
}
