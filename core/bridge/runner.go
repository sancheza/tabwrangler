// Package bridge provides the Native Messaging stdio bridge execution runner.
package bridge

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"strings"
	"sync"
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

// connectPollInterval is how often the bridge host looks for a running TabWrangler.
const connectPollInterval = 200 * time.Millisecond

// RunNativeMessagingBridge connects stdin/stdout Native Messaging to the local IPC socket.
//
// The host stays alive while the browser keeps its end open. When TabWrangler quits, the host waits
// and registers again when TabWrangler next starts, so a relaunch finds the browser already connected.
func RunNativeMessagingBridge() error {
	sockPath := SocketPath()
	network := "unix"
	if isTCPAddr(sockPath) {
		network = "tcp"
	}
	Log("RunNativeMessagingBridge started: sockPath=%s network=%s", sockPath, network)

	// Buffer messages from the browser while no TabWrangler is connected.
	inMessages := make(chan []byte, 100)
	stdinClosed := make(chan struct{})
	var regMu sync.Mutex
	var lastRegister []byte

	go func() {
		defer close(stdinClosed)
		for {
			payload, err := ReadNativeMessage(os.Stdin)
			if err != nil {
				Log("Browser stdin closed/error: %v", err)
				return
			}
			Log("Received native message on stdin (len=%d)", len(payload))
			if isRegister(payload) {
				regMu.Lock()
				lastRegister = append([]byte(nil), payload...)
				regMu.Unlock()
			}
			select {
			case inMessages <- payload:
			default:
				Log("Warning: inMessages channel full, dropping message")
			}
		}
	}()

	for first := true; ; first = false {
		conn, token := dialTabWrangler(network, sockPath, stdinClosed)
		if conn == nil {
			Log("Browser closed pipe before TUI connected, exiting bridge")
			return nil
		}
		Log("Bridge connected to local IPC server at %s", sockPath)

		// Messages buffered before this connection are answers to a TabWrangler that is gone.
		// The registration is replayed from the saved copy instead.
		for len(inMessages) > 0 {
			<-inMessages
		}
		regMu.Lock()
		reg := lastRegister
		regMu.Unlock()
		if reg != nil {
			if !first {
				Log("Bridge: registering again with the new TabWrangler")
			}
			if _, err := conn.Write(append(withToken(reg, token), '\n')); err != nil {
				Log("Error writing registration to IPC socket: %v", err)
				_ = conn.Close()
				continue
			}
		}

		if runSession(conn, token, inMessages, stdinClosed) {
			return nil
		}
		Log("Bridge: TabWrangler closed the connection; waiting for it to start again")
	}
}

// dialTabWrangler polls until it connects to a running TabWrangler or the browser closes its pipe.
// It returns a nil connection in the second case.
func dialTabWrangler(network, sockPath string, stdinClosed <-chan struct{}) (net.Conn, string) {
	for attempt := 1; ; attempt++ {
		began := time.Now()
		token, err := ReadToken(sockPath)
		var conn net.Conn
		if err == nil {
			conn, err = net.Dial(network, sockPath)
		}
		if err == nil {
			Log("Bridge: connect attempt %d succeeded after %s", attempt, time.Since(began).Round(time.Millisecond))
			return conn, token
		}
		if attempt <= 3 || attempt%300 == 0 {
			Log("Bridge: connect attempt %d failed after %s: %v", attempt, time.Since(began).Round(time.Millisecond), err)
		}
		select {
		case <-stdinClosed:
			return nil, ""
		case <-time.After(connectPollInterval):
		}
	}
}

// runSession forwards messages between the browser and one TabWrangler connection.
// It returns true when the browser closed its pipe and false when TabWrangler closed the connection.
func runSession(conn net.Conn, token string, inMessages chan []byte, stdinClosed <-chan struct{}) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(conn)
		scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
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

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case payload := <-inMessages:
				var js json.RawMessage
				if err := json.Unmarshal(payload, &js); err != nil {
					continue
				}
				payload = withToken(payload, token)
				Log("Forwarding stdin payload to TUI IPC socket (len=%d)", len(payload))
				if _, err := conn.Write(append(payload, '\n')); err != nil {
					Log("Error writing payload to IPC socket: %v", err)
					_ = conn.Close()
					return
				}
			case <-stdinClosed:
				for len(inMessages) > 0 {
					_, _ = conn.Write(append(withToken(<-inMessages, token), '\n'))
				}
				return
			case <-stop:
				return
			}
		}
	}()

	browserClosed := false
	select {
	case <-done:
	case <-stdinClosed:
		browserClosed = true
	}
	close(stop)
	wg.Wait()
	_ = conn.Close()
	return browserClosed
}

// isRegister reports whether a native message is the extension's registration.
func isRegister(payload []byte) bool {
	var msg struct {
		Action string `json:"action"`
	}
	return json.Unmarshal(payload, &msg) == nil && msg.Action == ActionRegister
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
