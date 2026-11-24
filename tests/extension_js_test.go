package tests

import (
	"os/exec"
	"testing"
)

// TestExtensionDiscard runs the extension's JavaScript tests with Node when it is installed.
func TestExtensionDiscard(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	out, err := exec.Command(node, "extension/discard.test.js").CombinedOutput()
	if err != nil {
		t.Fatalf("extension tests failed: %v\n%s", err, out)
	}
}
