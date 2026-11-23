// Package tui defines terminal styling, color palettes, and badges for TabWrangler
// following ADR 0020 conventions.
package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	// Brand and Chrome colors
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSecondary = lipgloss.Color("#04B575")
	ColorMuted     = lipgloss.Color("#666666")
	ColorHighlight = lipgloss.Color("#2D1B69")

	// ADR 0020 aligned terminal status colors
	ColorInfo    = lipgloss.Color("#00AF00") // Green
	ColorWarning = lipgloss.Color("#FFAF00") // 256-color amber (#FFAF00 / \033[38;5;214m)
	ColorError   = lipgloss.Color("#E06C75") // Red

	// Browser Badges
	ColorChrome  = lipgloss.Color("#4285F4")
	ColorFirefox = lipgloss.Color("#FF7139")
	ColorSafari  = lipgloss.Color("#007AFF")

	// Base Styles
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	StyleTabActive = lipgloss.NewStyle().
			Bold(true).
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CCCCCC")).
				Background(lipgloss.Color("#2A2A2A")).
				Padding(0, 1)

	StyleFilterStrip = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E5E5")).
				Background(lipgloss.Color("#1E1E2E")).
				Padding(0, 1)

	StyleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#A0A0A0")).
				Background(lipgloss.Color("#181825"))

	StyleTableRow = lipgloss.NewStyle().
			Padding(0, 0)

	StyleTableRowSelected = lipgloss.NewStyle().
				Background(ColorHighlight).
				Bold(true)

	StyleTableRowCursor = lipgloss.NewStyle().
				Background(lipgloss.Color("#3A3A4C")).
				Bold(true)

	StyleDetailFooter = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CCCCCC")).
				Background(lipgloss.Color("#181825")).
				Padding(0, 1).
				Border(lipgloss.NormalBorder(), true, false, false, false).
				BorderForeground(lipgloss.Color("#333344"))

	StyleKeybindBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888899")).
			Background(lipgloss.Color("#11111B")).
			Padding(0, 1)

	StyleToast = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorInfo).
			Padding(0, 1)

	StyleToastWarning = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorWarning).
				Padding(0, 1)

	StyleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Background(lipgloss.Color("#181825")).
			Padding(1, 2)
)
