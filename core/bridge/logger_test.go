package bridge

import (
	"os"
	"strings"
	"testing"
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
