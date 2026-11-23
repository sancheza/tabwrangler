package tests

import (
	"context"
	"testing"
	"time"

	"github.com/sancheza/tabwrangler/core/collector"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestCollectorWithMockAdapters(t *testing.T) {
	coll := collector.NewCollector()
	coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserChrome))
	coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserFirefox))
	coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserSafari))

	ctx := context.Background()
	tabs, errs := coll.FetchAll(ctx, 2*time.Second)

	if len(errs) != 0 {
		t.Fatalf("Expected 0 errors from mock adapters, got %v", errs)
	}

	if len(tabs) == 0 {
		t.Fatal("Expected tabs from mock adapters, got 0")
	}

	// Focus test
	firstTab := tabs[0]
	err := coll.FocusTab(ctx, firstTab)
	if err != nil {
		t.Fatalf("FocusTab returned error: %v", err)
	}

	// Discard test
	var discardTargets []types.Tab
	for _, tab := range tabs {
		if tab.Capabilities.CanDiscard && !tab.IsActive && !tab.IsDiscarded {
			discardTargets = append(discardTargets, tab)
		}
	}

	if len(discardTargets) > 0 {
		discarded, skipped, err := coll.DiscardTabs(ctx, discardTargets)
		if err != nil {
			t.Fatalf("DiscardTabs error: %v", err)
		}
		if discarded == 0 && skipped == 0 {
			t.Fatalf("Expected at least some discarded or skipped, got 0/0")
		}
	}

	// Close test
	closeTarget := []types.Tab{tabs[0]}
	closed, failed, err := coll.CloseTabs(ctx, closeTarget)
	if err != nil {
		t.Fatalf("CloseTabs error: %v", err)
	}
	if closed != 1 || failed != 0 {
		t.Fatalf("Expected 1 closed, 0 failed; got %d/%d", closed, failed)
	}
}
