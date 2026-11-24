// Package bridge provides IPC server and client communication over local domain sockets.
package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sancheza/tabwrangler/core/types"
)

const WindowsTCPAddr = "127.0.0.1:42823"

// SocketPath returns the OS-specific local IPC socket or loopback address.
func SocketPath() string {
	if runtime.GOOS == "windows" {
		return WindowsTCPAddr
	}
	uid := os.Getuid()
	return fmt.Sprintf("/tmp/tabwrangler-%d.sock", uid)
}

// Server handles incoming bridge connections from browser native messaging processes.
type Server struct {
	mu          sync.RWMutex
	sockPath    string
	listener    net.Listener
	connections map[types.BrowserType]net.Conn
	reqCounter  uint64
	pending     map[string]chan *Response
	pendingMu   sync.Mutex
	closed      bool
	startErr    error
	onRegister  func(types.BrowserType)
	token       string
}

// NewServer initializes a new IPC bridge server.
func NewServer() *Server {
	return NewServerWithPath(SocketPath())
}

// NewServerWithPath initializes an IPC bridge server with a custom socket path (useful for testing).
func NewServerWithPath(sockPath string) *Server {
	return &Server{
		sockPath:    sockPath,
		connections: make(map[types.BrowserType]net.Conn),
		pending:     make(map[string]chan *Response),
	}
}

// isTCPAddr tests if the specified target string represents a loopback TCP address rather than a filesystem socket path.
func isTCPAddr(addr string) bool {
	return strings.HasPrefix(addr, "127.0.0.1:") || strings.HasPrefix(addr, "localhost:") || strings.HasPrefix(addr, "0.0.0.0:") || strings.HasPrefix(addr, ":")
}

// Start begins listening on the local IPC socket or TCP port.
func (s *Server) Start(ctx context.Context) error {
	sockPath := s.sockPath
	if sockPath == "" {
		sockPath = SocketPath()
		s.sockPath = sockPath
	}

	network := "unix"
	if isTCPAddr(sockPath) {
		network = "tcp"
	} else {
		_ = os.Remove(sockPath)
	}

	token, err := newToken()
	if err != nil {
		return err
	}
	if err := writeToken(TokenPath(sockPath), token); err != nil {
		return err
	}
	s.mu.Lock()
	s.token = token
	s.mu.Unlock()

	// Create the socket with owner-only permissions from the start.
	restore := func() {}
	if network == "unix" {
		restore = restrictUmask()
	}
	ln, err := net.Listen(network, sockPath)
	restore()
	if err != nil {
		s.mu.Lock()
		s.startErr = err
		s.mu.Unlock()
		Log("Server.Start failed to listen on %s (%s): %v", network, sockPath, err)
		return fmt.Errorf("failed to listen on %s (%s): %w", network, sockPath, err)
	}
	Log("Server.Start listening on %s (%s)", network, sockPath)

	if network == "unix" && runtime.GOOS != "windows" {
		// Enforce strict 0600 permissions
		if err := os.Chmod(sockPath, 0600); err != nil {
			_ = ln.Close()
			return fmt.Errorf("failed setting socket permissions: %w", err)
		}
	}

	s.listener = ln

	go func() {
		<-ctx.Done()
		s.Close()
	}()

	go s.acceptLoop()
	return nil
}

// Close shuts down the server and cleans up the socket.
func (s *Server) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	if s.listener != nil {
		_ = s.listener.Close()
	}
	for _, conn := range s.connections {
		_ = conn.Close()
	}
	s.mu.Unlock()

	if runtime.GOOS != "windows" {
		_ = os.Remove(s.sockPath)
	}
	_ = os.Remove(TokenPath(s.sockPath))
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	Log("Server.handleConn accepted connection from %s", conn.RemoteAddr())
	reader := bufio.NewReader(conn)

	// First message should be registration
	line, err := reader.ReadBytes('\n')
	if err != nil {
		Log("Server.handleConn failed reading registration: %v", err)
		_ = conn.Close()
		return
	}

	var regMsg Request
	if err := json.Unmarshal(line, &regMsg); err != nil || regMsg.Action != ActionRegister {
		Log("Server.handleConn invalid registration payload (len=%d): %v", len(line), err)
		_ = conn.Close()
		return
	}
	s.mu.RLock()
	expectedToken := s.token
	s.mu.RUnlock()
	if !tokenMatches(expectedToken, regMsg.Token) {
		Log("Server.handleConn rejected registration: missing or invalid token")
		_ = conn.Close()
		return
	}

	browser := regMsg.Browser
	Log("Server.handleConn registered browser: %s", browser)
	s.mu.Lock()
	if old, exists := s.connections[browser]; exists {
		_ = old.Close()
	}
	s.connections[browser] = conn
	cb := s.onRegister
	s.mu.Unlock()
	if cb == nil {
		Log("Server.handleConn: %s registered but no register callback is set yet", browser)
	}

	if cb != nil {
		go cb(browser)
	}

	defer func() {
		Log("Server.handleConn disconnected browser: %s", browser)
		s.mu.Lock()
		if s.connections[browser] == conn {
			delete(s.connections, browser)
		}
		s.mu.Unlock()
		_ = conn.Close()
	}()

	// Loop reading responses from bridge
	for {
		respLine, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}

		var resp Response
		if err := json.Unmarshal(respLine, &resp); err != nil {
			Log("Server.handleConn failed unmarshaling bridge response (len=%d): %v", len(respLine), err)
			continue
		}

		s.pendingMu.Lock()
		ch, exists := s.pending[resp.ID]
		if exists {
			delete(s.pending, resp.ID)
			s.pendingMu.Unlock()
			ch <- &resp
		} else {
			s.pendingMu.Unlock()
		}
	}
}

// SetOnRegister sets a callback invoked when a browser client establishes a registration handshake.
func (s *Server) SetOnRegister(fn func(types.BrowserType)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onRegister = fn
}

// IsBrowserConnected reports whether a live connection exists for the specified browser.
func (s *Server) IsBrowserConnected(b types.BrowserType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.connections[b]
	return ok
}

// StartError returns any error encountered during server socket listen.
func (s *Server) StartError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startErr
}

// SendRequest transmits a command to the specified browser extension and awaits response.
func (s *Server) SendRequest(ctx context.Context, b types.BrowserType, req Request) (*Response, error) {
	s.mu.RLock()
	conn, ok := s.connections[b]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("bridge not connected for browser %s", b)
	}

	reqID := fmt.Sprintf("req-%d", atomic.AddUint64(&s.reqCounter, 1))
	req.ID = reqID
	req.Browser = b

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling request: %w", err)
	}
	data = append(data, '\n')

	respChan := make(chan *Response, 1)
	s.pendingMu.Lock()
	s.pending[reqID] = respChan
	s.pendingMu.Unlock()

	sent := time.Now()
	Log("SendRequest %s %s: sending", reqID, req.Action)
	if _, err := conn.Write(data); err != nil {
		s.pendingMu.Lock()
		delete(s.pending, reqID)
		s.pendingMu.Unlock()
		Log("SendRequest %s: write failed: %v", reqID, err)
		return nil, fmt.Errorf("failed writing to bridge socket: %w", err)
	}

	select {
	case <-ctx.Done():
		Log("SendRequest %s: cancelled after %s: %v", reqID, time.Since(sent).Round(time.Millisecond), ctx.Err())
		s.pendingMu.Lock()
		delete(s.pending, reqID)
		s.pendingMu.Unlock()
		return nil, ctx.Err()
	case resp := <-respChan:
		Log("SendRequest %s: response after %s, success=%v", reqID, time.Since(sent).Round(time.Millisecond), resp.Success)
		return resp, nil
	case <-time.After(3 * time.Second):
		Log("SendRequest %s: timed out after 3s waiting for the browser", reqID)
		s.pendingMu.Lock()
		delete(s.pending, reqID)
		s.pendingMu.Unlock()
		return nil, fmt.Errorf("timeout awaiting bridge response for %s", b)
	}
}
