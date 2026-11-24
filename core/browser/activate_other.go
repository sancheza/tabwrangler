//go:build !darwin && !windows

// Package browser provides cross-platform window activation and host automation utilities.
package browser

import (
	"github.com/sancheza/tabwrangler/core/types"
)

// ActivateBrowserApplication raises the specified browser application to the very front on the host OS.
func ActivateBrowserApplication(b types.BrowserType) {
	// Non-darwin platforms can implement window manager activation here.
}
