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
	"runtime"
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
	token, err := bridge.ReadToken(testSock)
	if err != nil {
		t.Fatalf("Failed reading bridge token: %v", err)
	}
	reg := fmt.Sprintf(`{"action":"register","browser":"firefox","token":%q}`, token) + "\n"
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

func TestUnmarshalFloatingLastAccessed(t *testing.T) {
	rawJSON := `{"id":"req-1","action":"list_tabs","success":true,"tabs":[{"id":1664087683,"windowId":1664087682,"title":"OneTab","url":"https://example.com","active":false,"pinned":true,"discarded":false,"lastAccessed":1793751482852.47,"incognito":false,"status":"complete"}]}`

	var resp bridge.Response
	if err := json.Unmarshal([]byte(rawJSON), &resp); err != nil {
		t.Fatalf("failed unmarshaling response with floating point lastAccessed: %v", err)
	}

	if len(resp.Tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(resp.Tabs))
	}

	if resp.Tabs[0].LastAccessed != 1793751482852.47 {
		t.Errorf("expected lastAccessed 1793751482852.47, got %v", resp.Tabs[0].LastAccessed)
	}
}

func TestBridgeServerRejectsMissingOrWrongToken(t *testing.T) {
	testSock := filepath.Join(os.TempDir(), fmt.Sprintf("tabwrangler-test-%d.sock", time.Now().UnixNano()))
	server := bridge.NewServerWithPath(testSock)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed starting bridge server: %v", err)
	}
	defer server.Close()

	info, err := os.Stat(bridge.TokenPath(testSock))
	if err != nil {
		t.Fatalf("Expected token file: %v", err)
	}
	sockInfo, err := os.Stat(testSock)
	if err != nil {
		t.Fatalf("Expected socket file: %v", err)
	}
	// Windows reports 0666 for every file; access there is controlled by the per-user directory.
	if runtime.GOOS != "windows" {
		if info.Mode().Perm() != 0600 {
			t.Errorf("Expected token file mode 0600, got %v", info.Mode().Perm())
		}
		if sockInfo.Mode().Perm() != 0600 {
			t.Errorf("Expected socket mode 0600, got %v", sockInfo.Mode().Perm())
		}
	}

	for _, reg := range []string{
		`{"action":"register","browser":"firefox"}`,
		`{"action":"register","browser":"firefox","token":"wrong"}`,
	} {
		conn, err := net.Dial("unix", testSock)
		if err != nil {
			t.Fatalf("Failed dialing bridge socket: %v", err)
		}
		_, _ = conn.Write([]byte(reg + "\n"))
		time.Sleep(50 * time.Millisecond)
		if server.IsBrowserConnected(types.BrowserFirefox) {
			t.Errorf("Registration %q must be rejected", reg)
		}
		conn.Close()
	}

	server.Close()
	if _, err := os.Stat(bridge.TokenPath(testSock)); !os.IsNotExist(err) {
		t.Error("Expected token file to be removed on Close")
	}
}

func TestBridgeRunnerInjectsTokenIntoRegistration(t *testing.T) {
	sockPath := bridge.SocketPath()
	if _, err := os.Stat(sockPath); err == nil {
		t.Skip("default bridge socket in use by a running TabWrangler")
	}
	server := bridge.NewServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed starting bridge server: %v", err)
	}
	defer server.Close()

	stdinR, stdinW, _ := os.Pipe()
	stdoutR, stdoutW, _ := os.Pipe()
	origIn, origOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = stdinR, stdoutW
	defer func() { os.Stdin, os.Stdout = origIn, origOut }()
	defer stdoutR.Close()

	done := make(chan struct{})
	go func() {
		_ = bridge.RunNativeMessagingBridge()
		close(done)
	}()

	// The extension's registration message carries no token; the bridge adds it.
	if err := bridge.WriteNativeMessage(stdinW, []byte(`{"action":"register","browser":"chrome"}`)); err != nil {
		t.Fatalf("Failed writing native message: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for !server.IsBrowserConnected(types.BrowserChrome) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !server.IsBrowserConnected(types.BrowserChrome) {
		t.Fatal("Expected Chrome to register through the bridge runner")
	}
	stdinW.Close()
	<-done
}
