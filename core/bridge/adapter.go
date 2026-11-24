// Package bridge provides BrowserAdapter implementations for Chrome and Firefox.
package bridge

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sancheza/tabwrangler/core/browser"
	"github.com/sancheza/tabwrangler/core/browser/firefox"
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

// IsAvailable reports whether a live companion extension bridge is connected,
// or for Firefox if session recovery files are available.
func (a *ExtensionAdapter) IsAvailable() bool {
	if a.server != nil && a.server.IsBrowserConnected(a.browser) {
		return true
	}
	if a.browser == types.BrowserFirefox {
		return len(firefox.FindRecoveryFiles()) > 0
	}
	return false
}

// ExtensionInfo reports whether the companion extension is connected and the version it reported.
func (a *ExtensionAdapter) ExtensionInfo() (connected bool, version string) {
	if a.server == nil {
		return false, ""
	}
	version, connected = a.server.ExtensionVersion(a.browser)
	return connected, version
}

// ServerError returns any background bridge server start error.
func (a *ExtensionAdapter) ServerError() error {
	if a.server != nil {
		return a.server.StartError()
	}
	return nil
}

// ListTabs fetches the active tab snapshot from the extension, falling back to session files for Firefox.
func (a *ExtensionAdapter) ListTabs(ctx context.Context) ([]types.Tab, error) {
	if a.server != nil && a.server.IsBrowserConnected(a.browser) {
		return a.listTabsViaBridge(ctx)
	}

	if a.browser == types.BrowserFirefox {
		files := firefox.FindRecoveryFiles()
		if len(files) > 0 {
			return firefox.ReadSessionTabs(files[0])
		}
	}

	return nil, fmt.Errorf("bridge not connected for %s", a.browser)
}

func (a *ExtensionAdapter) listTabsViaBridge(ctx context.Context) ([]types.Tab, error) {
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
			LastAccessedUnixMs: int64(raw.LastAccessed),
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
	if a.server == nil || !a.server.IsBrowserConnected(a.browser) {
		return fmt.Errorf("%s extension not connected (load extension/ via about:debugging)", a.browser)
	}
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
		URL:      tab.URL,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("focus failed: %s", resp.Error)
	}

	// Raise the browser application to the very front of the window stack on the host OS
	browser.ActivateBrowserWindow(a.browser, tab.Title)
	return nil
}

// DiscardTabs instructs the extension to discard memory for the specified tabs.
func (a *ExtensionAdapter) DiscardTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	return a.discard(ctx, tabs, false)
}

// ForceDiscardTabs also discards tabs that are showing in their window. The extension first switches
// that window to another loaded tab, because a browser cannot discard the tab on screen.
func (a *ExtensionAdapter) ForceDiscardTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	return a.discard(ctx, tabs, true)
}

func (a *ExtensionAdapter) discard(ctx context.Context, tabs []types.Tab, force bool) (int, int, error) {
	if a.server == nil || !a.server.IsBrowserConnected(a.browser) {
		return 0, len(tabs), fmt.Errorf("companion extension not connected for %s", a.browser)
	}
	tabIDs := make([]int, 0, len(tabs))
	skipped := 0

	for _, t := range tabs {
		if t.IsDiscarded || (!force && (!t.Capabilities.CanDiscard || t.IsActive)) {
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
		Force:  force,
	})
	if err != nil {
		return 0, skipped, err
	}

	return resp.Discarded, skipped + resp.Skipped, nil
}

// CloseTabs instructs the extension to close the specified tabs.
func (a *ExtensionAdapter) CloseTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	if a.server == nil || !a.server.IsBrowserConnected(a.browser) {
		return 0, len(tabs), fmt.Errorf("companion extension not connected for %s", a.browser)
	}
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

// ReopenTabs instructs the extension to reopen previously closed tabs by URL.
func (a *ExtensionAdapter) ReopenTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	if a.server == nil || !a.server.IsBrowserConnected(a.browser) {
		return 0, len(tabs), fmt.Errorf("companion extension not connected for %s", a.browser)
	}
	urls := make([]string, 0, len(tabs))
	var winID int
	for _, t := range tabs {
		if t.URL != "" {
			urls = append(urls, t.URL)
			if winID == 0 {
				if wid, err := strconv.Atoi(t.WindowID); err == nil {
					winID = wid
				}
			}
		}
	}
	if len(urls) == 0 {
		return 0, 0, nil
	}

	resp, err := a.server.SendRequest(ctx, a.browser, Request{
		Action:   ActionReopenTabs,
		URLs:     urls,
		WindowID: winID,
	})
	if err != nil {
		return 0, len(tabs), err
	}
	return resp.Opened, resp.Failed, nil
}

// WakeTabs instructs the extension to wake/reload discarded tabs back into memory.
func (a *ExtensionAdapter) WakeTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	if a.server == nil || !a.server.IsBrowserConnected(a.browser) {
		return 0, len(tabs), fmt.Errorf("companion extension not connected for %s", a.browser)
	}
	tabIDs := make([]int, 0, len(tabs))
	for _, t := range tabs {
		if id, err := strconv.Atoi(t.TabID); err == nil {
			tabIDs = append(tabIDs, id)
		}
	}
	if len(tabIDs) == 0 {
		return 0, 0, nil
	}

	resp, err := a.server.SendRequest(ctx, a.browser, Request{
		Action: ActionWakeTabs,
		TabIDs: tabIDs,
	})
	if err != nil {
		return 0, len(tabs), err
	}
	return resp.Woken, resp.Failed, nil
}
