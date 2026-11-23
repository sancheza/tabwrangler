// Package bridge provides BrowserAdapter implementations for Chrome and Firefox.
package bridge

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sancheza/tabwrangler/core/types"
)

// ExtensionAdapter implements collector.BrowserAdapter by communicating with an MV3 companion extension.
type ExtensionAdapter struct {
	browser types.BrowserType
	server  *Server
}

// NewExtensionAdapter constructs an adapter for Chrome or Firefox.
func NewExtensionAdapter(browser types.BrowserType, server *Server) *ExtensionAdapter {
	return &ExtensionAdapter{
		browser: browser,
		server:  server,
	}
}

// Browser returns the browser type this adapter handles.
func (a *ExtensionAdapter) Browser() types.BrowserType {
	return a.browser
}

// IsAvailable reports whether a live companion extension bridge is connected.
func (a *ExtensionAdapter) IsAvailable() bool {
	return a.server != nil && a.server.IsBrowserConnected(a.browser)
}

// ListTabs fetches the active tab snapshot from the extension.
func (a *ExtensionAdapter) ListTabs(ctx context.Context) ([]types.Tab, error) {
	resp, err := a.server.SendRequest(ctx, a.browser, Request{
		Action: ActionListTabs,
	})
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("extension returned error: %s", resp.Error)
	}

	tabs := make([]types.Tab, 0, len(resp.Tabs))
	for _, raw := range resp.Tabs {
		tabIDStr := strconv.Itoa(raw.ID)
		winIDStr := strconv.Itoa(raw.WindowID)
		domain := types.ExtractDomain(raw.URL)

		tabs = append(tabs, types.Tab{
			ID:                 fmt.Sprintf("%s:%s:%s", a.browser, winIDStr, tabIDStr),
			Browser:            a.browser,
			Profile:            "Default",
			WindowID:           winIDStr,
			TabID:              tabIDStr,
			Title:              raw.Title,
			URL:                raw.URL,
			Domain:             domain,
			FaviconURL:         raw.FaviconURL,
			LastAccessedUnixMs: raw.LastAccessed,
			IsActive:           raw.Active,
			IsPinned:           raw.Pinned,
			IsDiscarded:        raw.Discarded,
			IsSpecial:          types.IsInternalOrSpecialURL(raw.URL),
			Metrics: types.Metrics{
				CPUPercent: 0.0,
				RAMBytes:   0,
				Scope:      types.MetricScopeUnavailable,
			},
			Capabilities: types.Capabilities{
				CanFocus:   true,
				CanDiscard: !raw.Discarded && !raw.Active,
				CanClose:   true,
			},
		})
	}

	return tabs, nil
}

// FocusTab instructs the extension to activate the tab and focus the window.
func (a *ExtensionAdapter) FocusTab(ctx context.Context, tab types.Tab) error {
	tabID, err := strconv.Atoi(tab.TabID)
	if err != nil {
		return fmt.Errorf("invalid tab id: %w", err)
	}
	winID, err := strconv.Atoi(tab.WindowID)
	if err != nil {
		winID = 0
	}

	resp, err := a.server.SendRequest(ctx, a.browser, Request{
		Action:   ActionFocusTab,
		TabID:    tabID,
		WindowID: winID,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("focus failed: %s", resp.Error)
	}
	return nil
}

// DiscardTabs instructs the extension to discard memory for the specified tabs.
func (a *ExtensionAdapter) DiscardTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	tabIDs := make([]int, 0, len(tabs))
	skipped := 0

	for _, t := range tabs {
		if !t.Capabilities.CanDiscard || t.IsDiscarded || t.IsActive {
			skipped++
			continue
		}
		id, err := strconv.Atoi(t.TabID)
		if err == nil {
			tabIDs = append(tabIDs, id)
		} else {
			skipped++
		}
	}

	if len(tabIDs) == 0 {
		return 0, skipped, nil
	}

	resp, err := a.server.SendRequest(ctx, a.browser, Request{
		Action: ActionDiscardTabs,
		TabIDs: tabIDs,
	})
	if err != nil {
		return 0, skipped, err
	}

	return resp.Discarded, skipped + resp.Skipped, nil
}

// CloseTabs instructs the extension to close the specified tabs.
func (a *ExtensionAdapter) CloseTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	tabIDs := make([]int, 0, len(tabs))
	failed := 0

	for _, t := range tabs {
		id, err := strconv.Atoi(t.TabID)
		if err == nil {
			tabIDs = append(tabIDs, id)
		} else {
			failed++
		}
	}

	if len(tabIDs) == 0 {
		return 0, failed, nil
	}

	resp, err := a.server.SendRequest(ctx, a.browser, Request{
		Action: ActionCloseTabs,
		TabIDs: tabIDs,
	})
	if err != nil {
		return 0, failed, err
	}

	return resp.Closed, failed + resp.Failed, nil
}
