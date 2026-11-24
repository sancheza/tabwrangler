// Package setup installs Native Messaging manifests for Chrome and Firefox
// and verifies OS permissions.
package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	HostName                 = "com.tabwrangler.bridge"
	DefaultChromeExtensionID = "ambpjofjffekeccmjjeaadoeieemmlhc"
)

// chromeManifest represents the Native Messaging host JSON manifest schema for Chromium browsers.
type chromeManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

// firefoxManifest represents the Native Messaging host JSON manifest schema for Mozilla Firefox.
type firefoxManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedExtensions []string `json:"allowed_extensions"`
}

// EnsureRegistered checks if Native Messaging manifests are configured; if not, runs silent setup.
func EnsureRegistered() {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			homeDir, _ := os.UserHomeDir()
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		manifestPath := filepath.Join(localAppData, "TabWrangler", "NativeMessagingHosts", HostName+".json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			_ = Run("", false, "")
		}
	}
}

// Run executes the setup process, creating manifest files and verifying permissions.
func Run(binaryPath string, openBrowser bool, customChromeID string) error {
	if binaryPath == "" {
		var err error
		binaryPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed resolving executable path: %w", err)
		}
	}
	binaryPath, _ = filepath.Abs(binaryPath)
	var loaded []installedExtension // extensions a browser has already loaded
	var installedExtDir string      // where setup installed the extension files
	if runtime.GOOS == "windows" && filepath.Ext(binaryPath) == "" {
		binaryPath += ".exe"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed locating user home directory: %w", err)
	}

	fmt.Printf("[INFO] Configuring Native Messaging host for TabWrangler...\n")
	fmt.Printf("[INFO] Binary location: %s\n", binaryPath)

	allIDs := []string{DefaultChromeExtensionID}
	if customChromeID != "" && customChromeID != DefaultChromeExtensionID {
		allIDs = append(allIDs, customChromeID)
	}

	if runtime.GOOS == "darwin" {
		chromeDir := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
		firefoxDir := filepath.Join(homeDir, "Library", "Application Support", "Mozilla", "NativeMessagingHosts")

		if _, _, err := writeManifestsWithOrigins(chromeDir, firefoxDir, binaryPath, allIDs); err != nil {
			return err
		}

		// Copy manifest path to clipboard if pbcopy is available
		candidatePaths := []string{
			filepath.Join(filepath.Dir(binaryPath), "extension", "manifest.json"),
			"extension/manifest.json",
			filepath.Join(homeDir, "dev", "tabwrangler", "extension", "manifest.json"),
		}
		var manifestPath string
		for _, cand := range candidatePaths {
			abs, _ := filepath.Abs(cand)
			if _, err := os.Stat(abs); err == nil {
				manifestPath = abs
				break
			}
		}
		if manifestPath != "" {
			pb := exec.Command("pbcopy")
			pb.Stdin = strings.NewReader(manifestPath)
			if pb.Run() == nil {
				fmt.Printf("\033[32m[OK]\033[0m Copied extension manifest path to macOS clipboard: %s\n", manifestPath)
			}
		}

		// Verify Safari AppleScript automation permission
		fmt.Printf("[INFO] Verifying macOS Safari Automation permissions...\n")
		cmd := exec.Command("osascript", "-e", `tell application "Safari" to return name`)
		out, err := cmd.Output()
		if err != nil {
			fmt.Printf("\033[38;5;214m[WARNING]\033[0m Safari automation check failed or requires permission grant in macOS System Settings -> Privacy & Security -> Automation\n")
		} else {
			fmt.Printf("\033[32m[OK]\033[0m Safari automation verified: %s\n", string(out))
		}

		if openBrowser {
			fmt.Printf("[INFO] Opening Firefox add-on debugging page...\n")
			_ = exec.Command("open", "-a", "Firefox", "about:debugging").Run()
		}
	} else if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		appData := os.Getenv("APPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}

		// Establish permanent directory for TabWrangler
		tabWranglerDir := filepath.Join(localAppData, "TabWrangler")
		_ = os.MkdirAll(tabWranglerDir, 0755)

		// Copy binary into permanent AppData location so that moved/temp downloads don't break Chrome's path
		installedBinaryPath := filepath.Join(tabWranglerDir, "tabwrangler.exe")
		absBin, _ := filepath.Abs(binaryPath)
		if absBin != "" && absBin != installedBinaryPath {
			if err := InstallBinary(absBin, installedBinaryPath); err != nil {
				fmt.Printf("\033[38;5;214m[WARNING]\033[0m Could not install %s: %v\n", installedBinaryPath, err)
				fmt.Printf("          Close TabWrangler and your browsers, then run setup again.\n")
			} else {
				binaryPath = installedBinaryPath
				fmt.Printf("\033[32m[OK]\033[0m Installed persistent binary: %s\n", installedBinaryPath)
			}
		}

		// Locate and copy extension directory to persistent AppData location
		destExtensionDir := filepath.Join(tabWranglerDir, "extension")
		extCandidates := []string{
			filepath.Join(filepath.Dir(absBin), "extension"),
			"extension",
		}
		var srcExtensionDir string
		for _, cand := range extCandidates {
			candAbs, _ := filepath.Abs(cand)
			if fi, err := os.Stat(candAbs); err == nil && fi.IsDir() {
				srcExtensionDir = candAbs
				break
			}
		}
		if srcExtensionDir != "" && srcExtensionDir != destExtensionDir {
			if err := copyDir(srcExtensionDir, destExtensionDir); err == nil {
				fmt.Printf("\033[32m[OK]\033[0m Installed persistent extension directory: %s\n", destExtensionDir)
			}
		}

		chromeDir := filepath.Join(tabWranglerDir, "NativeMessagingHosts")
		firefoxDir := filepath.Join(appData, "Mozilla", "NativeMessagingHosts")

		// Scan for existing extension IDs in browser user profiles
		loaded = discoverInstalledExtensions(localAppData)
		if fi, err := os.Stat(destExtensionDir); err == nil && fi.IsDir() {
			installedExtDir = destExtensionDir
		}
		for _, dID := range extensionIDs(loaded) {
			found := false
			for _, existing := range allIDs {
				if existing == dID {
					found = true
					break
				}
			}
			if !found {
				allIDs = append(allIDs, dID)
				fmt.Printf("[INFO] Discovered existing installed TabWrangler extension ID: %s\n", dID)
			}
		}

		chromePath, firefoxPath, err := writeManifestsWithOrigins(chromeDir, firefoxDir, binaryPath, allIDs)
		if err != nil {
			return err
		}

		// Also write portable manifest directly into tabWranglerDir with "path": "tabwrangler.exe"
		// (guarantees zero spaces in binary path for Chromium process launcher)
		_, _, _ = writeManifestsWithOrigins(tabWranglerDir, firefoxDir, "tabwrangler.exe", allIDs)

		// Also place manifest directly into Chrome User Data NativeMessagingHosts if folder exists
		chromeUserDir := filepath.Join(localAppData, "Google", "Chrome", "User Data", "NativeMessagingHosts")
		_, _, _ = writeManifestsWithOrigins(chromeUserDir, firefoxDir, binaryPath, allIDs)

		// Register in Windows Registry under HKCU across both 32-bit and 64-bit views
		fmt.Printf("[INFO] Registering Native Messaging hosts in Windows Registry...\n")
		browserKeys := []string{
			`Google\Chrome`,
			`Microsoft\Edge`,
			`Chromium`,
			`BraveSoftware\Brave-Browser`,
		}
		for _, bk := range browserKeys {
			registerWindowsRegistry(bk, chromePath)
		}
		registerWindowsRegistry(`Mozilla`, firefoxPath)

		// Copy extension directory to Windows clipboard if clip is available
		var clipPath string
		if fi, err := os.Stat(destExtensionDir); err == nil && fi.IsDir() {
			clipPath = destExtensionDir
		} else if srcExtensionDir != "" {
			clipPath = srcExtensionDir
		}
		if clipPath != "" {
			clip := exec.Command("clip")
			clip.Stdin = strings.NewReader(clipPath)
			if clip.Run() == nil {
				fmt.Printf("\033[32m[OK]\033[0m Copied extension directory to Windows clipboard: %s\n", clipPath)
			}
		}
	} else {
		// Linux
		chromeDir := filepath.Join(homeDir, ".config", "google-chrome", "NativeMessagingHosts")
		firefoxDir := filepath.Join(homeDir, ".mozilla", "native-messaging-hosts")

		if _, _, err := writeManifestsWithOrigins(chromeDir, firefoxDir, binaryPath, allIDs); err != nil {
			return err
		}
	}

	fmt.Printf("\033[32m[OK]\033[0m Setup complete!\n\n")
	fmt.Print(nextSteps(loaded, installedExtDir))
	return nil
}

// writeManifests creates native messaging manifests for Chrome and Firefox using default and optional custom extension IDs.
func writeManifests(chromeDir, firefoxDir, binaryPath, customChromeID string) (string, string, error) {
	allIDs := []string{DefaultChromeExtensionID}
	if customChromeID != "" && customChromeID != DefaultChromeExtensionID {
		allIDs = append(allIDs, customChromeID)
	}
	return writeManifestsWithOrigins(chromeDir, firefoxDir, binaryPath, allIDs)
}

// writeManifestsWithOrigins serializes and writes Native Messaging JSON manifests to the specified directories
// configuring the allowed origins for Chrome and allowed extensions for Firefox.
func writeManifestsWithOrigins(chromeDir, firefoxDir, binaryPath string, extensionIDs []string) (string, string, error) {
	if err := os.MkdirAll(chromeDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed creating Chrome NativeMessagingHosts dir: %w", err)
	}
	if err := os.MkdirAll(firefoxDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed creating Firefox NativeMessagingHosts dir: %w", err)
	}

	var allowedOrigins []string
	for _, id := range extensionIDs {
		allowedOrigins = append(allowedOrigins, "chrome-extension://"+id+"/")
	}

	manifestBinPath := binaryPath
	if runtime.GOOS == "windows" {
		if rel, err := filepath.Rel(chromeDir, binaryPath); err == nil && !strings.HasPrefix(rel, "..\\..") {
			manifestBinPath = rel
		}
	}

	chromeMF := chromeManifest{
		Name:           HostName,
		Description:    "TabWrangler Native Messaging Host Bridge",
		Path:           manifestBinPath,
		Type:           "stdio",
		AllowedOrigins: allowedOrigins,
	}

	chromePath := filepath.Join(chromeDir, HostName+".json")
	cBytes, _ := json.MarshalIndent(chromeMF, "", "  ")
	if err := os.WriteFile(chromePath, cBytes, 0644); err != nil {
		return "", "", fmt.Errorf("failed writing Chrome manifest to %s: %w", chromePath, err)
	}
	fmt.Printf("\033[32m[OK]\033[0m Chrome native messaging manifest installed: %s\n", chromePath)

	ffMF := firefoxManifest{
		Name:        HostName,
		Description: "TabWrangler Native Messaging Host Bridge",
		Path:        binaryPath,
		Type:        "stdio",
		AllowedExtensions: []string{
			"companion@tabwrangler.local",
		},
	}

	firefoxPath := filepath.Join(firefoxDir, HostName+".json")
	fBytes, _ := json.MarshalIndent(ffMF, "", "  ")
	if err := os.WriteFile(firefoxPath, fBytes, 0644); err != nil {
		return "", "", fmt.Errorf("failed writing Firefox manifest to %s: %w", firefoxPath, err)
	}
	fmt.Printf("\033[32m[OK]\033[0m Firefox native messaging manifest installed: %s\n", firefoxPath)

	return chromePath, firefoxPath, nil
}

// registerWindowsRegistry registers the Native Messaging host JSON manifest path in the Windows Registry
// across standard HKCU, explicit 32-bit (/reg:32), explicit 64-bit (/reg:64), and Wow6432Node paths.
func registerWindowsRegistry(browserKey, manifestPath string) {
	// Register across standard, 32-bit (/reg:32), 64-bit (/reg:64), and Wow6432Node paths
	cmd1 := exec.Command("reg", "add", `HKCU\Software\`+browserKey+`\NativeMessagingHosts\`+HostName, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f")
	out, err := cmd1.CombinedOutput()

	cmd2 := exec.Command("reg", "add", `HKCU\Software\`+browserKey+`\NativeMessagingHosts\`+HostName, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f", "/reg:32")
	_ = cmd2.Run()

	cmd3 := exec.Command("reg", "add", `HKCU\Software\`+browserKey+`\NativeMessagingHosts\`+HostName, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f", "/reg:64")
	_ = cmd3.Run()

	cmd4 := exec.Command("reg", "add", `HKCU\Software\Wow6432Node\`+browserKey+`\NativeMessagingHosts\`+HostName, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f")
	_ = cmd4.Run()

	if err != nil {
		fmt.Printf("\033[38;5;214m[WARNING]\033[0m Failed adding %s registry key: %v (%s)\n", browserKey, err, strings.TrimSpace(string(out)))
	} else {
		fmt.Printf("\033[32m[OK]\033[0m Registered %s across 32-bit & 64-bit Windows Registry views\n", browserKey)
	}
}

// copyDir recursively copies a directory tree from src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

// installedExtension is a TabWrangler extension that a Chromium browser has loaded.
type installedExtension struct {
	ID   string
	Path string // folder the browser loaded it from
}

// discoverInstalledExtensions scans Chromium user profile directories (Chrome, Edge, Brave)
// across all profiles (Default, Profile 1, custom profile names with spaces) for loaded
// TabWrangler extensions, and reports each extension's ID and the folder it was loaded from.
func discoverInstalledExtensions(localAppData string) []installedExtension {
	var found []installedExtension
	searchRoots := []string{
		filepath.Join(localAppData, "Google", "Chrome", "User Data"),
		filepath.Join(localAppData, "Microsoft", "Edge", "User Data"),
		filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "User Data"),
	}
	for _, root := range searchRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			pf := filepath.Join(root, entry.Name(), "Preferences")
			data, err := os.ReadFile(pf)
			if err != nil {
				continue
			}
			var rootJSON struct {
				Extensions struct {
					Settings map[string]struct {
						Manifest struct {
							Name string `json:"name"`
						} `json:"manifest"`
						Path string `json:"path"`
					} `json:"settings"`
				} `json:"extensions"`
			}
			if err := json.Unmarshal(data, &rootJSON); err == nil {
				for extID, setting := range rootJSON.Extensions.Settings {
					if strings.Contains(strings.ToLower(setting.Manifest.Name), "tabwrangler") ||
						strings.Contains(strings.ToLower(setting.Path), "tabwrangler") {
						found = append(found, installedExtension{ID: extID, Path: setting.Path})
					}
				}
			}
		}
	}
	return found
}

// samePath reports whether two folder paths name the same folder (Windows paths ignore case).
func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// nextSteps returns the closing instructions of setup. When a browser has already loaded the
// extension, it says to reload it (or where the new files are, if the browser loaded another
// folder); otherwise it gives the first-time loading steps.
func nextSteps(installed []installedExtension, extDir string) string {
	var b strings.Builder
	if len(installed) == 0 || extDir == "" {
		b.WriteString("Next step: Load the companion extension into your browser:\n\n")
		b.WriteString("  • Chrome / Edge / Brave:\n")
		b.WriteString("      1. Navigate to chrome://extensions\n")
		b.WriteString("      2. Enable 'Developer mode' (top right)\n")
		b.WriteString("      3. Click 'Load unpacked' and select the extension/ directory\n\n")
		b.WriteString("  • Firefox:\n")
		b.WriteString("      1. Navigate to about:debugging#/runtime/this-firefox\n")
		b.WriteString("      2. Click 'Load Temporary Add-on...'\n")
		b.WriteString("      3. Select extension/manifest.json (Do NOT use 'Install Add-on From File' in about:addons)\n\n")
		b.WriteString("  • Safari (macOS):\n")
		b.WriteString("      Automated natively out-of-the-box. No extension required.\n\n")
		b.WriteString("Then run 'tabwrangler' to launch.\n")
		return b.String()
	}

	var elsewhere []string
	seen := map[string]bool{}
	for _, ext := range installed {
		if !samePath(ext.Path, extDir) && !seen[ext.Path] {
			seen[ext.Path] = true
			elsewhere = append(elsewhere, ext.Path)
		}
	}
	if len(elsewhere) == 0 {
		b.WriteString("The companion extension is already loaded from " + extDir + ".\n")
		b.WriteString("Reload it so the browser uses the new files:\n\n")
		b.WriteString("  • Chrome / Edge / Brave: open chrome://extensions and click the Reload button on 'TabWrangler Companion'.\n")
		b.WriteString("  • Firefox: open about:debugging#/runtime/this-firefox and click 'Reload' on 'TabWrangler Companion'.\n\n")
		b.WriteString("Press 'i' in TabWrangler to confirm the extension version.\n")
		return b.String()
	}
	b.WriteString("Setup installed the new extension files in " + extDir + ".\n")
	b.WriteString("Your browser loaded the extension from a different folder, which setup does not change:\n")
	for _, p := range elsewhere {
		b.WriteString("  • " + p + "\n")
	}
	b.WriteString("\nUse the new files in one of two ways:\n")
	b.WriteString("  1. Copy the contents of " + extDir + " over that folder, then click Reload on 'TabWrangler Companion' in chrome://extensions.\n")
	b.WriteString("  2. In chrome://extensions, remove 'TabWrangler Companion', click 'Load unpacked', and select " + extDir + ".\n")
	return b.String()
}

// InstallBinary copies the executable at src to dst, replacing any existing file.
//
// Windows refuses to overwrite a program that is running (the browser keeps TabWrangler's
// bridge host running), but allows renaming it. When the direct write fails, the running file
// is renamed aside and the new one is written under the original name.
func InstallBinary(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed reading %s: %w", src, err)
	}
	old := dst + ".old"
	_ = os.Remove(old) // left over from an earlier upgrade; fails harmlessly while that copy still runs
	if err := os.WriteFile(dst, data, 0755); err == nil {
		return nil
	}
	if err := os.Rename(dst, old); err != nil {
		return fmt.Errorf("failed replacing %s: %w", dst, err)
	}
	if err := os.WriteFile(dst, data, 0755); err != nil {
		_ = os.Rename(old, dst)
		return fmt.Errorf("failed writing %s: %w", dst, err)
	}
	_ = os.Remove(old)
	return nil
}

// extensionIDs lists the IDs of the given extensions.
func extensionIDs(exts []installedExtension) []string {
	var ids []string
	for _, e := range exts {
		ids = append(ids, e.ID)
	}
	return ids
}
