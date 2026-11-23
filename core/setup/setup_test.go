package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteManifests(t *testing.T) {
	tempDir := t.TempDir()
	chromeDir := filepath.Join(tempDir, "chrome")
	firefoxDir := filepath.Join(tempDir, "firefox")
	binPath := filepath.Join(tempDir, "tabwrangler")

	chromePath, ffPath, err := writeManifests(chromeDir, firefoxDir, binPath, "")
	if err != nil {
		t.Fatalf("writeManifests failed: %v", err)
	}

	// Verify Chrome manifest
	cBytes, err := os.ReadFile(chromePath)
	if err != nil {
		t.Fatalf("failed reading Chrome manifest: %v", err)
	}
	var cMF chromeManifest
	if err := json.Unmarshal(cBytes, &cMF); err != nil {
		t.Fatalf("invalid Chrome manifest JSON: %v", err)
	}
	if cMF.Name != HostName {
		t.Errorf("expected Chrome host name %q, got %q", HostName, cMF.Name)
	}
	if len(cMF.AllowedOrigins) != 1 || cMF.AllowedOrigins[0] != "chrome-extension://"+DefaultChromeExtensionID+"/" {
		t.Errorf("unexpected allowed origins: %v", cMF.AllowedOrigins)
	}

	// Verify Firefox manifest
	fBytes, err := os.ReadFile(ffPath)
	if err != nil {
		t.Fatalf("failed reading Firefox manifest: %v", err)
	}
	var fMF firefoxManifest
	if err := json.Unmarshal(fBytes, &fMF); err != nil {
		t.Fatalf("invalid Firefox manifest JSON: %v", err)
	}
	if fMF.Name != HostName {
		t.Errorf("expected Firefox host name %q, got %q", HostName, fMF.Name)
	}
	if len(fMF.AllowedExtensions) != 1 || fMF.AllowedExtensions[0] != "companion@tabwrangler.local" {
		t.Errorf("unexpected allowed extensions: %v", fMF.AllowedExtensions)
	}
}

func TestWriteManifestsWithCustomID(t *testing.T) {
	tempDir := t.TempDir()
	chromeDir := filepath.Join(tempDir, "chrome")
	firefoxDir := filepath.Join(tempDir, "firefox")
	binPath := filepath.Join(tempDir, "tabwrangler")

	customID := "abcdefghijklmnopabcdefghijklmnop"
	chromePath, _, err := writeManifests(chromeDir, firefoxDir, binPath, customID)
	if err != nil {
		t.Fatalf("writeManifests failed: %v", err)
	}

	cBytes, err := os.ReadFile(chromePath)
	if err != nil {
		t.Fatalf("failed reading Chrome manifest: %v", err)
	}
	var cMF chromeManifest
	if err := json.Unmarshal(cBytes, &cMF); err != nil {
		t.Fatalf("invalid Chrome manifest JSON: %v", err)
	}

	if len(cMF.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %d", len(cMF.AllowedOrigins))
	}
	if cMF.AllowedOrigins[0] != "chrome-extension://"+DefaultChromeExtensionID+"/" {
		t.Errorf("expected first origin %s, got %s", DefaultChromeExtensionID, cMF.AllowedOrigins[0])
	}
	if cMF.AllowedOrigins[1] != "chrome-extension://"+customID+"/" {
		t.Errorf("expected second origin %s, got %s", customID, cMF.AllowedOrigins[1])
	}
}
