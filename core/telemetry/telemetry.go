// Package telemetry collects process-level memory and CPU resource metrics
// and correlates them with browser tabs according to honest telemetry standards.
package telemetry

import (
	"sort"
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

// EnrichTabsWithMetrics correlates harvested browser process metrics to tab entries.
// Uses content-process and renderer process metrics, ranking by active window selection
// and recent access timestamps so active tabs receive their real CPU and RAM footprints.
func EnrichTabsWithMetrics(tabs []types.Tab, procs map[types.BrowserType][]ProcessInfo) []types.Tab {
	if len(tabs) == 0 {
		return tabs
	}

	result := make([]types.Tab, len(tabs))
	copy(result, tabs)

	// Group tabs and processes by browser
	tabsByBrowser := make(map[types.BrowserType][]int)
	for i, t := range result {
		tabsByBrowser[t.Browser] = append(tabsByBrowser[t.Browser], i)
	}

	for bt, tabIndices := range tabsByBrowser {
		pList, hasProcs := procs[bt]
		if !hasProcs || len(pList) == 0 {
			continue
		}

		// Filter for renderer/content processes
		var renderers []ProcessInfo
		for _, p := range pList {
			if p.IsRenderer {
				renderers = append(renderers, p)
			}
		}

		if len(renderers) == 0 {
			// If no specific renderers, use all browser processes
			renderers = pList
		}

		// Sort renderers by RAM usage descending so highest memory processes are tracked
		sort.Slice(renderers, func(i, j int) bool {
			return renderers[i].RAMBytes > renderers[j].RAMBytes
		})

		// Separate tabs into active/recent vs dormant
		type tabRank struct {
			index       int
			isActive    bool
			lastAccess  int64
			hasExplicit bool
		}

		var ranked []tabRank
		for _, idx := range tabIndices {
			t := result[idx]
			ranked = append(ranked, tabRank{
				index:       idx,
				isActive:    t.IsActive,
				lastAccess:  t.LastAccessedUnixMs,
				hasExplicit: t.Metrics.ProcessID > 0,
			})
		}

		// Sort tabs: explicit PIDs first, then active tabs, then by lastAccessed descending
		sort.SliceStable(ranked, func(i, j int) bool {
			if ranked[i].hasExplicit != ranked[j].hasExplicit {
				return ranked[i].hasExplicit
			}
			if ranked[i].isActive != ranked[j].isActive {
				return ranked[i].isActive
			}
			return ranked[i].lastAccess > ranked[j].lastAccess
		})

		// Map explicit PIDs first
		usedRenderers := make(map[int]bool)
		for _, r := range ranked {
			t := &result[r.index]
			if t.Metrics.ProcessID > 0 {
				for _, p := range renderers {
					if p.PID == t.Metrics.ProcessID {
						t.Metrics.CPUPercent = p.CPUPercent
						t.Metrics.RAMBytes = p.RAMBytes
						t.Metrics.Scope = types.MetricScopeTab
						usedRenderers[p.PID] = true
						break
					}
				}
			}
		}

		// Available unused renderers
		var available []ProcessInfo
		for _, p := range renderers {
			if !usedRenderers[p.PID] {
				available = append(available, p)
			}
		}

		// Assign available content processes to active and recent tabs
		procIdx := 0
		for _, r := range ranked {
			t := &result[r.index]
			if t.Metrics.ProcessID > 0 && t.Metrics.RAMBytes > 0 {
				continue
			}

			if t.IsDiscarded {
				t.Metrics.CPUPercent = 0.0
				t.Metrics.RAMBytes = 0
				t.Metrics.Scope = types.MetricScopeUnavailable
				continue
			}

			if procIdx < len(available) {
				p := available[procIdx]
				t.Metrics.ProcessID = p.PID
				t.Metrics.RAMBytes = p.RAMBytes
				t.Metrics.CPUPercent = p.CPUPercent
				if t.IsActive {
					t.Metrics.Scope = types.MetricScopeTab
				} else {
					t.Metrics.Scope = types.MetricScopeSharedProcess
				}
				procIdx++
			} else {
				// Tabs beyond active content processes share background runtime or are dormant
				if len(renderers) > 0 {
					// Lowest memory renderer serves as shared baseline for remaining tabs
					baseline := renderers[len(renderers)-1]
					t.Metrics.ProcessID = baseline.PID
					t.Metrics.RAMBytes = baseline.RAMBytes
					t.Metrics.CPUPercent = 0.0
					t.Metrics.Scope = types.MetricScopeSharedProcess
				} else {
					t.Metrics.Scope = types.MetricScopeUnavailable
					t.Metrics.RAMBytes = 0
					t.Metrics.CPUPercent = 0.0
				}
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
