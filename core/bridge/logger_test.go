package bridge

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestBridgeLogging(t *testing.T) {
	tempLog := t.TempDir() + "/test.log"
	logPath = tempLog

	Log("test log message: %s", "hello world")

	data, err := os.ReadFile(tempLog)
	if err != nil {
		t.Fatalf("failed reading log file: %v", err)
	}

	if info, err := os.Stat(tempLog); err == nil && info.Mode().Perm() != 0600 && os.PathSeparator == '/' {
		t.Errorf("expected log mode 0600, got %v", info.Mode().Perm())
	}

	content := string(data)
	if !strings.Contains(content, "test log message: hello world") {
		t.Errorf("expected log content to contain message, got: %s", content)
	}
}

func TestLogRotatesAtSizeCap(t *testing.T) {
	logPath = t.TempDir() + "/rotate.log"
	big := make([]byte, maxLogBytes)
	if err := os.WriteFile(logPath, big, 0600); err != nil {
		t.Fatal(err)
	}

	Log("after rotation")

	rotated, err := os.Stat(logPath + ".1")
	if err != nil || rotated.Size() != maxLogBytes {
		t.Fatalf("expected %d-byte rotated file, got %v, %v", maxLogBytes, rotated, err)
	}
	data, _ := os.ReadFile(logPath)
	if !strings.Contains(string(data), "after rotation") || len(data) > 200 {
		t.Errorf("expected a fresh log holding only the new line, got %d bytes", len(data))
	}
}

func TestPruneRotatedRemovesOldFileOnly(t *testing.T) {
	path := t.TempDir() + "/prune.log"
	rotated := path + ".1"
	if err := os.WriteFile(rotated, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}

	pruneRotated(path)
	if _, err := os.Stat(rotated); err != nil {
		t.Fatalf("a fresh rotated file must be kept: %v", err)
	}

	old := time.Now().Add(-maxRotatedAge - time.Hour)
	if err := os.Chtimes(rotated, old, old); err != nil {
		t.Fatal(err)
	}
	pruneRotated(path)
	if _, err := os.Stat(rotated); !os.IsNotExist(err) {
		t.Errorf("expected the old rotated file to be deleted, got %v", err)
	}
}
