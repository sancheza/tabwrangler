package tests

import (
	"testing"

	"github.com/sancheza/tabwrangler/core/browser"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestActivateBrowserApplication(t *testing.T) {
	// Should not panic on unsupported browsers
	browser.ActivateBrowserApplication(types.BrowserAll)

	// Test activation calls for supported browsers without error
	browser.ActivateBrowserApplication(types.BrowserFirefox)
	browser.ActivateBrowserApplication(types.BrowserChrome)
	browser.ActivateBrowserApplication(types.BrowserSafari)
}
