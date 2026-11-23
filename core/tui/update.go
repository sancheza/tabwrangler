// Package tui contains the update event handler and action dispatch for TabWrangler.
package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/sancheza/tabwrangler/core/types"
)

// Update processes Bubble Tea incoming messages and dispatches state updates.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.filterInput.SetWidth(msg.Width - 10)
		return m, nil

	case RefreshDoneMsg:
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

	case TickMsg:
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

	case "g":
		m.cursor = 0
		m.adjustViewport()

	case "G":
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
		for _, t := range m.visibleTabs {
			m.selectedIDs[t.ID] = true
		}

	case "A":
		m.selectedIDs = make(map[string]bool)

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

	case "S":
		m.sortDesc = !m.sortDesc
		m.FilterAndSortTabs()

	case "t":
		m.showSpecial = !m.showSpecial
		m.FilterAndSortTabs()

	case "u":
		m.showFullURL = !m.showFullURL

	case "enter", "f":
		if len(m.visibleTabs) > 0 && m.cursor < len(m.visibleTabs) {
			target := m.visibleTabs[m.cursor]
			return m, m.executeFocusTab(target)
		}

	case "d":
		targets := m.SelectedOrHighlightedTabs()
		if len(targets) > 0 {
			return m, m.executeDiscardTabs(targets)
		}

	case "c":
		targets := m.SelectedOrHighlightedTabs()
		if len(targets) > 3 {
			m.confirmClose = true
			m.closingTabs = targets
			return m, nil
		} else if len(targets) > 0 {
			return m, m.executeCloseTabs(targets)
		}

	case "r":
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
				Toast:  fmt.Sprintf("[DRY-RUN] Would discard %d tabs", len(tabs)),
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
				Toast:     fmt.Sprintf("Discard error: %v", err),
				IsWarning: true,
				Err:       err,
			}
		}

		var toast string
		if skipped > 0 {
			toast = fmt.Sprintf("Discarded %d tabs (%d skipped — unsupported or active)", discarded, skipped)
		} else {
			toast = fmt.Sprintf("Discarded %d tabs", discarded)
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
