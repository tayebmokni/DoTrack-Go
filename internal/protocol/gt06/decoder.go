// Package gt06 implements the GT06 GPS protocol decoder
package gt06

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"time"
	"tracking/internal/core/model"
)

// Common GT06 errors
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
)

// Decoder implements the GT06 protocol decoder
type Decoder struct {
	debug bool
}

func NewDecoder() *Decoder {
	return &Decoder{debug: false}
}

func (d *Decoder) EnableDebug(enable bool) {
	d.debug = enable
}

func (d *Decoder) logDebug(format string, v ...interface{}) {
	if d.debug {
		log.Printf("[GT06] "+format, v...)
	}
}

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

// Decode implements the GT06 protocol decoder
func (d *Decoder) Decode(data []byte) (*GT06Data, error) {
	d.logDebug("Starting packet decode...")

	if len(data) < 7 {
		return nil, fmt.Errorf("%w: packet requires at least 7 bytes", ErrPacketTooShort)
	}

	// Validate start bytes
	if data[0] != StartByte1 || data[1] != StartByte2 {
		return nil, fmt.Errorf("%w: expected 0x%02x%02x, got 0x%02x%02x",
			ErrInvalidHeader, StartByte1, StartByte2, data[0], data[1])
	}

	// Get protocol number
	protocolNumber := data[3]
	d.logDebug("Protocol number: 0x%02x", protocolNumber)

	// Validate length based on protocol
	var minLength int
	switch protocolNumber {
	case LoginMsg:
		minLength = 15 // start(2) + len(1) + proto(1) + imei(8) + checksum(2) + end(2)
	case LocationMsg:
		minLength = 26 // start(2) + len(1) + proto(1) + gps(18) + checksum(2) + end(2)
	case StatusMsg:
		minLength = 13 // start(2) + len(1) + proto(1) + status(4) + checksum(2) + end(2)
	case AlarmMsg:
		minLength = 27 // start(2) + len(1) + proto(1) + gps(18) + alarm(1) + checksum(2) + end(2)
	default:
		return nil, fmt.Errorf("%w: 0x%02x", ErrInvalidMessageType, protocolNumber)
	}

	if len(data) < minLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d for protocol 0x%02x",
			ErrPacketTooShort, len(data), minLength, protocolNumber)
	}

	// Validate declared length
	declaredLen := int(data[2])
	expectedLen := len(data) - 5 // subtract start(2), len(1), checksum(2)
	if declaredLen != expectedLen {
		return nil, fmt.Errorf("%w: declared=%d, actual=%d",
			ErrInvalidLength, declaredLen, expectedLen)
	}

	// Calculate checksum position and validate
	checksumPos := len(data) - 4 // before end bytes
	calcChecksum := calculateChecksum(data[2:checksumPos])
	recvChecksum := uint16(data[checksumPos])<<8 | uint16(data[checksumPos+1])

	if calcChecksum != recvChecksum {
		return nil, fmt.Errorf("%w: calc=0x%04x, recv=0x%04x",
			ErrInvalidChecksum, calcChecksum, recvChecksum)
	}

	// Validate end bytes
	if data[len(data)-2] != EndByte1 || data[len(data)-1] != EndByte2 {
		return nil, fmt.Errorf("%w: invalid end bytes", ErrMalformedPacket)
	}

	// Extract content (after protocol byte, before checksum)
	content := data[4:checksumPos]
	d.logDebug("Content length: %d bytes", len(content))

	// Process based on protocol
	var result *GT06Data
	var err error

	switch protocolNumber {
	case LoginMsg:
		result, err = d.decodeLoginMessage(content)
	case LocationMsg:
		result, err = d.decodeLocationMessage(content)
	case StatusMsg:
		result, err = d.decodeStatusMessage(content)
	case AlarmMsg:
		result, err = d.decodeAlarmMessage(content)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode %s message: %w",
			getMessageTypeName(protocolNumber), err)
	}

	return result, nil
}

func (d *Decoder) decodeLocationMessage(data []byte) (*GT06Data, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("location message too short: got %d bytes, need 10", len(data))
	}

	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	// Parse GPS status
	statusByte := data[0]
	result.GPSValid = (statusByte&0x01) == 0x01
	result.Satellites = int((statusByte >> 2) & 0x0F)

	// Parse coordinates
	var err error
	if result.Latitude, err = bcdToFloat(binary.BigEndian.Uint32(data[1:5])); err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}
	if result.Longitude, err = bcdToFloat(binary.BigEndian.Uint32(data[5:9])); err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}

	// Parse speed and course
	result.Speed = float64(data[9])
	if len(data) >= 11 {
		result.Course = float64(binary.BigEndian.Uint16(data[10:12]))
	}

	// Parse timestamp if present
	if len(data) >= 16 {
		if ts, err := d.parseTimestamp(bytes.NewReader(data[12:18])); err == nil {
			result.Timestamp = ts
		}
	}

	return result, nil
}

func (d *Decoder) decodeStatusMessage(data []byte) (*GT06Data, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("%w: status message too short", ErrInvalidLength)
	}

	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	// First byte contains power and GSM signal levels
	statusByte := data[0]
	result.PowerLevel = int((statusByte >> 4) & 0x0F)
	result.GSMSignal = int(statusByte & 0x0F)

	// Validate power level (0-15)
	if result.PowerLevel > 15 {
		return nil, fmt.Errorf("%w: power level %d exceeds maximum of 15",
			ErrMalformedPacket, result.PowerLevel)
	}

	// Add power and signal levels to status
	result.Status["powerLevel"] = result.PowerLevel
	result.Status["gsmSignal"] = result.GSMSignal

	// Parse additional status flags if present
	if len(data) > 1 {
		result.Status["charging"] = (data[1]&0x20 != 0)
		result.Status["engineOn"] = (data[1]&0x40 != 0)
	}

	return result, nil
}

func (d *Decoder) decodeLoginMessage(data []byte) (*GT06Data, error) {
	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	if len(data) < 8 {
		return nil, fmt.Errorf("login message too short")
	}

	// Extract IMEI from payload
	result.Status["imei"] = fmt.Sprintf("%x", data[:8])

	return result, nil
}

func (d *Decoder) decodeAlarmMessage(data []byte) (*GT06Data, error) {
	// First decode location data
	locationData, err := d.decodeLocationMessage(data[:len(data)-1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode location data in alarm message: %w", err)
	}

	// Extract alarm type from last byte
	alarmType := data[len(data)-1]
	d.logDebug("Processing alarm type: 0x%02x", alarmType)

	switch alarmType {
	case SosAlarm:
		locationData.Alarm = "sos"
	case PowerCutAlarm:
		locationData.Alarm = "powerCut"
	case VibrationAlarm:
		locationData.Alarm = "vibration"
	case FenceInAlarm:
		locationData.Alarm = "geofenceEnter"
	case FenceOutAlarm:
		locationData.Alarm = "geofenceExit"
	case LowBatteryAlarm:
		locationData.Alarm = "lowBattery"
	case OverspeedAlarm:
		locationData.Alarm = "overspeed"
	default:
		// For unknown alarm types, use a consistent format
		locationData.Alarm = fmt.Sprintf("unknown_%02x", alarmType)
	}

	// Add alarm type to status map
	if locationData.Status == nil {
		locationData.Status = make(map[string]interface{})
	}
	locationData.Status["alarm"] = locationData.Alarm

	return locationData, nil
}

func (d *Decoder) parseTimestamp(reader *bytes.Reader) (time.Time, error) {
	var timeBytes [6]byte
	if _, err := reader.Read(timeBytes[:]); err != nil {
		return time.Time{}, err
	}

	// Extract each component from BCD bytes
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

func bcdToFloat(bcd uint32) (float64, error) {
	degrees := float64(bcdToDec(byte(bcd>>24)))*10 +
		float64(bcdToDec(byte((bcd>>16)&0xFF)))/60 +
		float64(bcdToDec(byte((bcd>>8)&0xFF)))/3600
	return degrees, nil
}

func bcdToDec(b byte) int {
	return int(b>>4)*10 + int(b&0x0F)
}

func (d *Decoder) ToPosition(deviceID string, data *GT06Data) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Valid = data.GPSValid
	position.Timestamp = data.Timestamp
	position.Protocol = "gt06"
	position.Satellites = data.Satellites

	position.Status = make(map[string]interface{})
	if data.PowerLevel > 0 {
		position.Status["powerLevel"] = data.PowerLevel
	}
	if data.GSMSignal > 0 {
		position.Status["gsmSignal"] = data.GSMSignal
	}
	if data.Alarm != "" {
		position.Status["alarm"] = data.Alarm
	}

	// Add remaining status fields
	for k, v := range data.Status {
		if _, exists := position.Status[k]; !exists {
			position.Status[k] = v
		}
	}

	return position
}

func getMessageTypeName(protocolNumber byte) string {
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

func calculateChecksum(data []byte) uint16 {
	var sum uint16
	for _, b := range data {
		sum ^= uint16(b)
	}
	return sum
}

func (d *Decoder) GenerateResponse(msgType uint8, deviceID string) []byte {
	switch msgType {
	case LoginMsg:
		return d.generateLoginResponse(deviceID)
	case LocationMsg:
		return d.generateLocationResponse()
	case AlarmMsg:
		return d.generateAlarmResponse()
	default:
		return d.generateLocationResponse() // Default to location response
	}
}

func (d *Decoder) generateLoginResponse(deviceID string) []byte {
	// Login response format:
	// Start(2) + PackLen(1) + ProtocolNo(1) + DeviceID(n) + Time(2) + SerialNo(2) + Error(2) + CRC(2) + Stop(2)
	respLen := len(deviceID) + 10 // Add length of fixed fields
	resp := make([]byte, 0, respLen+5)

	// Start bytes
	resp = append(resp, StartByte1, StartByte2)

	// Packet length
	resp = append(resp, byte(respLen))

	// Protocol number (login response)
	resp = append(resp, LoginResp)

	// Device ID (copy from request)
	resp = append(resp, []byte(deviceID)...)

	// Current time (UTC)
	now := time.Now().UTC()
	resp = append(resp, byte(now.Hour()), byte(now.Minute()))

	// Serial number (0x0001)
	resp = append(resp, 0x00, 0x01)

	// Error code (0x0000 = success)
	resp = append(resp, 0x00, 0x00)

	// Calculate CRC
	crc := calculateCRC(resp[2:])
	resp = append(resp, byte(crc>>8), byte(crc))

	// End bytes
	resp = append(resp, EndByte1, EndByte2)

	return resp
}

func (d *Decoder) generateLocationResponse() []byte {
	resp := []byte{
		StartByte1, StartByte2, // Start bytes
		0x05,                   // Packet length
		LocationResp,           // Protocol number (location response)
		0x00, 0x01,            // Serial number
		0x00, 0x01,            // CRC
		EndByte1, EndByte2,    // End bytes
	}
	return resp
}

func (d *Decoder) generateAlarmResponse() []byte {
	resp := []byte{
		StartByte1, StartByte2, // Start bytes
		0x05,                   // Packet length
		AlarmResp,              // Protocol number (alarm response)
		0x00, 0x01,            // Serial number
		0x00, 0x01,            // CRC
		EndByte1, EndByte2,    // End bytes
	}
	return resp
}

func calculateCRC(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b)
	}
	return crc
}

type GT06Data struct {
	Latitude    float64
	Longitude   float64
	Speed       float64
	Course      float64
	Timestamp   time.Time
	Valid       bool
	GPSValid    bool
	Satellites  int
	PowerLevel  int
	GSMSignal   int
	Alarm       string
	Status      map[string]interface{}
}