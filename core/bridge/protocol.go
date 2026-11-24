// Package bridge defines IPC protocols and messages exchanged between the TabWrangler
// TUI, the native messaging host, and companion browser extensions.
package bridge

import (
	"encoding/json"

	"github.com/sancheza/tabwrangler/core/types"
)

// Action constants defining bridge RPC commands.
const (
	ActionListTabs    = "list_tabs"
	ActionFocusTab    = "focus_tab"
	ActionDiscardTabs = "discard_tabs"
	ActionCloseTabs   = "close_tabs"
	ActionRegister    = "register"
	ActionPing        = "ping"
)

// Request encapsulates a command dispatched to the companion extension or bridge server.
type Request struct {
	ID        string            `json:"id"`
	Action    string            `json:"action"`
	Browser   types.BrowserType `json:"browser,omitempty"`
	WindowID  int               `json:"window_id,omitempty"`
	TabID     int               `json:"tab_id,omitempty"`
	TabIDs    []int             `json:"tab_ids,omitempty"`
	URL       string            `json:"url,omitempty"`
	Params    map[string]any    `json:"params,omitempty"`
}

// Response encapsulates the result from a bridge RPC command.
type Response struct {
	ID        string            `json:"id"`
	Action    string            `json:"action"`
	Browser   types.BrowserType `json:"browser,omitempty"`
	Success   bool              `json:"success"`
	Error     string            `json:"error,omitempty"`
	Tabs      []RawExtensionTab `json:"tabs,omitempty"`
	Discarded int               `json:"discarded,omitempty"`
	Skipped   int               `json:"skipped,omitempty"`
	Closed    int               `json:"closed,omitempty"`
	Failed    int               `json:"failed,omitempty"`
}

// RawExtensionTab mirrors the tab representation emitted by the MV3 WebExtension.
type RawExtensionTab struct {
	ID           int     `json:"id"`
	WindowID     int     `json:"windowId"`
	Title        string  `json:"title"`
	URL          string  `json:"url"`
	FaviconURL   string  `json:"favIconUrl,omitempty"`
	Active       bool    `json:"active"`
	Pinned       bool    `json:"pinned"`
	Discarded    bool    `json:"discarded"`
	LastAccessed float64 `json:"lastAccessed"`
	Incognito    bool    `json:"incognito"`
	Status       string  `json:"status,omitempty"`
}

// MarshalJSON helper for Request.
func (r *Request) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
