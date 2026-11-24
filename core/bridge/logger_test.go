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

	content := string(data)
	if !strings.Contains(content, "test log message: hello world") {
		t.Errorf("expected log content to contain message, got: %s", content)
	}
}
