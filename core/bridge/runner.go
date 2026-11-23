// Package bridge provides the Native Messaging stdio bridge execution runner.
package bridge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// RunNativeMessagingBridge connects stdin/stdout Native Messaging to the local IPC socket.
func RunNativeMessagingBridge() error {
	sockPath := SocketPath()
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return fmt.Errorf("failed connecting to TabWrangler IPC socket at %s: %w", sockPath, err)
	}
	defer conn.Close()

	// Goroutine 1: Read from local IPC socket and write to browser extension stdout
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			_ = WriteNativeMessage(os.Stdout, line)
		}
	}()

	// Goroutine 2: Read from browser extension stdin and write to local IPC socket
	for {
		payload, err := ReadNativeMessage(os.Stdin)
		if err != nil {
			break
		}

		// Ensure newline delimited for the IPC socket
		var js json.RawMessage
		if err := json.Unmarshal(payload, &js); err == nil {
			_, _ = conn.Write(append(payload, '\n'))
		}
	}

	return nil
}
