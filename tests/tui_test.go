package tests

import (
	"testing"

	"github.com/sancheza/tabwrangler/core/collector"
	"github.com/sancheza/tabwrangler/core/tui"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestTUIModelFilteringAndSorting(t *testing.T) {
	coll := collector.NewCollector()
	m := tui.NewModel(coll, types.BrowserAll, false)

	// Simulate refresh with tabs
	tabs := []types.Tab{
		{
			ID:                 "chrome:1:1",
			Browser:            types.BrowserChrome,
			Title:              "Alpha Tab",
			URL:                "https://alpha.com",
			Domain:             "alpha.com",
			LastAccessedUnixMs: 1000,
			Metrics: types.Metrics{
				RAMBytes:   200 * 1024 * 1024,
				CPUPercent: 1.0,
			},
		},
		{
			ID:                 "firefox:1:2",
			Browser:            types.BrowserFirefox,
			Title:              "Beta Tab",
			URL:                "https://beta.com",
			Domain:             "beta.com",
			LastAccessedUnixMs: 2000,
			Metrics: types.Metrics{
				RAMBytes:   800 * 1024 * 1024,
				CPUPercent: 5.0,
			},
		},
		{
			ID:        "chrome:1:3",
			Browser:   types.BrowserChrome,
			Title:     "Special Settings",
			URL:       "chrome://settings",
			Domain:    "chrome://settings",
			IsSpecial: true,
		},
	}

	msg := tui.RefreshDoneMsg{Tabs: tabs}
	updated, _ := m.Update(msg)
	model := updated.(tui.Model)

	// Default: special tabs hidden
	model.FilterAndSortTabs()
	targets := model.SelectedOrHighlightedTabs()
	if len(targets) != 1 {
		t.Fatalf("Expected 1 highlighted tab, got %d", len(targets))
	}

	// Beta Tab has 800MB RAM, Alpha Tab has 200MB. SortRAM descending puts Beta first
	if targets[0].Title != "Beta Tab" {
		t.Fatalf("Expected 'Beta Tab' as top row sorted by RAM desc, got %s", targets[0].Title)
	}

	// Aggregate stats check
	count, ram := model.AggregateStats(types.BrowserAll)
	if count != 3 {
		t.Fatalf("Expected 3 total tabs in aggregate stats, got %d", count)
	}
	if ram != 1000*1024*1024 {
		t.Fatalf("Expected 1000MB total RAM, got %d", ram)
	}

	// Test Sort by Domain ascending
	model.SetSortColumn(tui.SortDomain)
	model.SetSortDescending(false)
	model.FilterAndSortTabs()
	domainSorted := model.SelectedOrHighlightedTabs()
	if len(domainSorted) == 0 || domainSorted[0].Domain != "alpha.com" {
		t.Fatalf("Expected 'alpha.com' first when sorted by Domain asc, got %v", domainSorted)
	}

	// Test Sort by Domain descending
	model.SetSortDescending(true)
	model.FilterAndSortTabs()
	domainSortedDesc := model.SelectedOrHighlightedTabs()
	if len(domainSortedDesc) == 0 || domainSortedDesc[0].Domain != "beta.com" {
		t.Fatalf("Expected 'beta.com' first when sorted by Domain desc, got %v", domainSortedDesc)
	}
}
