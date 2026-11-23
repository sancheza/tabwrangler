// Package telemetry collects process-level memory and CPU resource metrics
// and correlates them with browser tabs.
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

// ProcessCollector defines the interface for harvesting browser process metrics.
type ProcessCollector interface {
	CollectBrowserProcesses() (map[types.BrowserType][]ProcessInfo, error)
}

// EnrichTabsWithMetrics matches harvested browser process metrics to tab entries.
func EnrichTabsWithMetrics(tabs []types.Tab, procs map[types.BrowserType][]ProcessInfo) []types.Tab {
	if len(tabs) == 0 || len(procs) == 0 {
		return tabs
	}

	// Calculate aggregate renderer and content metrics per browser
	type browserAgg struct {
		totalRAM   uint64
		totalCPU   float64
		procCount  int
		renderers  []ProcessInfo
	}
	aggs := make(map[types.BrowserType]*browserAgg)

	for bt, pList := range procs {
		agg := &browserAgg{}
		for _, p := range pList {
			if p.IsRenderer {
				agg.totalRAM += p.RAMBytes
				agg.totalCPU += p.CPUPercent
				agg.procCount++
				agg.renderers = append(agg.renderers, p)
			}
		}
		aggs[bt] = agg
	}

	// Count tabs per browser
	tabCounts := make(map[types.BrowserType]int)
	for _, t := range tabs {
		if !t.IsDiscarded {
			tabCounts[t.Browser]++
		}
	}

	result := make([]types.Tab, len(tabs))
	copy(result, tabs)

	for i := range result {
		t := &result[i]
		if t.IsDiscarded {
			// Discarded tabs have negligible footprint
			t.Metrics = types.Metrics{
				CPUPercent: 0.0,
				RAMBytes:   15 * 1024 * 1024,
				Scope:      types.MetricScopeTab,
			}
			continue
		}

		agg, hasAgg := aggs[t.Browser]
		if !hasAgg || agg.procCount == 0 {
			continue
		}

		// If tab already has an explicit ProcessID that matches a specific process
		matched := false
		if t.Metrics.ProcessID > 0 {
			for _, p := range agg.renderers {
				if p.PID == t.Metrics.ProcessID {
					t.Metrics.CPUPercent = p.CPUPercent
					t.Metrics.RAMBytes = p.RAMBytes
					t.Metrics.Scope = types.MetricScopeTab
					matched = true
					break
				}
			}
		}

		if matched {
			continue
		}

		// Proportional allocation across active tabs for shared multi-process engines
		count := tabCounts[t.Browser]
		if count > 0 {
			shareRAM := agg.totalRAM / uint64(count)
			shareCPU := agg.totalCPU / float64(count)

			if agg.procCount >= count {
				t.Metrics.CPUPercent = shareCPU
				t.Metrics.RAMBytes = shareRAM
				t.Metrics.Scope = types.MetricScopeTab
			} else {
				t.Metrics.CPUPercent = shareCPU
				t.Metrics.RAMBytes = shareRAM
				t.Metrics.Scope = types.MetricScopeSharedProcess
			}
		}
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
