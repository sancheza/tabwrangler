package tests

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/sancheza/tabwrangler/core/browser/safari"
)

func TestSafariAdapter(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Safari tests only run on macOS")
	}

	adapter := safari.NewAdapter()
	if !adapter.IsAvailable() {
		t.Skip("osascript is not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tabs, err := adapter.ListTabs(ctx)
	if err != nil {
		t.Logf("Safari returned: %v (expected if Safari is closed or not permitted)", err)
		return
	}
	t.Logf("Found %d open Safari tabs", len(tabs))
	for _, tab := range tabs {
		t.Logf("Safari tab: [%s] %s (%s)", tab.ID, tab.Title, tab.URL)
	}
}
