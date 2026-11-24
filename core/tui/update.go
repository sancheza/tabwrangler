// Package tui contains the update event handler and action dispatch for TabWrangler.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/sancheza/tabwrangler/core/bridge"
	"github.com/sancheza/tabwrangler/core/types"
)

// Update processes Bubble Tea incoming messages and dispatches state updates.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			bridge.Log("ui: first window size %dx%d", msg.Width, msg.Height)
		}
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.filterInput.SetWidth(msg.Width - 10)
		return m, nil

	case RefreshDoneMsg:
		if msg.Seq != 0 && msg.Seq < m.appliedSeq {
			// A newer refresh already finished; this one started earlier and would replace fresher data.
			bridge.Log("ui: refresh #%d ignored (newer result #%d already shown)", msg.Seq, m.appliedSeq)
			return m, nil
		}
		bridge.Log("ui: refresh #%d applied: %d tabs", msg.Seq, len(msg.Tabs))
		if msg.Seq != 0 {
			m.appliedSeq = msg.Seq
		}
		m.isRefreshing = false
		m.allTabs = msg.Tabs
		m.browserErrors = msg.Errors
		if msg.BrowserStats != nil {
			m.browserStats = msg.BrowserStats
		}
		m.FilterAndSortTabs()
		return m, nil

	case ActionDoneMsg:
		m.toast = msg.Toast
		m.toastIsWarning = msg.IsWarning
		// Trigger automatic refresh and toast clearing timer
		clearCmd := tea.Tick(4*time.Second, func(time.Time) tea.Msg {
			return ClearToastMsg{}
		})
		return m, tea.Batch(clearCmd, m.triggerRefresh())

	case ClearToastMsg:
		m.toast = ""
		return m, nil

	case TriggerRefreshMsg:
		bridge.Log("ui: refresh requested by a browser registering")
		return m, m.triggerRefresh()

	case TickMsg:
		bridge.Log("ui: timer refresh")
		return m, tea.Batch(m.triggerRefresh(), doTick())

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	// 1. Text input filter mode
	if m.isFiltering {
		switch keyStr {
		case "enter":
			m.isFiltering = false
			m.filterInput.Blur()
			m.FilterAndSortTabs()
			return m, nil
		case "esc":
			m.isFiltering = false
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.FilterAndSortTabs()
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.FilterAndSortTabs()
			return m, cmd
		}
	}

	// 2. Close confirmation mode
	if m.confirmClose {
		switch keyStr {
		case "y", "Y":
			m.confirmClose = false
			toClose := m.closingTabs
			m.closingTabs = nil
			m.undoStack = append(m.undoStack, UndoAction{Type: UndoClose, Tabs: toClose})
			return m, m.executeCloseTabs(toClose)
		case "n", "N", "esc":
			m.confirmClose = false
			m.closingTabs = nil
			m.toast = "Close cancelled"
			m.toastIsWarning = false
			return m, nil
		default:
			return m, nil
		}
	}

	// 3. Modals (Help / Diagnostics)
	if m.showHelp {
		if keyStr == "esc" || keyStr == "?" || keyStr == "q" {
			m.showHelp = false
		}
		return m, nil
	}
	if m.showDiagnostics {
		if keyStr == "esc" || keyStr == "i" || keyStr == "q" {
			m.showDiagnostics = false
		}
		return m, nil
	}

	// 4. Standard Navigation and Shortcuts
	// A retired key shows a hint and does nothing else.
	if hint, retired := retiredKeys[keyStr]; retired {
		m.toast = hint
		m.toastIsWarning = true
		return m, tea.Tick(4*time.Second, func(time.Time) tea.Msg { return ClearToastMsg{} })
	}
	// Every other capital letter does what its lowercase key does, so Caps Lock is harmless.
	if len(keyStr) == 1 && keyStr[0] >= 'A' && keyStr[0] <= 'Z' {
		keyStr = strings.ToLower(keyStr)
	}
	switch keyStr {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if len(m.visibleTabs) > 0 {
			m.cursor++
			if m.cursor >= len(m.visibleTabs) {
				m.cursor = len(m.visibleTabs) - 1
			}
			m.adjustViewport()
		}

	case "k", "up":
		if len(m.visibleTabs) > 0 {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.adjustViewport()
		}

	case "g", "home":
		m.cursor = 0
		m.adjustViewport()

	case "end":
		if len(m.visibleTabs) > 0 {
			m.cursor = len(m.visibleTabs) - 1
			m.adjustViewport()
		}

	case " ":
		if len(m.visibleTabs) > 0 && m.cursor < len(m.visibleTabs) {
			tabID := m.visibleTabs[m.cursor].ID
			if m.selectedIDs[tabID] {
				delete(m.selectedIDs, tabID)
			} else {
				m.selectedIDs[tabID] = true
			}
		}

	case "a":
		// Select every visible tab; when all are already selected, clear the selection.
		allSelected := len(m.visibleTabs) > 0
		for _, t := range m.visibleTabs {
			if !m.selectedIDs[t.ID] {
				allSelected = false
				break
			}
		}
		if allSelected {
			m.selectedIDs = make(map[string]bool)
		} else {
			for _, t := range m.visibleTabs {
				m.selectedIDs[t.ID] = true
			}
		}

	case "0":
		m.activeBrowser = types.BrowserAll
		m.FilterAndSortTabs()

	case "1":
		m.activeBrowser = types.BrowserChrome
		m.FilterAndSortTabs()

	case "2":
		m.activeBrowser = types.BrowserFirefox
		m.FilterAndSortTabs()

	case "3":
		m.activeBrowser = types.BrowserSafari
		m.FilterAndSortTabs()

	case "[":
		m.cycleBrowser(false)
		m.FilterAndSortTabs()

	case "]":
		m.cycleBrowser(true)
		m.FilterAndSortTabs()

	case "/":
		m.isFiltering = true
		return m, m.filterInput.Focus()

	case "esc":
		if m.filterInput.Value() != "" {
			m.filterInput.SetValue("")
			m.FilterAndSortTabs()
		} else if len(m.selectedIDs) > 0 {
			m.selectedIDs = make(map[string]bool)
		}

	case "s":
		m.StepSort()
		m.FilterAndSortTabs()

	case "o":
		m.sortDesc = !m.sortDesc
		m.FilterAndSortTabs()

	case "t":
		m.showSpecial = !m.showSpecial
		m.FilterAndSortTabs()

	case "u":
		if len(m.undoStack) == 0 {
			m.toast = "Nothing to undo"
			m.toastIsWarning = true
			return m, nil
		}
		lastIdx := len(m.undoStack) - 1
		action := m.undoStack[lastIdx]
		m.undoStack = m.undoStack[:lastIdx]

		if action.Type == UndoClose {
			return m, m.executeUndoClose(action.Tabs)
		} else if action.Type == UndoDiscard {
			return m, m.executeUndoDiscard(action.Tabs)
		}

	case "v":
		m.showFullURL = !m.showFullURL

	case "enter", "f":
		if len(m.visibleTabs) > 0 && m.cursor < len(m.visibleTabs) {
			target := m.visibleTabs[m.cursor]
			return m, m.executeFocusTab(target)
		}

	case "h":
		targets := m.SelectedOrHighlightedTabs()
		if len(targets) > 0 {
			m.undoStack = append(m.undoStack, UndoAction{Type: UndoDiscard, Tabs: targets})
			return m, m.executeDiscardTabs(targets)
		}

	case "c":
		targets := m.SelectedOrHighlightedTabs()
		if len(targets) > 3 {
			m.confirmClose = true
			m.closingTabs = targets
			return m, nil
		} else if len(targets) > 0 {
			m.undoStack = append(m.undoStack, UndoAction{Type: UndoClose, Tabs: targets})
			return m, m.executeCloseTabs(targets)
		}

	case "r":
		bridge.Log("ui: r pressed")
		m.toast = "Refreshing tabs..."
		m.toastIsWarning = false
		return m, m.triggerRefresh()

	case "i":
		m.showDiagnostics = true

	case "?":
		m.showHelp = true
	}

	return m, nil
}

func (m *Model) adjustViewport() {
	tableHeight := m.height - 8
	if tableHeight < 3 {
		tableHeight = 3
	}
	if m.cursor < m.viewportOffset {
		m.viewportOffset = m.cursor
	} else if m.cursor >= m.viewportOffset+tableHeight {
		m.viewportOffset = m.cursor - tableHeight + 1
	}
}

func (m *Model) cycleBrowser(forward bool) {
	browsers := m.allowedBrowsers
	curIdx := 0
	for i, b := range browsers {
		if b == m.activeBrowser {
			curIdx = i
			break
		}
	}
	if forward {
		m.activeBrowser = browsers[(curIdx+1)%len(browsers)]
	} else {
		m.activeBrowser = browsers[(curIdx-1+len(browsers))%len(browsers)]
	}
}

func (m *Model) cycleSortColumn() {
	cols := []SortColumn{SortRAM, SortCPU, SortLastUsed, SortDomain, SortTitle, SortBrowser}
	for i, c := range cols {
		if c == m.sortCol {
			m.sortCol = cols[(i+1)%len(cols)]
			return
		}
	}
	m.sortCol = SortRAM
}

func isPrimarySortDirection(col SortColumn, desc bool) bool {
	switch col {
	case SortRAM, SortCPU, SortLastUsed:
		return desc == true // primary for numeric metrics is highest/newest first
	case SortDomain, SortTitle, SortBrowser:
		return desc == false // primary for text fields is A-Z
	default:
		return desc == true
	}
}

func defaultSortDirection(col SortColumn) bool {
	switch col {
	case SortRAM, SortCPU, SortLastUsed:
		return true // desc
	case SortDomain, SortTitle, SortBrowser:
		return false // asc
	default:
		return true
	}
}

// StepSort flips between primary and secondary direction before advancing to the next column.
func (m *Model) StepSort() {
	if isPrimarySortDirection(m.sortCol, m.sortDesc) {
		m.sortDesc = !m.sortDesc
	} else {
		m.cycleSortColumn()
		m.sortDesc = defaultSortDirection(m.sortCol)
	}
}

func (m *Model) executeFocusTab(tab types.Tab) tea.Cmd {
	if m.dryRun {
		return func() tea.Msg {
			return ActionDoneMsg{
				Action: "focus",
				Toast:  fmt.Sprintf("[DRY-RUN] Would focus '%s' (%s)", tab.Title, tab.Browser),
			}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := m.collector.FocusTab(ctx, tab)
		if err != nil {
			return ActionDoneMsg{
				Action:    "focus",
				Toast:     fmt.Sprintf("Failed focusing tab: %v", err),
				IsWarning: true,
				Err:       err,
			}
		}
		return ActionDoneMsg{
			Action: "focus",
			Toast:  fmt.Sprintf("Focused: %s", tab.Title),
		}
	}
}

func (m *Model) executeDiscardTabs(tabs []types.Tab) tea.Cmd {
	if m.dryRun {
		return func() tea.Msg {
			return ActionDoneMsg{
				Action: "discard",
				Toast:  fmt.Sprintf("[DRY-RUN] Would hibernate %d tabs", len(tabs)),
			}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		discarded, skipped, err := m.collector.DiscardTabs(ctx, tabs)
		if err != nil {
			return ActionDoneMsg{
				Action:    "discard",
				Toast:     fmt.Sprintf("Hibernate error: %v", err),
				IsWarning: true,
				Err:       err,
			}
		}

		var toast string
		if skipped > 0 {
			toast = fmt.Sprintf("Hibernated %d tabs (%d skipped — already hibernated, not supported, or the only loaded tab in its window)", discarded, skipped)
		} else {
			toast = fmt.Sprintf("Hibernated %d tabs", discarded)
		}

		return ActionDoneMsg{
			Action: "discard",
			Toast:  toast,
		}
	}
}

func (m *Model) executeCloseTabs(tabs []types.Tab) tea.Cmd {
	if m.dryRun {
		return func() tea.Msg {
			return ActionDoneMsg{
				Action: "close",
				Toast:  fmt.Sprintf("[DRY-RUN] Would close %d tabs", len(tabs)),
			}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		closed, failed, err := m.collector.CloseTabs(ctx, tabs)
		if err != nil {
			return ActionDoneMsg{
				Action:    "close",
				Toast:     fmt.Sprintf("Close error: %v", err),
				IsWarning: true,
				Err:       err,
			}
		}

		var toast string
		if failed > 0 {
			toast = fmt.Sprintf("Closed %d/%d tabs (%d failed)", closed, len(tabs), failed)
		} else {
			toast = fmt.Sprintf("Closed %d tabs", closed)
		}

		return ActionDoneMsg{
			Action: "close",
			Toast:  toast,
		}
	}
}

func (m *Model) executeUndoClose(tabs []types.Tab) tea.Cmd {
	if m.dryRun {
		return func() tea.Msg {
			return ActionDoneMsg{
				Action: "undo_close",
				Toast:  fmt.Sprintf("[DRY-RUN] Would reopen %d tab(s)", len(tabs)),
			}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		opened, failed, err := m.collector.ReopenTabs(ctx, tabs)
		var toast string
		isWarn := false
		if err != nil {
			toast = fmt.Sprintf("Undo error: %v (reopened %d tabs, %d failed)", err, opened, failed)
			isWarn = true
		} else {
			toast = fmt.Sprintf("Undo: reopened %d closed tab(s)", opened)
		}
		return ActionDoneMsg{
			Action:    "undo_close",
			Toast:     toast,
			IsWarning: isWarn,
		}
	}
}

func (m *Model) executeUndoDiscard(tabs []types.Tab) tea.Cmd {
	if m.dryRun {
		return func() tea.Msg {
			return ActionDoneMsg{
				Action: "undo_discard",
				Toast:  fmt.Sprintf("[DRY-RUN] Would wake %d hibernated tab(s)", len(tabs)),
			}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		woken, failed, err := m.collector.WakeTabs(ctx, tabs)
		var toast string
		isWarn := false
		if err != nil {
			toast = fmt.Sprintf("Undo error: %v (restored %d tabs, %d failed)", err, woken, failed)
			isWarn = true
		} else {
			toast = fmt.Sprintf("Undo: restored %d hibernated tab(s)", woken)
		}
		return ActionDoneMsg{
			Action:    "undo_discard",
			Toast:     toast,
			IsWarning: isWarn,
		}
	}
}

// retiredKeys maps keys that used to run a command to a hint that names the key to use now.
// Pressing one shows the hint and does nothing else.
var retiredKeys = map[string]string{
	"d": "'d' was retired; press 'h' to hibernate",
	"D": "'D' was retired; press 'h' to hibernate",
	"A": "'A' was retired; press 'a' to select all, and again to clear the selection",
	"G": "'G' was retired; press End to jump to the bottom",
	"S": "'S' was retired; press 'o' to flip the sort direction",
	"U": "'U' was retired; press 'v' to show or hide the full URL column",
}
