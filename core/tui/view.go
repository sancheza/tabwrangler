// Package tui provides terminal rendering for TabWrangler components.
package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sancheza/tabwrangler/core/types"
)

// View renders the terminal user interface according to Bubble Tea v2 contract.
func (m Model) View() tea.View {
	if !m.ready {
		v := tea.NewView("Initializing TabWrangler...")
		v.AltScreen = true
		return v
	}

	// If confirmation dialog active
	if m.confirmClose {
		content := m.renderConfirmationModal()
		v := tea.NewView(m.placeOverlay(content))
		v.AltScreen = true
		return v
	}

	// If help modal active
	if m.showHelp {
		content := m.renderHelpModal()
		v := tea.NewView(m.placeOverlay(content))
		v.AltScreen = true
		return v
	}

	// If diagnostics modal active
	if m.showDiagnostics {
		content := m.renderDiagnosticsModal()
		v := tea.NewView(m.placeOverlay(content))
		v.AltScreen = true
		return v
	}

	// Normal View Composition
	var b strings.Builder

	b.WriteString(m.renderTopBar() + "\n")
	b.WriteString(m.renderFilterStrip() + "\n")
	b.WriteString(m.renderTable() + "\n")
	b.WriteString(m.renderDetailFooter() + "\n")
	b.WriteString(m.renderKeybindBar())

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m Model) renderTopBar() string {
	var tabs []string

	browserConfigs := []struct {
		bType types.BrowserType
		key   string
		label string
	}{
		{types.BrowserAll, "0", "All"},
		{types.BrowserChrome, "1", "Chrome"},
		{types.BrowserFirefox, "2", "Firefox"},
		{types.BrowserSafari, "3", "Safari"},
	}

	for _, cfg := range browserConfigs {
		count, ram := m.AggregateStats(cfg.bType)
		var item string

		if cfg.bType != types.BrowserAll {
			if _, hasErr := m.browserErrors[cfg.bType]; hasErr && count == 0 {
				item = fmt.Sprintf("[%s] %s (bridge disconnected)", cfg.key, cfg.label)
			} else {
				item = fmt.Sprintf("[%s] %s (%d · %s)", cfg.key, cfg.label, count, FormatBytes(ram))
			}
		} else {
			item = fmt.Sprintf("[%s] %s (%d · %s)", cfg.key, cfg.label, count, FormatBytes(ram))
		}

		if m.activeBrowser == cfg.bType {
			tabs = append(tabs, StyleTabActive.Render(item))
		} else {
			tabs = append(tabs, StyleTabInactive.Render(item))
		}
	}

	bar := strings.Join(tabs, " ")
	return lipgloss.NewStyle().Width(m.width).Render(bar)
}

func (m Model) renderFilterStrip() string {
	var left string
	if m.isFiltering {
		left = m.filterInput.View()
	} else if m.filterInput.Value() != "" {
		left = fmt.Sprintf("/ %s (press '/' to edit, 'Esc' to clear)", m.filterInput.Value())
	} else {
		left = "Filter: '/' to search"
	}

	sortDir := "▼"
	if !m.sortDesc {
		sortDir = "▲"
	}
	sortLabel := fmt.Sprintf("[%s %s]", m.sortCol, sortDir)

	specialLabel := "[Special Tabs: Hidden]"
	if m.showSpecial {
		specialLabel = "[Special Tabs: Shown]"
	}

	refreshLabel := ""
	if m.isRefreshing {
		refreshLabel = "⟳ Refreshing... "
	}

	right := fmt.Sprintf("%s%s  %s", refreshLabel, sortLabel, specialLabel)
	availWidth := m.width - lipgloss.Width(right) - 4
	if availWidth < 10 {
		availWidth = 10
	}
	leftStyled := lipgloss.NewStyle().MaxWidth(availWidth).Render(left)

	strip := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyled,
		strings.Repeat(" ", max(1, m.width-lipgloss.Width(leftStyled)-lipgloss.Width(right)-2)),
		right,
	)

	return StyleFilterStrip.Width(m.width).Render(strip)
}

func (m Model) renderTable() string {
	tableHeight := m.height - 8
	if tableHeight < 3 {
		tableHeight = 3
	}

	colSel := 4
	colBrw := 5
	colDomain := 20
	colLastUsed := 11
	colCPU := 8
	colRAM := 10
	colState := 11
	colURL := 30

	fixedWidth := colSel + colBrw + colDomain + colLastUsed + colCPU + colRAM + colState + 8
	if m.showFullURL {
		fixedWidth += colURL + 2
	}
	colTitle := m.width - fixedWidth
	if colTitle < 20 {
		colTitle = 20
	}

	var headerLine string
	if m.showFullURL {
		headerLine = fmt.Sprintf(" %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			colSel, "Sel",
			colBrw, "Brw",
			colDomain, "Domain",
			colTitle, "Title",
			colURL, "URL",
			colLastUsed, "Last Used",
			colCPU, "CPU",
			colRAM, "RAM",
			colState, "State",
		)
	} else {
		headerLine = fmt.Sprintf(" %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			colSel, "Sel",
			colBrw, "Brw",
			colDomain, "Domain",
			colTitle, "Title",
			colLastUsed, "Last Used",
			colCPU, "CPU",
			colRAM, "RAM",
			colState, "State",
		)
	}

	var lines []string
	lines = append(lines, StyleTableHeader.Width(m.width).Render(headerLine))

	if len(m.visibleTabs) == 0 {
		msg := "No open tabs matching current filter."
		if m.isRefreshing {
			msg = "Querying browser bridges for active tabs..."
		}
		lines = append(lines, lipgloss.NewStyle().Padding(1, 2).Foreground(ColorMuted).Render(msg))
		for len(lines) < tableHeight+1 {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	start := m.viewportOffset
	end := start + tableHeight
	if end > len(m.visibleTabs) {
		end = len(m.visibleTabs)
	}

	for i := start; i < end; i++ {
		t := m.visibleTabs[i]
		isCursor := (i == m.cursor)
		isSelected := m.selectedIDs[t.ID]

		selBox := "[ ]"
		if isSelected {
			selBox = "[x]"
		}

		brwBadge := t.BrowserBadge()
		domain := truncate(t.Domain, colDomain)
		title := truncate(t.Title, colTitle)
		lastUsed := t.FormattedLastUsed()
		cpu := t.FormattedCPU()
		ram := t.FormattedRAM()
		state := t.StateLabel()

		var rowStr string
		if m.showFullURL {
			urlStr := truncate(t.URL, colURL)
			rowStr = fmt.Sprintf(" %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
				colSel, selBox,
				colBrw, brwBadge,
				colDomain, domain,
				colTitle, title,
				colURL, urlStr,
				colLastUsed, lastUsed,
				colCPU, cpu,
				colRAM, ram,
				colState, state,
			)
		} else {
			rowStr = fmt.Sprintf(" %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
				colSel, selBox,
				colBrw, brwBadge,
				colDomain, domain,
				colTitle, title,
				colLastUsed, lastUsed,
				colCPU, cpu,
				colRAM, ram,
				colState, state,
			)
		}

		style := StyleTableRow
		if isCursor {
			style = StyleTableRowCursor
		} else if isSelected {
			style = StyleTableRowSelected
		}

		lines = append(lines, style.Width(m.width).Render(rowStr))
	}

	for len(lines) < tableHeight+1 {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderDetailFooter() string {
	if len(m.visibleTabs) == 0 || m.cursor >= len(m.visibleTabs) {
		return StyleDetailFooter.Width(m.width).Render("No tab selected.")
	}

	t := m.visibleTabs[m.cursor]
	scopeNote := "Metric Scope: exact tab measurement"
	if t.Metrics.Scope == types.MetricScopeSharedProcess {
		scopeNote = "Metric Scope: shared process (~ denotes shared footprint)"
	} else if t.Metrics.Scope == types.MetricScopeUnavailable {
		scopeNote = "Metric Scope: unavailable (— per-tab attribution restricted by browser engine)"
		if s, ok := m.browserStats[t.Browser]; ok && s.TotalRAMBytes > 0 {
			scopeNote = fmt.Sprintf("Metric Scope: unavailable (— browser footprint %s across %d processes)",
				FormatBytes(s.TotalRAMBytes), s.ProcessCount)
		}
	}

	line1 := fmt.Sprintf("Title: %s", t.Title)
	line2 := fmt.Sprintf("URL: %s  |  Profile: %s  |  Window: %s  |  %s",
		t.URL, t.Profile, t.WindowID, scopeNote)

	content := line1 + "\n" + line2
	return StyleDetailFooter.Width(m.width).Render(content)
}

func (m Model) renderKeybindBar() string {
	if m.toast != "" {
		toastStyled := StyleToast.Render(m.toast)
		if m.toastIsWarning {
			toastStyled = StyleToastWarning.Render(m.toast)
		}
		bar := toastStyled + "  " + lipgloss.NewStyle().Faint(true).Render("(Press 'i' for extension setup guide, '?' for help)")
		return StyleKeybindBar.Width(m.width).Render(bar)
	}

	shortcuts := "Enter/f: Focus  |  d: Discard  |  c: Close  |  Space: Select  |  a/A: All/None  |  s: Sort  |  /: Filter  |  r: Refresh  |  ?: Help  |  q: Quit"
	return StyleKeybindBar.Width(m.width).Render(shortcuts)
}

func (m Model) renderConfirmationModal() string {
	num := len(m.closingTabs)
	title := lipgloss.NewStyle().Bold(true).Foreground(ColorWarning).Render("⚠ Close Tabs Confirmation")
	msg := fmt.Sprintf("Close %d selected tabs across browsers?\n\nPress 'y' to confirm, 'n' or 'Esc' to cancel.", num)
	return StyleModal.Render(title + "\n\n" + msg)
}

func (m Model) renderHelpModal() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("TabWrangler — Keyboard Shortcuts")
	table := `
Navigation:
  j / k or ↓ / ↑    Move cursor down / up
  g / G             Jump to top / bottom of list
  Space             Toggle selection on highlighted tab
  a / A             Select all visible tabs / Clear selection
  0 – 3             Filter browser (0: All, 1: Chrome, 2: Firefox, 3: Safari)
  [ / ]             Cycle through browser scopes

Filtering & View:
  /                 Enter fuzzy/substring search
  Esc               Clear search filter / cancel modal
  s / S             Cycle sort column (RAM, CPU, Last Used, Browser, Title) / Reverse
  t                 Toggle special/internal tabs (chrome://*, about:*)
  u                 Toggle inline full URL column

Actions:
  Enter or f        Focus tab & raise browser window
  d                 Discard/unload selected tabs from memory
  c                 Close selected tabs (confirms if > 3 tabs)
  r                 Refresh tab snapshot on demand
  i                 Show bridge & permissions diagnostics
  ?                 Toggle this help overlay
  q / Ctrl+C        Quit TabWrangler
`
	return StyleModal.Render(title + "\n" + table + "\nPress Esc or '?' to return.")
}

func (m Model) renderDiagnosticsModal() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("TabWrangler — Diagnostics & Bridge Status")
	var items []string

	for _, b := range []types.BrowserType{types.BrowserChrome, types.BrowserFirefox, types.BrowserSafari} {
		count, ram := m.AggregateStats(b)
		status := "\033[32mActive\033[0m"
		if err, hasErr := m.browserErrors[b]; hasErr && count == 0 {
			status = fmt.Sprintf("\033[38;5;214mDisconnected (%v)\033[0m", err)
		}

		procInfo := ""
		if s, ok := m.browserStats[b]; ok && s.ProcessCount > 0 {
			procInfo = fmt.Sprintf(" [%d procs: %d renderers, Total CPU: %.1f%%]",
				s.ProcessCount, s.RendererCount, s.TotalCPUPercent)
		}

		items = append(items, fmt.Sprintf("• %s: %s | Tabs: %d | Host Footprint: %s%s",
			strings.Title(string(b)), status, count, FormatBytes(ram), procInfo))
	}

	items = append(items, "")
	items = append(items, lipgloss.NewStyle().Bold(true).Render("To enable interactive tab actions (Focus / Discard / Close):"))
	items = append(items, "  • Firefox: Open about:debugging#/runtime/this-firefox -> click 'Load Temporary Add-on' -> select extension/manifest.json")
	items = append(items, "  • Chrome:  Open chrome://extensions -> enable 'Developer mode' -> 'Load unpacked' -> select extension/")
	items = append(items, "  • Safari:  Automated natively via macOS AppleScript (Focus and Close work out-of-the-box)")
	items = append(items, "")
	items = append(items, "IPC Bridge Socket: "+lipgloss.NewStyle().Faint(true).Render("/tmp/tabwrangler-$UID.sock"))
	items = append(items, "Extension Code: 100% first-party Manifest V3 (no third-party dependencies or external requests)")
	items = append(items, "")
	items = append(items, "Press 'Esc' or 'i' to close.")

	return StyleModal.Render(title + "\n\n" + strings.Join(items, "\n"))
}

func (m Model) placeOverlay(content string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
