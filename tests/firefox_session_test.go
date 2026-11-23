package tests

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pierrec/lz4/v4"
)

func TestFirefoxSession(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot find home")
	}

	globPattern := filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles", "*", "sessionstore-backups", "recovery.jsonlz4")
	matches, err := filepath.Glob(globPattern)
	if err != nil || len(matches) == 0 {
		t.Skip("No Firefox recovery file found")
	}

	target := matches[0]
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("Failed reading recovery file: %v", err)
	}

	magic := []byte("mozLz40\x00")
	if !bytes.HasPrefix(data, magic) {
		t.Fatalf("Missing mozLz40 magic header")
	}

	var uncompressedSize uint32
	if err := binary.Read(bytes.NewReader(data[8:12]), binary.LittleEndian, &uncompressedSize); err != nil {
		t.Fatalf("Failed reading size: %v", err)
	}

	compressed := data[12:]
	decompressed := make([]byte, uncompressedSize)
	n, err := lz4.UncompressBlock(compressed, decompressed)
	if err != nil {
		t.Fatalf("LZ4 uncompress failed (expected size %d): %v", uncompressedSize, err)
	}

	var session struct {
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

	if err := json.Unmarshal(decompressed[:n], &session); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	totalTabs := 0
	for wIdx, win := range session.Windows {
		totalTabs += len(win.Tabs)
		if wIdx == 0 && len(win.Tabs) > 0 {
			tab := win.Tabs[0]
			var title, u string
			if tab.Index > 0 && tab.Index <= len(tab.Entries) {
				title = tab.Entries[tab.Index-1].Title
				u = tab.Entries[tab.Index-1].URL
			}
			t.Logf("Sample Tab 0: title=%q, url=%q, pinned=%v, lastAccessed=%d", title, u, tab.Pinned, tab.LastAccessed)
		}
	}

	t.Logf("Successfully parsed Firefox sessionstore! Found %d open tabs across %d windows", totalTabs, len(session.Windows))
}
