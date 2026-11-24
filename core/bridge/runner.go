// Package bridge provides the Native Messaging stdio bridge execution runner.
package bridge

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"strings"
	"time"
)

// IsNativeMessagingInvocation checks command line arguments to detect if TabWrangler
// was spawned by a browser as a native messaging host.
//
// Firefox spawns: <binary> <path-to-native-manifest.json> <extension-id>
// Chrome spawns:  <binary> chrome-extension://<extension-id>/ [--parent-window=<hwnd>]
// Manual test:    <binary> bridge
func IsNativeMessagingInvocation(args []string) bool {
	if len(args) < 2 {
		return false
	}
	for _, arg := range args[1:] {
		if arg == "bridge" {
			return true
		}
		if strings.HasPrefix(arg, "chrome-extension://") {
			return true
		}
		if strings.HasPrefix(arg, "--parent-window") {
			return true
		}
		if strings.HasSuffix(arg, ".json") && (strings.Contains(arg, "tabwrangler") || strings.Contains(arg, "NativeMessagingHosts")) {
			return true
		}
		if strings.Contains(arg, "com.tabwrangler.bridge") {
			return true
		}
	}
	return false
}

// RunNativeMessagingBridge connects stdin/stdout Native Messaging to the local IPC socket.
func RunNativeMessagingBridge() error {
	sockPath := SocketPath()
	network := "unix"
	if isTCPAddr(sockPath) {
		network = "tcp"
	}
	Log("RunNativeMessagingBridge started: sockPath=%s network=%s", sockPath, network)

	// Buffer messages from browser stdin while waiting for TUI IPC connection
	inMessages := make(chan []byte, 100)
	stdinClosed := make(chan struct{})

	// Goroutine: Read from browser extension stdin immediately
	go func() {
		defer close(stdinClosed)
		for {
			payload, err := ReadNativeMessage(os.Stdin)
			if err != nil {
				Log("Browser stdin closed/error: %v", err)
				return
			}
			Log("Received native message on stdin (len=%d)", len(payload))
			select {
			case inMessages <- payload:
			default:
				Log("Warning: inMessages channel full, dropping message")
			}
		}
	}()

	// Connect to local TabWrangler IPC server (wait patiently if TUI is launching)
	var conn net.Conn
	var err error
	var token string
	attempt := 0
	for {
		attempt++
		began := time.Now()
		token, err = ReadToken(sockPath)
		if err == nil {
			conn, err = net.Dial(network, sockPath)
		}
		if err != nil && (attempt <= 5 || attempt%20 == 0) {
			Log("Bridge: connect attempt %d failed after %s: %v", attempt, time.Since(began).Round(time.Millisecond), err)
		}
		if err == nil {
			Log("Bridge: connect attempt %d succeeded after %s", attempt, time.Since(began).Round(time.Millisecond))
			Log("Bridge connected to local IPC server at %s", sockPath)
			break
		}
		select {
		case <-stdinClosed:
			Log("Browser closed pipe before TUI connected, exiting bridge")
			return nil
		case <-time.After(500 * time.Millisecond):
		}
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
			Log("Forwarding response from TUI to browser stdout (len=%d)", len(line))
			if err := WriteNativeMessage(os.Stdout, line); err != nil {
				Log("Error writing native message to stdout: %v", err)
				return
			}
		}
	}()

	// Goroutine 2: Flush buffered messages, then forward incoming stdin messages to local IPC socket
	go func() {
		for {
			select {
			case payload, ok := <-inMessages:
				if !ok {
					_ = conn.Close()
					return
				}
				// Ensure newline delimited for the IPC socket
				var js json.RawMessage
				if err := json.Unmarshal(payload, &js); err == nil {
					payload = withToken(payload, token)
					Log("Forwarding stdin payload to TUI IPC socket (len=%d)", len(payload))
					if _, err := conn.Write(append(payload, '\n')); err != nil {
						Log("Error writing payload to IPC socket: %v", err)
						return
					}
				}
			case <-stdinClosed:
				// Drain any remaining inMessages
				for len(inMessages) > 0 {
					payload := withToken(<-inMessages, token)
					_, _ = conn.Write(append(payload, '\n'))
				}
				_ = conn.Close()
				return
			}
		}
	}()

	<-done
	return nil
}

// withToken adds the per-run token to a registration message. Other messages pass through unchanged.
func withToken(payload []byte, token string) []byte {
	var msg map[string]any
	if err := json.Unmarshal(payload, &msg); err != nil || msg["action"] != ActionRegister {
		return payload
	}
	Log("Bridge: adding the token to the register message")
	msg["token"] = token
	out, err := json.Marshal(msg)
	if err != nil {
		return payload
	}
	return out
}
