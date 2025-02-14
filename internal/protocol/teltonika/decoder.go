// Package teltonika implements the Teltonika GPS tracker protocol decoder
// Protocol Information:
// The Teltonika protocol uses binary format with IEEE 754 encoding:
//   - Position data uses double-precision floating-point
//   - Optional fields like altitude, speed use appropriate numeric types
//   - All values are in big-endian byte order
//
// Data Structure:
//   - Latitude:  8 bytes (IEEE 754 double)
//   - Longitude: 8 bytes (IEEE 754 double)
//   - Altitude:  4 bytes (optional, float32)
//   - Speed:     2 bytes (optional, uint16, km/h * 10)
//   - Course:    2 bytes (optional, uint16, degrees)
//
// For detailed protocol specification, see the Teltonika protocol documentation.

package teltonika

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"time"
	"tracking/internal/core/model"
)

// Common Teltonika errors
var (
	ErrPacketTooShort    = errors.New("data too short for Teltonika protocol")
	ErrInvalidCoordinate = errors.New("invalid coordinates")
	ErrInvalidValue      = errors.New("invalid field value")
	ErrMalformedPacket   = errors.New("malformed packet structure")
)

type Decoder struct {
	debug bool
}

func NewDecoder() *Decoder {
	return &Decoder{
		debug: false,
	}
}

// EnableDebug enables detailed logging for protocol parsing
func (d *Decoder) EnableDebug(enable bool) {
	d.debug = enable
}

// logDebug logs debug messages if debug mode is enabled
func (d *Decoder) logDebug(format string, v ...interface{}) {
	if d.debug {
		log.Printf("[Teltonika] "+format, v...)
	}
}

// logPacket logs packet details in hexadecimal format
func (d *Decoder) logPacket(data []byte, prefix string) {
	if !d.debug {
		return
	}

	var hexStr string
	for i, b := range data {
		if i > 0 && i%16 == 0 {
			hexStr += "\n        "
		}
		hexStr += fmt.Sprintf("%02x ", b)
	}
	d.logDebug("%s Packet [%d bytes]:\n        %s", prefix, len(data), hexStr)
}

type TeltonikaData struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	Speed     float64
	Course    float64
	Timestamp time.Time
	Valid     bool
	Status    map[string]interface{}
}

func (d *Decoder) Decode(data []byte) (*TeltonikaData, error) {
	d.logDebug("Starting packet decode...")
	d.logPacket(data, "Received")

	if len(data) < 16 {
		return nil, fmt.Errorf("%w: got %d bytes, need at least 16",
			ErrPacketTooShort, len(data))
	}

	reader := bytes.NewReader(data)
	result := &TeltonikaData{
		Timestamp: time.Now(),
		Valid:     true,
		Status:    make(map[string]interface{}),
	}

	// Read latitude (IEEE 754 double-precision)
	if err := binary.Read(reader, binary.BigEndian, &result.Latitude); err != nil {
		return nil, fmt.Errorf("failed to read latitude: %w", err)
	}

	// Read longitude (IEEE 754 double-precision)
	if err := binary.Read(reader, binary.BigEndian, &result.Longitude); err != nil {
		return nil, fmt.Errorf("failed to read longitude: %w", err)
	}

	// Validate coordinates
	if !isValidCoordinate(result.Latitude, result.Longitude) {
		return nil, fmt.Errorf("%w: lat=%.6f, lon=%.6f",
			ErrInvalidCoordinate, result.Latitude, result.Longitude)
	}

	// Read optional fields if available
	if reader.Len() >= 4 {
		var altitude float32
		if err := binary.Read(reader, binary.BigEndian, &altitude); err != nil {
			return nil, fmt.Errorf("failed to read altitude: %w", err)
		}
		result.Altitude = float64(altitude)
		result.Status["altitude"] = result.Altitude
	}

	if reader.Len() >= 2 {
		var speed uint16
		if err := binary.Read(reader, binary.BigEndian, &speed); err != nil {
			return nil, fmt.Errorf("failed to read speed: %w", err)
		}
		result.Speed = float64(speed) / 10.0 // Convert to km/h
		result.Status["speed"] = result.Speed
	}

	if reader.Len() >= 2 {
		var course uint16
		if err := binary.Read(reader, binary.BigEndian, &course); err != nil {
			return nil, fmt.Errorf("failed to read course: %w", err)
		}
		if course > 360 {
			return nil, fmt.Errorf("%w: invalid course value %d", ErrInvalidValue, course)
		}
		result.Course = float64(course)
		result.Status["course"] = result.Course
	}

	d.logDebug("Successfully decoded packet: %+v", result)
	return result, nil
}

func isValidCoordinate(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func (d *Decoder) ToPosition(deviceID string, data *TeltonikaData) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Altitude = data.Altitude
	position.Protocol = "teltonika"
	position.Timestamp = data.Timestamp

	// Copy all status fields
	position.Status = make(map[string]interface{})
	for k, v := range data.Status {
		position.Status[k] = v
	}

	return position
}