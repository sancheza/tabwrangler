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

const Version = "1.0.0"

func printUsage() {
	helpText := `TabWrangler — Cross-Browser Tab Inspector & Control TUI

Usage:
  tabwrangler [flags]
  tabwrangler setup
  tabwrangler bridge

Flags:
  -v, --version          Print version and exit
  -h, --help             Show this help message and exit
      --browser string   Pre-filter to specific browser (all, chrome, firefox, safari) (default "all")
      --mock             Run with mock browser provider for testing and demonstration
      --dry-run          Preview actions without modifying or closing actual tabs

Commands:
  setup                  Install browser native messaging manifests and verify permissions
  bridge                 Execute native messaging bridge over stdio (called by browsers)

Examples:
  tabwrangler
  tabwrangler --browser chrome
  tabwrangler --mock
  tabwrangler setup
`
	fmt.Print(helpText)
}

func main() {
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

	if len(os.Args) > 1 {
		subcommand := os.Args[1]
		switch subcommand {
		case "setup":
			openBrowser := false
			for _, arg := range os.Args[2:] {
				if arg == "--open" || arg == "-o" {
					openBrowser = true
				}
			}
			if err := setup.Run("", openBrowser); err != nil {
				fmt.Fprintf(os.Stderr, "Error during setup: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		case "bridge":
			if err := bridge.RunNativeMessagingBridge(); err != nil {
				fmt.Fprintf(os.Stderr, "Bridge error: %v\n", err)
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

	// Initialize collector
	coll := collector.NewCollector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if mockFlag {
		coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserChrome))
		coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserFirefox))
		coll.RegisterAdapter(collector.NewMockAdapter(types.BrowserSafari))
	} else {
		// Start IPC bridge server for Chrome & Firefox
		bridgeServer := bridge.NewServer()
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

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
