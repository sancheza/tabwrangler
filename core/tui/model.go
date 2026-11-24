// Package tui implements the Bubble Tea v2 terminal interface for TabWrangler.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/sancheza/tabwrangler/core/collector"
	"github.com/sancheza/tabwrangler/core/telemetry"
	"github.com/sancheza/tabwrangler/core/types"
)

// SortColumn represents the active column used for sorting.
type SortColumn string

const (
	SortRAM      SortColumn = "RAM"
	SortCPU      SortColumn = "CPU"
	SortLastUsed SortColumn = "Last Used"
	SortDomain   SortColumn = "Domain"
	SortTitle    SortColumn = "Title"
	SortBrowser  SortColumn = "Browser"
)

// RefreshDoneMsg is emitted when an on-demand tab refresh finishes.
type RefreshDoneMsg struct {
	Tabs         []types.Tab
	Errors       map[types.BrowserType]error
	BrowserStats map[types.BrowserType]telemetry.BrowserProcessStats
}

// ActionDoneMsg is emitted when a focus, discard, or close action completes.
type ActionDoneMsg struct {
	Action    string
	Toast     string
	IsWarning bool
	Err       error
}

// ClearToastMsg is emitted to clear toast messages after a display interval.
type ClearToastMsg struct{}

// TriggerRefreshMsg triggers an immediate on-demand tab snapshot in response to an external event.
type TriggerRefreshMsg struct{}

// Model encapsulates the Bubble Tea state for TabWrangler.
type Model struct {
	collector       *collector.Collector
	dryRun          bool
	width           int
	height          int
	ready           bool
	isRefreshing    bool

	// Tab data
	allTabs         []types.Tab
	visibleTabs     []types.Tab
	selectedIDs     map[string]bool
	browserErrors   map[types.BrowserType]error
	browserStats    map[types.BrowserType]telemetry.BrowserProcessStats

	// Browser filtering & scoping
	activeBrowser   types.BrowserType
	allowedBrowsers []types.BrowserType

	// Search & display toggles
	filterInput     textinput.Model
	isFiltering     bool
	showSpecial     bool
	showFullURL     bool

	// Sorting
	sortCol         SortColumn
	sortDesc        bool

	// Navigation & virtual table viewport
	cursor          int
	viewportOffset  int

	// Overlays & Confirmation
	showHelp        bool
	showDiagnostics bool
	confirmClose    bool
	closingTabs     []types.Tab

	// Feedback
	toast           string
	toastIsWarning  bool
}

// NewModel constructs a Model initialized with the given collector.
func NewModel(c *collector.Collector, initialBrowser types.BrowserType, dryRun bool) Model {
	ti := textinput.New()
	ti.Placeholder = "Filter tabs by title, domain, or URL..."
	ti.Prompt = "/ "
	ti.CharLimit = 120

	return Model{
		collector:       c,
		dryRun:          dryRun,
		activeBrowser:   initialBrowser,
		selectedIDs:     make(map[string]bool),
		browserErrors:   make(map[types.BrowserType]error),
		browserStats:    make(map[types.BrowserType]telemetry.BrowserProcessStats),
		filterInput:     ti,
		showSpecial:     false,
		showFullURL:     false,
		sortCol:         SortRAM,
		sortDesc:        true,
		cursor:          0,
		viewportOffset:  0,
		allowedBrowsers: []types.BrowserType{
			types.BrowserAll,
			types.BrowserChrome,
			types.BrowserFirefox,
			types.BrowserSafari,
		},
	}
}

// TickMsg is emitted periodically to refresh tabs and process stats.
type TickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Init triggers initial on-demand tab snapshot.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.triggerRefresh(),
		doTick(),
	)
}

// triggerRefresh queries all registered browser adapters asynchronously.
func (m *Model) triggerRefresh() tea.Cmd {
	m.isRefreshing = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		tabs, errs := m.collector.FetchAll(ctx, 3*time.Second)

		// Enrich tabs with live process telemetry
		procCollector := telemetry.NewOSCollector()
		var stats map[types.BrowserType]telemetry.BrowserProcessStats
		if procs, err := procCollector.CollectBrowserProcesses(); err == nil && len(procs) > 0 {
			tabs = telemetry.EnrichTabsWithMetrics(tabs, procs)
			stats = telemetry.CalculateBrowserStats(procs)
		}

		return RefreshDoneMsg{
			Tabs:         tabs,
			Errors:       errs,
			BrowserStats: stats,
		}
	}
}

// FilterAndSortTabs recomputes the visibleTabs slice based on active filters and sort criteria.
func (m *Model) FilterAndSortTabs() {
	var filtered []types.Tab
	query := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))

	for _, t := range m.allTabs {
		// Browser filter
		if m.activeBrowser != types.BrowserAll && t.Browser != m.activeBrowser {
			continue
		}

		// Special tabs filter
		if !m.showSpecial && t.IsSpecial {
			continue
		}

		// Text search query
		if query != "" {
			inTitle := strings.Contains(strings.ToLower(t.Title), query)
			inURL := strings.Contains(strings.ToLower(t.URL), query)
			inDomain := strings.Contains(strings.ToLower(t.Domain), query)
			if !inTitle && !inURL && !inDomain {
				continue
			}
		}

		filtered = append(filtered, t)
	}

	// Apply Sorting
	sort.SliceStable(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		var less bool

		switch m.sortCol {
		case SortRAM:
			less = a.Metrics.RAMBytes < b.Metrics.RAMBytes
		case SortCPU:
			less = a.Metrics.CPUPercent < b.Metrics.CPUPercent
		case SortLastUsed:
			less = a.LastAccessedUnixMs < b.LastAccessedUnixMs
		case SortDomain:
			less = strings.ToLower(a.Domain) < strings.ToLower(b.Domain)
		case SortTitle:
			less = strings.ToLower(a.Title) < strings.ToLower(b.Title)
		case SortBrowser:
			less = a.Browser < b.Browser
		default:
			less = a.Metrics.RAMBytes < b.Metrics.RAMBytes
		}

		if m.sortDesc {
			return !less
		}
		return less
	})

	m.visibleTabs = filtered

	// Clamp cursor
	if len(m.visibleTabs) == 0 {
		m.cursor = 0
		m.viewportOffset = 0
	} else {
		if m.cursor >= len(m.visibleTabs) {
			m.cursor = len(m.visibleTabs) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

// SelectedOrHighlightedTabs returns all selected tabs, or the currently highlighted tab if none selected.
func (m *Model) SelectedOrHighlightedTabs() []types.Tab {
	var selected []types.Tab
	for _, t := range m.visibleTabs {
		if m.selectedIDs[t.ID] {
			selected = append(selected, t)
		}
	}
	if len(selected) > 0 {
		return selected
	}
	if len(m.visibleTabs) > 0 && m.cursor < len(m.visibleTabs) {
		return []types.Tab{m.visibleTabs[m.cursor]}
	}
	return nil
}

// AggregateStats calculates the total tab count and aggregate RAM for a given browser scope.
func (m *Model) AggregateStats(b types.BrowserType) (int, uint64) {
	count := 0
	for _, t := range m.allTabs {
		if b == types.BrowserAll || t.Browser == b {
			count++
		}
	}

	// If we have live host process telemetry, use the exact process tree memory footprint
	if len(m.browserStats) > 0 {
		if b == types.BrowserAll {
			var totalRAM uint64
			for _, s := range m.browserStats {
				totalRAM += s.TotalRAMBytes
			}
			return count, totalRAM
		}
		if s, ok := m.browserStats[b]; ok && s.TotalRAMBytes > 0 {
			return count, s.TotalRAMBytes
		}
	}

	// Fallback to summing tab metrics (e.g. in mock mode)
	var ram uint64
	for _, t := range m.allTabs {
		if b == types.BrowserAll || t.Browser == b {
			ram += t.Metrics.RAMBytes
		}
	}
	return count, ram
}

// SelectedSortColumn returns the currently active sorting column.
func (m *Model) SelectedSortColumn() SortColumn {
	return m.sortCol
}

// SetSortColumn sets the active sorting column.
func (m *Model) SetSortColumn(c SortColumn) {
	m.sortCol = c
}

// SetSortDescending configures sort order direction.
func (m *Model) SetSortDescending(desc bool) {
	m.sortDesc = desc
}

// IsSortDescending returns whether the active sort is descending.
func (m *Model) IsSortDescending() bool {
	return m.sortDesc
}

// FormatBytes formats byte quantities into readable strings.
func FormatBytes(b uint64) string {
	const unit = 1024
	f := float64(b)
	if f < unit*unit {
		return fmt.Sprintf("%.0f KB", f/unit)
	} else if f < unit*unit*unit {
		return fmt.Sprintf("%.1f MB", f/(unit*unit))
	}
	return fmt.Sprintf("%.1f GB", f/(unit*unit*unit))
}
