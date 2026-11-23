// Package types defines common data structures for TabWrangler tabs,
// metrics, and browser state.
package types

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// BrowserType identifies a supported web browser.
type BrowserType string

const (
	BrowserChrome  BrowserType = "chrome"
	BrowserFirefox BrowserType = "firefox"
	BrowserSafari  BrowserType = "safari"
	BrowserAll     BrowserType = "all"
)

// MetricScope indicates the degree of precision for telemetry measurements.
type MetricScope string

const (
	MetricScopeTab           MetricScope = "tab"
	MetricScopeSharedProcess MetricScope = "shared_process"
	MetricScopeUnavailable   MetricScope = "unavailable"
)

// Metrics represents CPU and memory telemetry associated with a tab.
type Metrics struct {
	CPUPercent float64     `json:"cpu_percent"`
	RAMBytes   uint64      `json:"ram_bytes"`
	Scope      MetricScope `json:"scope"`
	ProcessID  int         `json:"process_id,omitempty"`
}

// Capabilities defines what actions can be executed against a tab.
type Capabilities struct {
	CanFocus   bool `json:"can_focus"`
	CanDiscard bool `json:"can_discard"`
	CanClose   bool `json:"can_close"`
}

// Tab represents an open browser tab with metadata and resource metrics.
type Tab struct {
	ID                 string       `json:"id"`
	Browser            BrowserType  `json:"browser"`
	Profile            string       `json:"profile"`
	WindowID           string       `json:"window_id"`
	TabID              string       `json:"tab_id"`
	Title              string       `json:"title"`
	URL                string       `json:"url"`
	Domain             string       `json:"domain"`
	FaviconURL         string       `json:"favicon_url"`
	LastAccessedUnixMs int64        `json:"last_accessed_unix_ms"`
	IsActive           bool         `json:"is_active"`
	IsPinned           bool         `json:"is_pinned"`
	IsDiscarded        bool         `json:"is_discarded"`
	IsSpecial          bool         `json:"is_special"`
	Metrics            Metrics      `json:"metrics"`
	Capabilities       Capabilities `json:"capabilities"`
}

// ExtractDomain parses a URL string and extracts the hostname.
func ExtractDomain(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if strings.HasPrefix(rawURL, "about:") || strings.HasPrefix(rawURL, "chrome://") || strings.HasPrefix(rawURL, "edge://") || strings.HasPrefix(rawURL, "safari-resource://") {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		// Attempt parsing after adding pseudo scheme if missing
		if !strings.Contains(rawURL, "://") {
			u, err = url.Parse("http://" + rawURL)
			if err == nil && u.Host != "" {
				return strings.TrimPrefix(u.Host, "www.")
			}
		}
		return rawURL
	}
	return strings.TrimPrefix(u.Host, "www.")
}

// IsInternalOrSpecialURL checks if a URL belongs to internal browser or extension pages.
func IsInternalOrSpecialURL(rawURL string) bool {
	lower := strings.ToLower(strings.TrimSpace(rawURL))
	if lower == "" || lower == "about:blank" || lower == "about:newtab" {
		return true
	}
	prefixes := []string{
		"chrome://",
		"chrome-extension://",
		"about:",
		"edge://",
		"safari-resource://",
		"file://",
		"moz-extension://",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// FormattedRAM returns a human-readable representation of memory usage.
func (t *Tab) FormattedRAM() string {
	if t.Metrics.Scope == MetricScopeUnavailable || t.Metrics.RAMBytes == 0 {
		return "—"
	}

	const unit = 1024
	b := float64(t.Metrics.RAMBytes)
	var formatted string
	if b < unit*unit {
		formatted = fmt.Sprintf("%.0f KB", b/unit)
	} else if b < unit*unit*unit {
		formatted = fmt.Sprintf("%.0f MB", b/(unit*unit))
	} else {
		formatted = fmt.Sprintf("%.1f GB", b/(unit*unit*unit))
	}

	if t.Metrics.Scope == MetricScopeSharedProcess {
		return "~" + formatted
	}
	return formatted
}

// FormattedCPU returns a human-readable representation of CPU usage percentage.
func (t *Tab) FormattedCPU() string {
	if t.Metrics.Scope == MetricScopeUnavailable {
		return "—"
	}
	formatted := fmt.Sprintf("%.1f%%", t.Metrics.CPUPercent)
	if t.Metrics.Scope == MetricScopeSharedProcess {
		return "~" + formatted
	}
	return formatted
}

// FormattedLastUsed returns a human-readable relative duration since the tab was last accessed.
func (t *Tab) FormattedLastUsed() string {
	if t.LastAccessedUnixMs <= 0 {
		return "—"
	}
	nowMs := time.Now().UnixMilli()
	diffMs := nowMs - t.LastAccessedUnixMs
	if diffMs < 0 {
		return "now"
	}

	dur := time.Duration(diffMs) * time.Millisecond
	if dur < 1*time.Minute {
		return "now"
	} else if dur < 60*time.Minute {
		return fmt.Sprintf("%dm", int(dur.Minutes()))
	} else if dur < 24*time.Hour {
		return fmt.Sprintf("%dh", int(dur.Hours()))
	}
	days := int(dur.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// StateLabel returns a badge label describing the tab's current state.
func (t *Tab) StateLabel() string {
	if t.IsPinned {
		return "Pinned"
	}
	if t.IsDiscarded {
		return "Discarded"
	}
	if t.IsActive {
		return "Active"
	}
	return "Idle"
}

// BrowserBadge returns a compact 2-character browser abbreviation.
func (t *Tab) BrowserBadge() string {
	switch t.Browser {
	case BrowserChrome:
		return "CH"
	case BrowserFirefox:
		return "FF"
	case BrowserSafari:
		return "SF"
	default:
		return "??"
	}
}
