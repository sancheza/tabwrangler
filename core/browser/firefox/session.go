// Package firefox provides session recovery and native tab inspection for Firefox.
package firefox

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/pierrec/lz4/v4"
	"github.com/sancheza/tabwrangler/core/types"
)

var MozLz4Magic = []byte("mozLz40\x00")

type sessionStoreData struct {
	Windows []struct {
		Tabs []struct {
			Entries []struct {
				URL   string `json:"url"`
				Title string `json:"title"`
			} `json:"entries"`
			Index        int   `json:"index"`
			Pinned       bool  `json:"pinned"`
			Hidden       bool  `json:"hidden"`
			LastAccessed int64 `json:"lastAccessed"`
		} `json:"tabs"`
		Selected int `json:"selected"`
	} `json:"windows"`
}

// FindRecoveryFiles searches standard Firefox profile paths for active session recovery files.
func FindRecoveryFiles() []string {
	var candidates []string
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var basePattern string
	switch runtime.GOOS {
	case "darwin":
		basePattern = filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles", "*", "sessionstore-backups", "recovery.jsonlz4")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		basePattern = filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "*", "sessionstore-backups", "recovery.jsonlz4")
	default:
		// Linux
		basePattern = filepath.Join(home, ".mozilla", "firefox", "*", "sessionstore-backups", "recovery.jsonlz4")
	}

	matches, err := filepath.Glob(basePattern)
	if err == nil && len(matches) > 0 {
		candidates = append(candidates, matches...)
	}

	// Also check recovery.baklz4 if primary not found
	if len(candidates) == 0 {
		bakPattern := filepath.Dir(basePattern) + "/*.baklz4"
		bakMatches, _ := filepath.Glob(bakPattern)
		candidates = append(candidates, bakMatches...)
	}

	return candidates
}

// ReadSessionTabs parses the Firefox recovery.jsonlz4 file and maps it to Tab models.
func ReadSessionTabs(recoveryFilePath string) ([]types.Tab, error) {
	data, err := os.ReadFile(recoveryFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading firefox session file: %w", err)
	}

	if len(data) < 12 || !bytes.HasPrefix(data, MozLz4Magic) {
		return nil, fmt.Errorf("invalid mozLz4 magic header in %s", recoveryFilePath)
	}

	var uncompressedSize uint32
	if err := binary.Read(bytes.NewReader(data[8:12]), binary.LittleEndian, &uncompressedSize); err != nil {
		return nil, fmt.Errorf("failed reading uncompressed size: %w", err)
	}

	// Safety cap at 64MB for session store
	if uncompressedSize > 64*1024*1024 {
		return nil, fmt.Errorf("session store uncompressed size exceeds limit: %d", uncompressedSize)
	}

	compressed := data[12:]
	decompressed := make([]byte, uncompressedSize)
	n, err := lz4.UncompressBlock(compressed, decompressed)
	if err != nil {
		return nil, fmt.Errorf("failed decompressing firefox lz4 block: %w", err)
	}

	var store sessionStoreData
	if err := json.Unmarshal(decompressed[:n], &store); err != nil {
		return nil, fmt.Errorf("failed parsing firefox session json: %w", err)
	}

	var tabs []types.Tab
	for wIdx, win := range store.Windows {
		winID := strconv.Itoa(wIdx + 1)
		selectedIdx := win.Selected

		for tIdx, tab := range win.Tabs {
			tabID := strconv.Itoa(tIdx + 1)
			if tab.Hidden {
				// Tab is hidden (e.g. in another tab group or container)
				continue
			}

			var title, rawURL string
			if tab.Index > 0 && tab.Index <= len(tab.Entries) {
				entry := tab.Entries[tab.Index-1]
				title = entry.Title
				rawURL = entry.URL
			} else if len(tab.Entries) > 0 {
				entry := tab.Entries[len(tab.Entries)-1]
				title = entry.Title
				rawURL = entry.URL
			}

			if rawURL == "" {
				rawURL = "about:blank"
			}
			if title == "" {
				title = rawURL
			}

			domain := types.ExtractDomain(rawURL)
			isActive := ((tIdx + 1) == selectedIdx)

			tabs = append(tabs, types.Tab{
				ID:                 fmt.Sprintf("firefox:%s:%s", winID, tabID),
				Browser:            types.BrowserFirefox,
				Profile:            filepath.Base(filepath.Dir(filepath.Dir(recoveryFilePath))),
				WindowID:           winID,
				TabID:              tabID,
				Title:              title,
				URL:                rawURL,
				Domain:             domain,
				LastAccessedUnixMs: tab.LastAccessed,
				IsActive:           isActive,
				IsPinned:           tab.Pinned,
				IsDiscarded:        false,
				IsSpecial:          types.IsInternalOrSpecialURL(rawURL),
				Metrics: types.Metrics{
					CPUPercent: 0.0,
					RAMBytes:   0,
					Scope:      types.MetricScopeUnavailable,
				},
				Capabilities: types.Capabilities{
					CanFocus:   false, // Requires companion extension
					CanDiscard: false, // Requires companion extension
					CanClose:   false, // Requires companion extension
				},
			})
		}
	}

	return tabs, nil
}
