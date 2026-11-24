package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "target")

	subDir := filepath.Join(src, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed creating subdir: %v", err)
	}

	testFile := filepath.Join(subDir, "test.txt")
	expectedContent := "tabwrangler copy test"
	if err := os.WriteFile(testFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("failed writing test file: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	dstFile := filepath.Join(dst, "sub", "test.txt")
	gotContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed reading copied file: %v", err)
	}
	if string(gotContent) != expectedContent {
		t.Fatalf("expected content %q, got %q", expectedContent, string(gotContent))
	}
}

func TestInstallBinaryReplacesLockedFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions are not enforced for root")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "new.exe")
	dst := filepath.Join(dir, "installed.exe")
	if err := os.WriteFile(src, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	// A read-only file stands in for a running program that cannot be overwritten in place.
	if err := os.WriteFile(dst, []byte("old"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := InstallBinary(src, dst); err != nil {
		t.Fatalf("InstallBinary failed: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "new" {
		t.Fatalf("installed file = %q, %v; want %q", got, err, "new")
	}
	if _, err := os.Stat(dst + ".old"); !os.IsNotExist(err) {
		t.Errorf("expected the renamed old copy to be removed, stat err = %v", err)
	}
}

func TestInstallBinaryMissingSource(t *testing.T) {
	dir := t.TempDir()
	if err := InstallBinary(filepath.Join(dir, "nope.exe"), filepath.Join(dir, "dst.exe")); err == nil {
		t.Fatal("expected an error for a missing source file")
	}
}

func TestNextStepsFirstInstallShowsLoadUnpacked(t *testing.T) {
	got := nextSteps(nil, "")
	if !strings.Contains(got, "Load unpacked") || !strings.Contains(got, "Load Temporary Add-on") {
		t.Errorf("first-install steps should explain loading the extension:\n%s", got)
	}
	// An extension folder with nothing loaded yet is still a first install.
	if got := nextSteps(nil, filepath.Join("x", "extension")); !strings.Contains(got, "Load unpacked") {
		t.Errorf("nothing loaded yet should show the loading steps:\n%s", got)
	}
}

func TestNextStepsUpdateSaysReload(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "TabWrangler", "extension")
	loaded := []installedExtension{{ID: "abc", Path: dir}, {ID: "abc", Path: dir}}
	got := nextSteps(loaded, dir)
	if strings.Contains(got, "Load unpacked") {
		t.Errorf("an update must not ask to load the extension again:\n%s", got)
	}
	for _, want := range []string{"Reload", "chrome://extensions", "about:debugging", dir} {
		if !strings.Contains(got, want) {
			t.Errorf("update steps should mention %q:\n%s", want, got)
		}
	}
}

func TestNextStepsUpdateWhenLoadedFromAnotherFolder(t *testing.T) {
	base := t.TempDir()
	installed := filepath.Join(base, "TabWrangler", "extension")
	elsewhere := filepath.Join(base, "Downloads", "tabwrangler", "extension")
	got := nextSteps([]installedExtension{{ID: "abc", Path: elsewhere}}, installed)
	for _, want := range []string{elsewhere, installed, "different folder", "Copy the contents"} {
		if !strings.Contains(got, want) {
			t.Errorf("steps should mention %q:\n%s", want, got)
		}
	}
}

func TestDiscoverInstalledExtensionsReadsPathFromPreferences(t *testing.T) {
	local := t.TempDir()
	profile := filepath.Join(local, "Google", "Chrome", "User Data", "Profile 1")
	if err := os.MkdirAll(profile, 0755); err != nil {
		t.Fatal(err)
	}
	extPath := filepath.Join(local, "TabWrangler", "extension")
	prefs := fmt.Sprintf(`{"extensions":{"settings":{
		"aaaa":{"manifest":{"name":"TabWrangler Companion"},"path":%q},
		"bbbb":{"manifest":{"name":"Some Other Extension"},"path":"C:\\other"}}}}`, extPath)
	if err := os.WriteFile(filepath.Join(profile, "Preferences"), []byte(prefs), 0600); err != nil {
		t.Fatal(err)
	}
	got := discoverInstalledExtensions(local)
	if len(got) != 1 || got[0].ID != "aaaa" || got[0].Path != extPath {
		t.Fatalf("discoverInstalledExtensions = %+v; want the TabWrangler extension with its path", got)
	}
}
