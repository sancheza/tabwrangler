// Package bridge provides Native Messaging frame reading and writing conforming to
// Chrome and Firefox WebExtension standards.
package bridge

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ReadNativeMessage reads a 4-byte little-endian length prefix followed by JSON payload.
func ReadNativeMessage(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	// Safety cap at 16MB per Native Messaging spec
	if length > 16*1024*1024 {
		return nil, fmt.Errorf("native message payload too large: %d bytes", length)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("failed reading full native message payload: %w", err)
	}

	return buf, nil
}

// WriteNativeMessage writes a 4-byte little-endian length prefix followed by the payload.
func WriteNativeMessage(w io.Writer, payload []byte) error {
	if len(payload) > 16*1024*1024 {
		return errors.New("native message payload exceeds 16MB limit")
	}

	length := uint32(len(payload))
	if err := binary.Write(w, binary.LittleEndian, length); err != nil {
		return fmt.Errorf("failed writing native message length header: %w", err)
	}

	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("failed writing native message payload: %w", err)
	}

	return nil
}

// WriteNativeJSON serializes an object to JSON and writes it as a native message.
func WriteNativeJSON(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed marshaling JSON native message: %w", err)
	}
	return WriteNativeMessage(w, data)
}
