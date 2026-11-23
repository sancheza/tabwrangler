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
)

const HostName = "com.tabwrangler.bridge"

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
func Run(binaryPath string) error {
	if binaryPath == "" {
		var err error
		binaryPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed resolving executable path: %w", err)
		}
	}
	binaryPath, _ = filepath.Abs(binaryPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed locating user home directory: %w", err)
	}

	fmt.Printf("[INFO] Configuring Native Messaging host for TabWrangler...\n")
	fmt.Printf("[INFO] Binary location: %s\n", binaryPath)

	if runtime.GOOS == "darwin" {
		chromeDir := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
		firefoxDir := filepath.Join(homeDir, "Library", "Application Support", "Mozilla", "NativeMessagingHosts")

		if err := writeManifests(chromeDir, firefoxDir, binaryPath); err != nil {
			return err
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
	} else if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")
		chromeDir := filepath.Join(localAppData, "Google", "Chrome", "User Data", "NativeMessagingHosts")
		firefoxDir := filepath.Join(appData, "Mozilla", "NativeMessagingHosts")

		if err := writeManifests(chromeDir, firefoxDir, binaryPath); err != nil {
			return err
		}
	} else {
		// Linux
		chromeDir := filepath.Join(homeDir, ".config", "google-chrome", "NativeMessagingHosts")
		firefoxDir := filepath.Join(homeDir, ".mozilla", "native-messaging-hosts")

		if err := writeManifests(chromeDir, firefoxDir, binaryPath); err != nil {
			return err
		}
	}

	fmt.Printf("\033[32m[OK]\033[0m Setup complete. Load the extension from the 'extension/' directory into your browsers.\n")
	return nil
}

func writeManifests(chromeDir, firefoxDir, binaryPath string) error {
	if err := os.MkdirAll(chromeDir, 0755); err != nil {
		return fmt.Errorf("failed creating Chrome NativeMessagingHosts dir: %w", err)
	}
	if err := os.MkdirAll(firefoxDir, 0755); err != nil {
		return fmt.Errorf("failed creating Firefox NativeMessagingHosts dir: %w", err)
	}

	chromeMF := chromeManifest{
		Name:        HostName,
		Description: "TabWrangler Native Messaging Host Bridge",
		Path:        binaryPath,
		Type:        "stdio",
		AllowedOrigins: []string{
			"chrome-extension://*/",
		},
	}

	chromePath := filepath.Join(chromeDir, HostName+".json")
	cBytes, _ := json.MarshalIndent(chromeMF, "", "  ")
	if err := os.WriteFile(chromePath, cBytes, 0644); err != nil {
		return fmt.Errorf("failed writing Chrome manifest to %s: %w", chromePath, err)
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
		return fmt.Errorf("failed writing Firefox manifest to %s: %w", firefoxPath, err)
	}
	fmt.Printf("\033[32m[OK]\033[0m Firefox native messaging manifest installed: %s\n", firefoxPath)

	return nil
}
