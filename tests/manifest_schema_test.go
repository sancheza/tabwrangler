package tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type manifestV3 struct {
	ManifestVersion int    `json:"manifest_version"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Key             string `json:"key"`
	Background      struct {
		ServiceWorker string   `json:"service_worker"`
		Scripts       []string `json:"scripts"`
	} `json:"background"`
	Icons                   map[string]string `json:"icons"`
	BrowserSpecificSettings struct {
		Gecko struct {
			ID string `json:"id"`
		} `json:"gecko"`
	} `json:"browser_specific_settings"`
}

// computeExtensionID computes the 32-character Chrome extension ID from an RSA public key base64 string.
func computeExtensionID(keyBase64 string) (string, error) {
	derBytes, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(derBytes)
	first16 := hash[:16]
	hexStr := hex.EncodeToString(first16)

	// Chrome uses an mpdecimal-like mapping: hex char '0'..'9' -> 'a'..'j', 'a'..'f' -> 'k'..'p'
	var sb strings.Builder
	for _, c := range hexStr {
		if c >= '0' && c <= '9' {
			sb.WriteRune(rune('a' + (c - '0')))
		} else if c >= 'a' && c <= 'f' {
			sb.WriteRune(rune('a' + 10 + (c - 'a')))
		}
	}
	return sb.String(), nil
}

func TestChromeManifestV3Compliance(t *testing.T) {
	manifestPath := filepath.Join("..", "extension", "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = filepath.Join("extension", "manifest.json")
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed reading extension manifest: %v", err)
	}
	baseDir := filepath.Dir(manifestPath)

	var m manifestV3
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("extension manifest is invalid JSON: %v", err)
	}

	// 1. Manifest version must be 3
	if m.ManifestVersion != 3 {
		t.Errorf("expected manifest_version 3, got %d", m.ManifestVersion)
	}

	// 2. MV3 background MUST NOT have scripts array (causes Chrome fatal load failure)
	if len(m.Background.Scripts) > 0 {
		t.Errorf("manifest background contains 'scripts' (%v); Chrome MV3 strictly forbids this and requires service_worker only", m.Background.Scripts)
	}

	// 3. Service worker must be defined and exist on disk
	if m.Background.ServiceWorker == "" {
		t.Errorf("manifest background.service_worker must be specified")
	} else {
		swPath := filepath.Join(baseDir, m.Background.ServiceWorker)
		if _, err := os.Stat(swPath); err != nil {
			t.Errorf("service worker file %s does not exist: %v", swPath, err)
		}
	}

	// 4. Key must exist and produce deterministic Chrome extension ID
	if m.Key == "" {
		t.Errorf("manifest must define 'key' so Chrome extension ID is fixed across all platforms")
	} else {
		extID, err := computeExtensionID(m.Key)
		if err != nil {
			t.Errorf("failed decoding manifest key: %v", err)
		} else {
			expectedID := "ambpjofjffekeccmjjeaadoeieemmlhc"
			if extID != expectedID {
				t.Errorf("extension ID mismatch: computed %s, expected %s", extID, expectedID)
			}
		}
	}

	// 5. Icons must exist on disk
	for size, iconRel := range m.Icons {
		iconPath := filepath.Join(baseDir, iconRel)
		if _, err := os.Stat(iconPath); err != nil {
			t.Errorf("icon (%s) at %s does not exist: %v", size, iconPath, err)
		}
	}

	// 6. Firefox gecko ID must be configured
	if m.BrowserSpecificSettings.Gecko.ID == "" {
		t.Errorf("gecko.id is required for Firefox companion add-on")
	}
}
