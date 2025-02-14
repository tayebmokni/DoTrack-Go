```go
package test

import (
	"net"
	"testing"
	"time"
)

func TestProtocolHandling(t *testing.T) {
	// Connect to TCP server
	conn, err := net.Dial("tcp", "localhost:5023")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Test GT06 protocol
	gt06LoginPacket := []byte{
		0x78, 0x78, // Start bytes
		0x0D,       // Length
		0x01,       // Protocol (login)
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, // Device ID
		0x00, 0x01, // Serial number
		0x00, 0x01, // Checksum
		0x0D, 0x0A, // End bytes
	}

	// Send login packet
	_, err = conn.Write(gt06LoginPacket)
	if err != nil {
		t.Fatalf("Failed to send GT06 login packet: %v", err)
	}

	// Read response
	response := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response format
	if n < 4 || response[0] != 0x78 || response[1] != 0x78 {
		t.Errorf("Invalid GT06 response format")
	}
}
```
