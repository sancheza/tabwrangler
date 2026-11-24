package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sancheza/tabwrangler/core/bridge"
	"github.com/sancheza/tabwrangler/core/types"
)

func TestNativeMessagingReadWrite(t *testing.T) {
	var buf bytes.Buffer
	msg := []byte(`{"action":"list_tabs","id":"req-1"}`)

	// Write native message
	err := bridge.WriteNativeMessage(&buf, msg)
	if err != nil {
		t.Fatalf("WriteNativeMessage returned error: %v", err)
	}

	// Verify header length prefix
	var length uint32
	headerReader := bytes.NewReader(buf.Bytes()[:4])
	if err := binary.Read(headerReader, binary.LittleEndian, &length); err != nil {
		t.Fatalf("Failed reading length header: %v", err)
	}
	if int(length) != len(msg) {
		t.Fatalf("Expected header length %d, got %d", len(msg), length)
	}

	// Read native message
	readPayload, err := bridge.ReadNativeMessage(&buf)
	if err != nil {
		t.Fatalf("ReadNativeMessage returned error: %v", err)
	}

	if !bytes.Equal(readPayload, msg) {
		t.Fatalf("Expected payload %s, got %s", string(msg), string(readPayload))
	}
}

func TestNativeMessageSizeLimit(t *testing.T) {
	oversized := make([]byte, 17*1024*1024) // 17MB
	var buf bytes.Buffer
	err := bridge.WriteNativeMessage(&buf, oversized)
	if err == nil {
		t.Fatal("Expected error writing oversized message (> 16MB), got nil")
	}
}

func TestIsNativeMessagingInvocation(t *testing.T) {
	cases := []struct {
		args     []string
		expected bool
	}{
		{[]string{"tabwrangler"}, false},
		{[]string{"tabwrangler", "--browser", "chrome"}, false},
		{[]string{"tabwrangler", "setup"}, false},
		{[]string{"tabwrangler", "bridge"}, true},
		{[]string{"tabwrangler", "chrome-extension://abcdefghijklmnop/"}, true},
		{[]string{"tabwrangler", "--parent-window=12345", "chrome-extension://ambpjofjffekeccmjjeaadoeieemmlhc/"}, true},
		{[]string{"tabwrangler", "chrome-extension://ambpjofjffekeccmjjeaadoeieemmlhc/", "--parent-window=0"}, true},
		{[]string{"tabwrangler", "/path/to/com.tabwrangler.bridge.json", "companion@tabwrangler.local"}, true},
		{[]string{"tabwrangler", "/Users/foo/Library/Application Support/Mozilla/NativeMessagingHosts/com.tabwrangler.bridge.json"}, true},
	}

	for _, c := range cases {
		got := bridge.IsNativeMessagingInvocation(c.args)
		if got != c.expected {
			t.Errorf("IsNativeMessagingInvocation(%v) = %v, expected %v", c.args, got, c.expected)
		}
	}
}

func TestBridgeServerRegistrationAndDispatch(t *testing.T) {
	testSock := filepath.Join(os.TempDir(), fmt.Sprintf("tabwrangler-test-%d.sock", time.Now().UnixNano()))
	server := bridge.NewServerWithPath(testSock)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed starting bridge server: %v", err)
	}
	defer server.Close()

	if server.IsBrowserConnected(types.BrowserFirefox) {
		t.Fatal("Expected Firefox to not be connected initially")
	}

	// Dial custom IPC socket as simulated bridge client
	conn, err := net.Dial("unix", testSock)
	if err != nil {
		t.Fatalf("Failed dialing bridge socket: %v", err)
	}
	defer conn.Close()

	// Send registration message
	reg := `{"action":"register","browser":"firefox"}` + "\n"
	if _, err := conn.Write([]byte(reg)); err != nil {
		t.Fatalf("Failed sending registration: %v", err)
	}

	// Wait for registration to process
	time.Sleep(50 * time.Millisecond)

	if !server.IsBrowserConnected(types.BrowserFirefox) {
		t.Fatal("Expected Firefox to be connected after registration")
	}

	// Simulate response handling in a goroutine
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Bytes()
			var req bridge.Request
			if err := json.Unmarshal(line, &req); err != nil {
				continue
			}
			if req.Action == bridge.ActionFocusTab {
				resp := bridge.Response{
					ID:      req.ID,
					Action:  req.Action,
					Success: true,
				}
				data, _ := json.Marshal(resp)
				_, _ = conn.Write(append(data, '\n'))
			}
		}
	}()

	// Use adapter to FocusTab
	adapter := bridge.NewExtensionAdapter(types.BrowserFirefox, server)
	tab := types.Tab{
		TabID:    "101",
		WindowID: "1",
		Title:    "Test Tab",
		URL:      "https://example.com",
		Browser:  types.BrowserFirefox,
	}

	focusCtx, focusCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer focusCancel()
	if err := adapter.FocusTab(focusCtx, tab); err != nil {
		t.Fatalf("FocusTab failed: %v", err)
	}
}
