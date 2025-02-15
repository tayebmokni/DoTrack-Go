// Package gt06 implements decoders for the GT06 GPS protocol
package gt06

import (
	"bytes"
	"errors"
	"fmt"
	"time"
)

// GT06Data represents the decoded data from a GT06 protocol packet
type GT06Data struct {
	Valid       bool
	GPSValid    bool
	Latitude    float64
	Longitude   float64
	Speed       float64
	Course      float64
	Timestamp   time.Time
	Satellites  int
	PowerLevel  int
	GSMSignal   int
	Alarm       string
	Status      map[string]interface{}
}

// Protocol constants
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

	// Minimum packet sizes
	MinPacketLength    = 15  // Minimum length for any valid packet
	MinLoginLength     = 15  // start(2) + len(1) + proto(1) + imei(8) + checksum(2) + end(2)
	MinLocationLength  = 26  // start(2) + len(1) + proto(1) + gps(18) + checksum(2) + end(2)
	MinStatusLength    = 13  // start(2) + len(1) + proto(1) + status(4) + checksum(2) + end(2)
	MinAlarmLength     = 27  // start(2) + len(1) + proto(1) + gps(18) + alarm(1) + checksum(2) + end(2)
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

// PacketHeader represents the common header structure for GT06 packets
type PacketHeader struct {
	Length    byte
	Protocol  byte
	TotalSize int
}

// Shared utility functions

// ParseTimestamp parses BCD encoded timestamp from GT06 data
func ParseTimestamp(reader *bytes.Reader) (time.Time, error) {
	var timeBytes [6]byte
	if _, err := reader.Read(timeBytes[:]); err != nil {
		return time.Time{}, err
	}

	year := 2000 + ((int(timeBytes[0])>>4)*10 + int(timeBytes[0]&0x0F))
	month := (int(timeBytes[1])>>4)*10 + int(timeBytes[1]&0x0F)
	day := (int(timeBytes[2])>>4)*10 + int(timeBytes[2]&0x0F)
	hour := (int(timeBytes[3])>>4)*10 + int(timeBytes[3]&0x0F)
	minute := (int(timeBytes[4])>>4)*10 + int(timeBytes[4]&0x0F)
	second := (int(timeBytes[5])>>4)*10 + int(timeBytes[5]&0x0F)

	if month < 1 || month > 12 || day < 1 || day > 31 ||
		hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, ErrInvalidTimestamp
	}

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC), nil
}

// BcdToFloat converts BCD encoded coordinates to float64
func BcdToFloat(bcd uint32) (float64, error) {
	degrees := float64(BcdToDec(byte(bcd>>24)))*10 +
		float64(BcdToDec(byte((bcd>>16)&0xFF)))/60 +
		float64(BcdToDec(byte((bcd>>8)&0xFF)))/3600
	return degrees, nil
}

// BcdToDec converts a BCD byte to decimal
func BcdToDec(b byte) int {
	return int(b>>4)*10 + int(b&0x0F)
}

// GetMessageTypeName returns a human-readable name for message types
func GetMessageTypeName(protocolNumber byte) string {
	switch protocolNumber {
	case LoginMsg:
		return "login"
	case LocationMsg:
		return "location"
	case StatusMsg:
		return "status"
	case AlarmMsg:
		return "alarm"
	default:
		return fmt.Sprintf("unknown_0x%02x", protocolNumber)
	}
}

// GetAlarmName returns a human-readable name for alarm types
func GetAlarmName(alarmType byte) string {
	switch alarmType {
	case SosAlarm:
		return "sos"
	case PowerCutAlarm:
		return "powerCut"
	case VibrationAlarm:
		return "vibration"
	case FenceInAlarm:
		return "geofenceEnter"
	case FenceOutAlarm:
		return "geofenceExit"
	case LowBatteryAlarm:
		return "lowBattery"
	case OverspeedAlarm:
		return "overspeed"
	default:
		return fmt.Sprintf("unknown_%02x", alarmType)
	}
}

// CalculateChecksum calculates the checksum for GT06 packets
func CalculateChecksum(data []byte) uint16 {
	var sum uint16
	for _, b := range data {
		sum ^= uint16(b)
	}
	return sum
}

// ValidateCoordinates checks if coordinates are within valid ranges
func ValidateCoordinates(lat, lon float64) error {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return fmt.Errorf("%w: lat=%.6f, lon=%.6f",
			ErrInvalidCoordinate, lat, lon)
	}
	return nil
}