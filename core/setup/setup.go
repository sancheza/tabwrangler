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

type chromeManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

type firefoxManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedExtensions []string `json:"allowed_extensions"`
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
	if runtime.GOOS == "windows" && filepath.Ext(binaryPath) == "" {
		binaryPath += ".exe"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed locating user home directory: %w", err)
	}

	fmt.Printf("[INFO] Configuring Native Messaging host for TabWrangler...\n")
	fmt.Printf("[INFO] Binary location: %s\n", binaryPath)

	if runtime.GOOS == "darwin" {
		chromeDir := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
		firefoxDir := filepath.Join(homeDir, "Library", "Application Support", "Mozilla", "NativeMessagingHosts")

		if _, _, err := writeManifests(chromeDir, firefoxDir, binaryPath, customChromeID); err != nil {
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

		chromeDir := filepath.Join(localAppData, "TabWrangler", "NativeMessagingHosts")
		firefoxDir := filepath.Join(appData, "Mozilla", "NativeMessagingHosts")

		chromePath, firefoxPath, err := writeManifests(chromeDir, firefoxDir, binaryPath, customChromeID)
		if err != nil {
			return err
		}

		// Register in Windows Registry under HKCU
		fmt.Printf("[INFO] Registering Native Messaging hosts in Windows Registry...\n")
		registerWindowsRegistry(`Google\Chrome`, chromePath)
		registerWindowsRegistry(`Microsoft\Edge`, chromePath)
		registerWindowsRegistry(`Mozilla`, firefoxPath)

		// Copy extension directory to Windows clipboard if clip is available
		candidatePaths := []string{
			filepath.Join(filepath.Dir(binaryPath), "extension"),
			"extension",
		}
		for _, cand := range candidatePaths {
			abs, _ := filepath.Abs(cand)
			if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
				clip := exec.Command("clip")
				clip.Stdin = strings.NewReader(abs)
				if clip.Run() == nil {
					fmt.Printf("\033[32m[OK]\033[0m Copied extension directory to Windows clipboard: %s\n", abs)
				}
				break
			}
		}
	} else {
		// Linux
		chromeDir := filepath.Join(homeDir, ".config", "google-chrome", "NativeMessagingHosts")
		firefoxDir := filepath.Join(homeDir, ".mozilla", "native-messaging-hosts")

		if _, _, err := writeManifests(chromeDir, firefoxDir, binaryPath, customChromeID); err != nil {
			return err
		}
	}

	fmt.Printf("\033[32m[OK]\033[0m Setup complete. Load the extension from the 'extension/' directory into your browsers.\n")
	return nil
}

func writeManifests(chromeDir, firefoxDir, binaryPath, customChromeID string) (string, string, error) {
	if err := os.MkdirAll(chromeDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed creating Chrome NativeMessagingHosts dir: %w", err)
	}
	if err := os.MkdirAll(firefoxDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed creating Firefox NativeMessagingHosts dir: %w", err)
	}

	allowedOrigins := []string{
		"chrome-extension://" + DefaultChromeExtensionID + "/",
	}
	if customChromeID != "" && customChromeID != DefaultChromeExtensionID {
		allowedOrigins = append(allowedOrigins, "chrome-extension://"+customChromeID+"/")
	}

	chromeMF := chromeManifest{
		Name:           HostName,
		Description:    "TabWrangler Native Messaging Host Bridge",
		Path:           binaryPath,
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

func registerWindowsRegistry(browserKey, manifestPath string) {
	cmd := exec.Command("reg", "add", `HKCU\Software\`+browserKey+`\NativeMessagingHosts\`+HostName, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("\033[38;5;214m[WARNING]\033[0m Failed adding %s registry key: %v (%s)\n", browserKey, err, strings.TrimSpace(string(out)))
	} else {
		fmt.Printf("\033[32m[OK]\033[0m Registered %s in Windows Registry: HKCU\\Software\\%s\\NativeMessagingHosts\\%s\n", browserKey, browserKey, HostName)
	}
}
