package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/sancheza/tabwrangler/core/collector"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestInitialSortOrder(t *testing.T) {
	coll := collector.NewCollector()
	m := NewModel(coll, types.BrowserAll, false)

	if m.SelectedSortColumn() != SortDomain {
		t.Errorf("expected initial sort column %v, got %v", SortDomain, m.SelectedSortColumn())
	}
	if m.IsSortDescending() != false {
		t.Errorf("expected initial sort descending to be false, got %v", m.IsSortDescending())
	}

	// Trigger a WindowSizeMsg so ready=true
	resized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = resized.(Model)

	// Trigger a RefreshDoneMsg
	refreshMsg := RefreshDoneMsg{
		Tabs: []types.Tab{
			{ID: "1", Title: "Zebra", Domain: "zebra.com", Browser: types.BrowserChrome, Metrics: types.Metrics{RAMBytes: 500}},
			{ID: "2", Title: "Apple", Domain: "apple.com", Browser: types.BrowserChrome, Metrics: types.Metrics{RAMBytes: 100}},
		},
		Errors: make(map[types.BrowserType]error),
	}
	updated, _ := m.Update(refreshMsg)
	mUpdated := updated.(Model)

	if mUpdated.SelectedSortColumn() != SortDomain {
		t.Errorf("expected sort column after refresh %v, got %v", SortDomain, mUpdated.SelectedSortColumn())
	}
	if len(mUpdated.visibleTabs) != 2 {
		t.Fatalf("expected 2 visible tabs, got %d", len(mUpdated.visibleTabs))
	}
	if mUpdated.visibleTabs[0].Domain != "apple.com" {
		t.Errorf("expected first tab to be apple.com, got %s", mUpdated.visibleTabs[0].Domain)
	}

	view := mUpdated.View()
	if !strings.Contains(view.Content, "[Domain ▲]") {
		t.Errorf("expected view to contain '[Domain ▲]', got:\n%s", view.Content)
	}
}

func TestUndoStackOperations(t *testing.T) {
	coll := collector.NewCollector()
	coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserChrome))

	m := NewModel(coll, types.BrowserAll, false)
	resized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = resized.(Model)

	tab1 := types.Tab{
		ID:           "chrome:1:1",
		Browser:      types.BrowserChrome,
		WindowID:     "1",
		TabID:        "1",
		Title:        "Tab One",
		URL:          "https://example.com/1",
		Domain:       "example.com",
		Capabilities: types.Capabilities{CanClose: true, CanDiscard: true},
	}
	tab2 := types.Tab{
		ID:           "chrome:1:2",
		Browser:      types.BrowserChrome,
		WindowID:     "1",
		TabID:        "2",
		Title:        "Tab Two",
		URL:          "https://example.com/2",
		Domain:       "example.com",
		Capabilities: types.Capabilities{CanClose: true, CanDiscard: true},
	}

	refreshMsg := RefreshDoneMsg{
		Tabs:   []types.Tab{tab1, tab2},
		Errors: make(map[types.BrowserType]error),
	}
	updated, _ := m.Update(refreshMsg)
	m = updated.(Model)

	// Test 1: Press 'u' with empty undo stack -> toast warning
	uKey := tea.KeyPressMsg(tea.Key{Text: "u", Code: 'u'})
	updated, _ = m.Update(uKey)
	m = updated.(Model)
	if m.toast != "Nothing to undo" {
		t.Fatalf("expected 'Nothing to undo', got %q", m.toast)
	}

	// Test 2: Discard highlighted tab (press 'd')
	dKey := tea.KeyPressMsg(tea.Key{Text: "d", Code: 'd'})
	updated, cmd := m.Update(dKey)
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected non-nil cmd on discard")
	}
	if m.UndoStackLen() != 1 {
		t.Fatalf("expected undo stack len 1 after discard, got %d", m.UndoStackLen())
	}
	if m.undoStack[0].Type != UndoDiscard {
		t.Fatalf("expected UndoDiscard action, got %v", m.undoStack[0].Type)
	}

	// Test 3: Press 'u' to undo discard
	updated, undoCmd := m.Update(uKey)
	m = updated.(Model)
	if undoCmd == nil {
		t.Fatal("expected non-nil cmd on undo discard")
	}
	if m.UndoStackLen() != 0 {
		t.Fatalf("expected undo stack len 0 after undo, got %d", m.UndoStackLen())
	}

	// Test 4: Close tab (press 'c')
	cKey := tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'})
	updated, closeCmd := m.Update(cKey)
	m = updated.(Model)
	if closeCmd == nil {
		t.Fatal("expected non-nil cmd on close")
	}
	if m.UndoStackLen() != 1 {
		t.Fatalf("expected undo stack len 1 after close, got %d", m.UndoStackLen())
	}
	if m.undoStack[0].Type != UndoClose {
		t.Fatalf("expected UndoClose action, got %v", m.undoStack[0].Type)
	}

	// Test 5: Press 'u' to undo close
	updated, undoCloseCmd := m.Update(uKey)
	m = updated.(Model)
	if undoCloseCmd == nil {
		t.Fatal("expected non-nil cmd on undo close")
	}
	if m.UndoStackLen() != 0 {
		t.Fatalf("expected undo stack len 0 after undo close, got %d", m.UndoStackLen())
	}

	// Test 6: Press 'U' to toggle full URL
	bigUKey := tea.KeyPressMsg(tea.Key{Text: "U", Code: 'U'})
	if m.showFullURL {
		t.Fatal("expected showFullURL initial false")
	}
	updated, _ = m.Update(bigUKey)
	m = updated.(Model)
	if !m.showFullURL {
		t.Fatal("expected showFullURL true after pressing U")
	}
}
