// Package telemetry collects process-level memory and CPU resource metrics
// and correlates them with browser tabs according to honest telemetry standards.
package telemetry

import (
	"strings"

	"github.com/sancheza/tabwrangler/core/types"
)

// ProcessInfo represents a running OS process associated with a browser.
type ProcessInfo struct {
	PID        int
	PPID       int
	Name       string
	Browser    types.BrowserType
	CPUPercent float64
	RAMBytes   uint64
	IsRenderer bool
}

// BrowserProcessStats aggregates host resource usage for an entire browser process tree.
type BrowserProcessStats struct {
	TotalRAMBytes   uint64
	TotalCPUPercent float64
	ProcessCount    int
	RendererCount   int
}

// ProcessCollector defines the interface for harvesting browser process metrics.
type ProcessCollector interface {
	CollectBrowserProcesses() (map[types.BrowserType][]ProcessInfo, error)
}

// CalculateBrowserStats computes aggregate host resource metrics across all processes for each browser.
func CalculateBrowserStats(procs map[types.BrowserType][]ProcessInfo) map[types.BrowserType]BrowserProcessStats {
	stats := make(map[types.BrowserType]BrowserProcessStats)

	for bt, pList := range procs {
		var totalRAM uint64
		var totalCPU float64
		renderers := 0

		for _, p := range pList {
			totalRAM += p.RAMBytes
			totalCPU += p.CPUPercent
			if p.IsRenderer {
				renderers++
			}
		}

		stats[bt] = BrowserProcessStats{
			TotalRAMBytes:   totalRAM,
			TotalCPUPercent: totalCPU,
			ProcessCount:    len(pList),
			RendererCount:   renderers,
		}
	}

	return stats
}

// EnrichTabsWithMetrics matches harvested browser process metrics to tab entries.
// Adheres strictly to Honest Telemetry: if exact 1:1 tab-to-process attribution
// is not established, scope remains MetricScopeUnavailable ('—') rather than
// inventing a uniform artificial average.
func EnrichTabsWithMetrics(tabs []types.Tab, procs map[types.BrowserType][]ProcessInfo) []types.Tab {
	if len(tabs) == 0 {
		return tabs
	}

	result := make([]types.Tab, len(tabs))
	copy(result, tabs)

	// Build lookup by PID for each browser
	pidsByBrowser := make(map[types.BrowserType]map[int]ProcessInfo)
	for bt, pList := range procs {
		pidsByBrowser[bt] = make(map[int]ProcessInfo)
		for _, p := range pList {
			pidsByBrowser[bt][p.PID] = p
		}
	}

	for i := range result {
		t := &result[i]

		if t.IsDiscarded {
			t.Metrics = types.Metrics{
				CPUPercent: 0.0,
				RAMBytes:   0,
				Scope:      types.MetricScopeUnavailable,
			}
			continue
		}

		// If tab already has an explicit ProcessID that matches a specific process
		if t.Metrics.ProcessID > 0 {
			if browserProcs, ok := pidsByBrowser[t.Browser]; ok {
				if p, matched := browserProcs[t.Metrics.ProcessID]; matched {
					t.Metrics.CPUPercent = p.CPUPercent
					t.Metrics.RAMBytes = p.RAMBytes
					t.Metrics.Scope = types.MetricScopeTab
					continue
				}
			}
		}

		// If tab already has non-zero telemetry from mock provider or bridge
		if t.Metrics.Scope == types.MetricScopeTab || t.Metrics.Scope == types.MetricScopeSharedProcess {
			if t.Metrics.RAMBytes > 0 || t.Metrics.CPUPercent > 0 {
				continue
			}
		}

		// When OS/browser boundary does not expose 1:1 renderer PID to tab mapping,
		// leave as unavailable ('—') to preserve honest telemetry.
		t.Metrics.Scope = types.MetricScopeUnavailable
		t.Metrics.RAMBytes = 0
		t.Metrics.CPUPercent = 0.0
	}

	return result
}

// IsRendererProcess checks if command line or process name signifies a web renderer.
func IsRendererProcess(name, cmdline string) (bool, types.BrowserType) {
	nameLower := strings.ToLower(name)
	cmdLower := strings.ToLower(cmdline)

	// Google Chrome
	if strings.Contains(nameLower, "chrome") {
		if strings.Contains(nameLower, "helper") || strings.Contains(cmdLower, "--type=renderer") {
			return true, types.BrowserChrome
		}
		return false, types.BrowserChrome
	}

	// Mozilla Firefox
	if strings.Contains(nameLower, "firefox") || strings.Contains(nameLower, "plugin-container") {
		if strings.Contains(cmdLower, "-contentproc") || strings.Contains(nameLower, "plugin-container") {
			return true, types.BrowserFirefox
		}
		return false, types.BrowserFirefox
	}

	// Apple Safari WebKit WebContent
	if strings.Contains(nameLower, "webkit") || strings.Contains(nameLower, "safari") {
		if strings.Contains(nameLower, "webcontent") || strings.Contains(cmdLower, "webcontent") {
			return true, types.BrowserSafari
		}
		return false, types.BrowserSafari
	}

	return false, ""
}
