// Package main provides the entry point for TabWrangler, a cross-browser tab inspector
// and control TUI conforming to Dev ADR 0005 command-line conventions.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/sancheza/tabwrangler/core/browser/safari"
	"github.com/sancheza/tabwrangler/core/bridge"
	"github.com/sancheza/tabwrangler/core/collector"
	"github.com/sancheza/tabwrangler/core/setup"
	"github.com/sancheza/tabwrangler/core/tui"
	"github.com/sancheza/tabwrangler/core/types"
)

const Version = types.Version

// printUsage renders formatted, colorized CLI usage instructions, examples, and keyboard shortcuts to stdout.
func printUsage() {
	useColor := os.Getenv("NO_COLOR") == ""

	cReset := ""
	cDim := ""
	cGreen := ""
	cAmber := ""
	cCyan := ""
	cBoldCyan := ""
	cBoldWhite := ""

	if useColor {
		cReset = "\033[0m"
		cDim = "\033[2m"
		cGreen = "\033[32m"
		cAmber = "\033[38;5;214m" // True 256-color amber per Dev ADR 0020
		cCyan = "\033[36m"
		cBoldCyan = "\033[1;36m"
		cBoldWhite = "\033[1;37m"
	}

	var b strings.Builder

	// Title Banner
	b.WriteString(cBoldCyan + "TabWrangler" + cReset + " " + cDim + "v" + Version + cReset + " — " + cBoldWhite + "Cross-Browser Tab Inspector & Resource Control TUI\n" + cReset)
	b.WriteString(cDim + "Inspect, search, correlate OS process memory/CPU, and manage tabs across Chrome, Firefox, and Safari.\n\n" + cReset)

	// Usage
	b.WriteString(cBoldCyan + "USAGE:\n" + cReset)
	b.WriteString("  " + cGreen + "tabwrangler" + cReset + " [flags]\n")
	b.WriteString("  " + cGreen + "tabwrangler setup" + cReset + " [" + cCyan + "--open" + cReset + "] [" + cCyan + "--chrome-id" + cReset + " <id>]\n")
	b.WriteString("  " + cGreen + "tabwrangler bridge" + cReset + "\n\n")

	// Description
	b.WriteString(cBoldCyan + "DESCRIPTION:\n" + cReset)
	b.WriteString("  TabWrangler aggregates open tabs across Google Chrome, Mozilla Firefox, and Apple\n")
	b.WriteString("  Safari into a single, high-performance terminal interface. It correlates live host\n")
	b.WriteString("  process telemetry (RAM and CPU usage) with individual tabs and gives you instant\n")
	b.WriteString("  keyboard controls to focus windows, discard memory hogs, and close tab clutter.\n\n")

	// Commands
	b.WriteString(cBoldCyan + "COMMANDS:\n" + cReset)
	b.WriteString("  " + cGreen + "setup" + cReset + "             Install Native Messaging manifests and configure host permissions\n")
	b.WriteString("                    Options: " + cCyan + "--open" + cReset + ", " + cCyan + "-o" + cReset + " (opens browser add-on debugging page directly)\n")
	b.WriteString("                             " + cCyan + "--chrome-id" + cReset + " <id> (specifies custom unpacked Chrome extension ID)\n")
	b.WriteString("  " + cGreen + "bridge" + cReset + "            Run stdio Native Messaging host bridge (invoked automatically by browsers)\n\n")

	// Flags
	b.WriteString(cBoldCyan + "FLAGS:\n" + cReset)
	b.WriteString("  " + cCyan + "-h, --help" + cReset + "          Show this help screen and exit\n")
	b.WriteString("  " + cCyan + "-v, --version" + cReset + "       Print version information (" + Version + ") and exit\n")
	b.WriteString("      " + cCyan + "--browser" + cReset + " " + cAmber + "<name>" + cReset + "  Filter initial view: " + cAmber + "all" + cReset + ", " + cAmber + "chrome" + cReset + ", " + cAmber + "firefox" + cReset + ", " + cAmber + "safari" + cReset + " (default: all)\n")
	b.WriteString("      " + cCyan + "--mock" + cReset + "          Launch with simulated multi-browser tab data for testing and demos\n")
	b.WriteString("      " + cCyan + "--dry-run" + cReset + "       Preview tab focus, discard, and close mutations without modifying state\n\n")

	// Keyboard Controls in TUI
	b.WriteString(cBoldCyan + "TUI KEYBOARD SHORTCUTS:\n" + cReset)
	b.WriteString("  " + cBoldWhite + "Navigation:" + cReset + "\n")
	b.WriteString("    " + cAmber + "j" + cReset + " / " + cAmber + "k" + cReset + " or " + cAmber + "↑" + cReset + " / " + cAmber + "↓" + cReset + "   Move selection cursor down / up\n")
	b.WriteString("    " + cAmber + "g" + cReset + " / " + cAmber + "G" + cReset + "          Jump to top / bottom of tab list\n")
	b.WriteString("  " + cBoldWhite + "Actions:" + cReset + "\n")
	b.WriteString("    " + cAmber + "Enter" + cReset + " or " + cAmber + "f" + cReset + "     Focus tab and raise browser window to very front of screen\n")
	b.WriteString("    " + cAmber + "h" + cReset + " or " + cAmber + "d" + cReset + "          Hibernate tab memory (Chrome & Firefox; keeps tab in browser)\n")
	b.WriteString("    " + cAmber + "c" + cReset + "              Close highlighted or selected tabs (prompts confirmation for > 3 tabs)\n")
	b.WriteString("    " + cAmber + "u" + cReset + "              Undo last tab action (reopen closed tab or wake hibernated tab)\n")
	b.WriteString("  " + cBoldWhite + "Selection:" + cReset + "\n")
	b.WriteString("    " + cAmber + "Space" + cReset + "          Toggle checkbox selection on highlighted tab\n")
	b.WriteString("    " + cAmber + "a" + cReset + " / " + cAmber + "A" + cReset + "          Select all visible tabs / Clear all selections\n")
	b.WriteString("  " + cBoldWhite + "Filters & Sorting:" + cReset + "\n")
	b.WriteString("    " + cAmber + "/" + cReset + "              Open live filter query (filters Title, URL, Domain)\n")
	b.WriteString("    " + cAmber + "s" + cReset + "              Step sort direction (most ⇄ least, A ⇄ Z), then advance column\n")
	b.WriteString("    " + cAmber + "S" + cReset + "              Invert current sort direction\n")
	b.WriteString("    " + cAmber + "0" + cReset + "–" + cAmber + "3" + cReset + " or " + cAmber + "[" + cReset + " / " + cAmber + "]" + cReset + "    Switch browser tab filter: [0] All, [1] Chrome, [2] Firefox, [3] Safari\n")
	b.WriteString("  " + cBoldWhite + "Display & Helpers:" + cReset + "\n")
	b.WriteString("    " + cAmber + "t" + cReset + "              Toggle visibility of internal/special tabs (chrome://, about:)\n")
	b.WriteString("    " + cAmber + "U" + cReset + "              Toggle full URL column visibility\n")
	b.WriteString("    " + cAmber + "r" + cReset + "              Refresh tab and host process telemetry snapshot on demand\n")
	b.WriteString("    " + cAmber + "?" + cReset + "              Show in-app diagnostics modal and bridge status\n")
	b.WriteString("    " + cAmber + "q" + cReset + "              Quit TabWrangler\n\n")

	// Examples
	b.WriteString(cBoldCyan + "EXAMPLES:\n" + cReset)
	b.WriteString("  " + cDim + "# Launch interactive TUI aggregating all browsers" + cReset + "\n")
	b.WriteString("  " + cGreen + "tabwrangler" + cReset + "\n\n")
	b.WriteString("  " + cDim + "# Start TUI pre-filtered to Firefox tabs only" + cReset + "\n")
	b.WriteString("  " + cGreen + "tabwrangler --browser firefox" + cReset + "\n\n")
	b.WriteString("  " + cDim + "# Safe dry-run mode: test discarding or closing tabs without making mutations" + cReset + "\n")
	b.WriteString("  " + cGreen + "tabwrangler --dry-run" + cReset + "\n\n")
	b.WriteString("  " + cDim + "# Explore the interface using mock browser data (no browsers required)" + cReset + "\n")
	b.WriteString("  " + cGreen + "tabwrangler --mock" + cReset + "\n\n")
	b.WriteString("  " + cDim + "# Run initial setup and automatically open the Firefox add-on debugging page" + cReset + "\n")
	b.WriteString("  " + cGreen + "tabwrangler setup --open" + cReset + "\n\n")

	// Setup Guide
	b.WriteString(cBoldCyan + "QUICK COMPANION SETUP (Chrome & Firefox):\n" + cReset)
	b.WriteString("  1. Run " + cGreen + "tabwrangler setup --open" + cReset + " to register manifests and copy extension path to clipboard.\n")
	b.WriteString("  2. In Firefox: On " + cCyan + "about:debugging" + cReset + ", click " + cBoldWhite + "This Firefox" + cReset + " -> " + cBoldWhite + "Load Temporary Add-on..." + cReset + " -> paste clipboard (" + cAmber + "Cmd+V" + cReset + ").\n")
	b.WriteString("  3. In Chrome: Navigate to " + cCyan + "chrome://extensions" + cReset + ", enable " + cBoldWhite + "Developer mode" + cReset + ", click " + cBoldWhite + "Load unpacked" + cReset + ", select " + cCyan + "extension/" + cReset + ".\n")
	b.WriteString("  4. Safari works out of the box with zero extensions required via native macOS automation.\n")

	fmt.Print(b.String())
}

// main parses command-line flags and subcommands, initializes the browser collector and IPC bridge,
// and starts the interactive Bubble Tea terminal user interface.
func main() {
	bridge.Log("TabWrangler process started: args=%v", os.Args)
	// Custom flag set to strictly support -v/--version and -h/--help per Dev ADR 0005
	var (
		versionFlagShort bool
		versionFlagLong  bool
		helpFlagShort    bool
		helpFlagLong     bool
		browserFlag      string
		mockFlag         bool
		dryRunFlag       bool
	)

	fs := flag.NewFlagSet("tabwrangler", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = printUsage

	fs.BoolVar(&versionFlagShort, "v", false, "Print version")
	fs.BoolVar(&versionFlagLong, "version", false, "Print version")
	fs.BoolVar(&helpFlagShort, "h", false, "Show help")
	fs.BoolVar(&helpFlagLong, "help", false, "Show help")
	fs.StringVar(&browserFlag, "browser", "all", "Filter initial browser")
	fs.BoolVar(&mockFlag, "mock", false, "Run with mock provider")
	fs.BoolVar(&dryRunFlag, "dry-run", false, "Report mutations without modifying state")

	// Detect native messaging bridge invocation spawned by browser
	if bridge.IsNativeMessagingInvocation(os.Args) {
		if err := bridge.RunNativeMessagingBridge(); err != nil {
			fmt.Fprintf(os.Stderr, "Bridge error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(os.Args) > 1 {
		subcommand := os.Args[1]
		switch subcommand {
		case "setup":
			openBrowser := false
			chromeID := ""
			args := os.Args[2:]
			for i := 0; i < len(args); i++ {
				arg := args[i]
				if arg == "--open" || arg == "-o" {
					openBrowser = true
				} else if (arg == "--chrome-id" || arg == "-c") && i+1 < len(args) {
					chromeID = args[i+1]
					i++
				} else if strings.HasPrefix(arg, "--chrome-id=") {
					chromeID = strings.TrimPrefix(arg, "--chrome-id=")
				}
			}
			if err := setup.Run("", openBrowser, chromeID); err != nil {
				fmt.Fprintf(os.Stderr, "Error during setup: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	if versionFlagShort || versionFlagLong {
		fmt.Printf("tabwrangler %s\n", Version)
		os.Exit(0)
	}

	if helpFlagShort || helpFlagLong {
		printUsage()
		os.Exit(0)
	}

	initialBrowser := types.BrowserAll
	switch strings.ToLower(strings.TrimSpace(browserFlag)) {
	case "chrome":
		initialBrowser = types.BrowserChrome
	case "firefox":
		initialBrowser = types.BrowserFirefox
	case "safari":
		initialBrowser = types.BrowserSafari
	default:
		initialBrowser = types.BrowserAll
	}

	// Ensure native messaging host manifests are registered on local system
	if !mockFlag {
		setup.EnsureRegistered()
	}

	// Initialize collector
	coll := collector.NewCollector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var bridgeServer *bridge.Server
	if mockFlag {
		coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserChrome))
		coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserFirefox))
		coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserSafari))
	} else {
		// Start IPC bridge server for Chrome & Firefox
		bridgeServer = bridge.NewServer()
		if err := bridgeServer.Start(ctx); err != nil {
			// If IPC socket fails, log warning but continue
			fmt.Fprintf(os.Stderr, "Notice: local bridge socket unavailable: %v\n", err)
		}

		coll.RegisterAdapter(bridge.NewExtensionAdapter(types.BrowserChrome, bridgeServer))
		coll.RegisterAdapter(bridge.NewExtensionAdapter(types.BrowserFirefox, bridgeServer))
		coll.RegisterAdapter(safari.NewAdapter())
	}

	model := tui.NewModel(coll, initialBrowser, dryRunFlag)
	p := tea.NewProgram(model)

	if bridgeServer != nil {
		bridgeServer.SetOnRegister(func(b types.BrowserType) {
			p.Send(tui.TriggerRefreshMsg{})
		})
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
