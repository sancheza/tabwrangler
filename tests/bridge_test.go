package tests

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/sancheza/tabwrangler/core/bridge"
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
