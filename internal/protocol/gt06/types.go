// Package gt06 implements decoders for the GT06 GPS protocol
package gt06

import "time"

// GT06Data represents the decoded data from a GT06 protocol packet
type GT06Data struct {
	Valid       bool
	GPSValid    bool
	Satellites  int
	Latitude    float64
	Longitude   float64
	Speed       float64
	Course      float64
	PowerLevel  int
	GSMSignal   int
	Alarm       string
	Timestamp   time.Time
	Status      map[string]interface{}
}

// Constants shared between decoder implementations
const (
	// Packet markers
	StartByte1 = 0x78
	StartByte2 = 0x78
	EndByte1   = 0x0D
	EndByte2   = 0x0A

	// Message types
	LoginMsg    = 0x01
	LocationMsg = 0x12
	StatusMsg   = 0x13
	AlarmMsg    = 0x16

	// Alarm types
	SosAlarm        = 0x01
	PowerCutAlarm   = 0x02
	VibrationAlarm  = 0x03
	FenceInAlarm    = 0x04
	FenceOutAlarm   = 0x05
	LowBatteryAlarm = 0x06
	OverspeedAlarm  = 0x07

	// Response types
	LoginResp    = 0x01
	LocationResp = 0x12
	AlarmResp    = 0x16
)

// Common errors
var (
	ErrInvalidHeader      = errors.New("invalid GT06 protocol header")
	ErrPacketTooShort     = errors.New("data too short for GT06 protocol")
	ErrInvalidChecksum    = errors.New("invalid checksum")
	ErrInvalidCoordinate  = errors.New("invalid BCD coordinate value")
	ErrInvalidTimestamp   = errors.New("invalid timestamp values")
	ErrInvalidLength      = errors.New("packet length mismatch")
	ErrInvalidMessageType = errors.New("unsupported message type")
	ErrMalformedPacket    = errors.New("malformed packet structure")
)
