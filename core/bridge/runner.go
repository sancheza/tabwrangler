// Package bridge provides the Native Messaging stdio bridge execution runner.
package bridge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// IsNativeMessagingInvocation checks command line arguments to detect if TabWrangler
// was spawned by a browser as a native messaging host.
//
// Firefox spawns: <binary> <path-to-native-manifest.json> <extension-id>
// Chrome spawns:  <binary> chrome-extension://<extension-id>/
// Manual test:    <binary> bridge
func IsNativeMessagingInvocation(args []string) bool {
	if len(args) < 2 {
		return false
	}
	arg1 := args[1]
	if arg1 == "bridge" {
		return true
	}
	if strings.HasPrefix(arg1, "chrome-extension://") {
		return true
	}
	if strings.HasSuffix(arg1, ".json") && (strings.Contains(arg1, "tabwrangler") || strings.Contains(arg1, "NativeMessagingHosts")) {
		return true
	}
	if strings.Contains(arg1, "com.tabwrangler.bridge") {
		return true
	}
	return false
}

// RunNativeMessagingBridge connects stdin/stdout Native Messaging to the local IPC socket.
func RunNativeMessagingBridge() error {
	sockPath := SocketPath()

	// Retry connecting briefly in case the TUI process is in the middle of socket setup
	var conn net.Conn
	var err error
	for i := 0; i < 5; i++ {
		conn, err = net.Dial("unix", sockPath)
		if err == nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("failed connecting to TabWrangler IPC socket at %s: %w", sockPath, err)
	}
	defer conn.Close()

	done := make(chan struct{})

	// Goroutine 1: Read from local IPC socket and write to browser extension stdout
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(conn)
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 16*1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			if err := WriteNativeMessage(os.Stdout, line); err != nil {
				return
			}
		}
	}()

	// Goroutine 2: Read from browser extension stdin and write to local IPC socket
	go func() {
		for {
			payload, err := ReadNativeMessage(os.Stdin)
			if err != nil {
				_ = conn.Close()
				return
			}

			// Ensure newline delimited for the IPC socket
			var js json.RawMessage
			if err := json.Unmarshal(payload, &js); err == nil {
				if _, err := conn.Write(append(payload, '\n')); err != nil {
					return
				}
			}
		}
	}()

	<-done
	return nil
}
