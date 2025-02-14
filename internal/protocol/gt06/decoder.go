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
		log.Printf("[GT06] "+format, v...)
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

// GT06 protocol constants
const (
	startByte1 = 0x78
	startByte2 = 0x78
	minLength  = 10 // Minimum length: start(2) + length(1) + protocol(1) + checksum(2) + end(2)
	endByte1   = 0x0D
	endByte2   = 0x0A

	// Command types
	loginMsg    = 0x01
	locationMsg = 0x12
	statusMsg   = 0x13
	alarmMsg    = 0x16

	// Response types
	loginResp    = 0x05
	locationResp = 0x13
	alarmResp    = 0x15

	// Alarm types
	sosAlarm        = 0x01
	powerCutAlarm   = 0x02
	vibrationAlarm  = 0x04
	fenceInAlarm    = 0x10
	fenceOutAlarm   = 0x11
	lowBatteryAlarm = 0x20
	overspeedAlarm  = 0x40
)

// Fix packet length calculation and checksum validation
func (d *Decoder) Decode(data []byte) (*GT06Data, error) {
	d.logDebug("Starting packet decode...")
	d.logPacket(data, "Received")

	// Check minimum length
	if len(data) < minLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d",
			ErrPacketTooShort, len(data), minLength)
	}

	// Validate start bytes
	if data[0] != startByte1 || data[1] != startByte2 {
		return nil, fmt.Errorf("%w: expected 0x%02x%02x, got 0x%02x%02x",
			ErrInvalidHeader, startByte1, startByte2, data[0], data[1])
	}

	// Get packet length (includes protocol number and payload, excludes length byte itself)
	packetLength := int(data[2])
	d.logDebug("Packet length: %d", packetLength)

	// Calculate total expected length
	expectedLength := 2 + 1 + packetLength + 2 + 2 // start(2) + len(1) + payload(n) + crc(2) + end(2)
	if len(data) != expectedLength {
		return nil, fmt.Errorf("%w: payload=%d, total=%d, expected=%d",
			ErrInvalidLength, packetLength, len(data), expectedLength)
	}

	// Get protocol number
	protocolNumber := data[3]
	d.logDebug("Protocol number: 0x%02x", protocolNumber)

	// Validate protocol number
	switch protocolNumber {
	case loginMsg, locationMsg, statusMsg, alarmMsg:
		// Valid protocol numbers
	default:
		return nil, fmt.Errorf("%w: 0x%02x", ErrInvalidMessageType, protocolNumber)
	}

	// Extract payload (excluding start, length, and protocol)
	payloadStart := 4 // After start(2) + len(1) + protocol(1)
	payloadEnd := len(data) - 4 // Before crc(2) + end(2)
	payload := data[payloadStart:payloadEnd]

	// Calculate and verify checksum
	calculatedChecksum := calculateChecksum(data[2:payloadEnd]) // Include length byte in checksum
	packetChecksum := uint16(data[payloadEnd])<<8 | uint16(data[payloadEnd+1])
	if calculatedChecksum != packetChecksum {
		return nil, fmt.Errorf("%w: calculated=0x%04x, received=0x%04x",
			ErrInvalidChecksum, calculatedChecksum, packetChecksum)
	}

	// Verify end bytes
	if data[len(data)-2] != endByte1 || data[len(data)-1] != endByte2 {
		return nil, fmt.Errorf("%w: invalid end bytes 0x%02x%02x",
			ErrMalformedPacket, data[len(data)-2], data[len(data)-1])
	}

	// Process packet based on protocol number
	var result *GT06Data
	var err error

	switch protocolNumber {
	case loginMsg:
		d.logDebug("Processing login message")
		result, err = d.decodeLoginMessage(payload)
	case locationMsg:
		d.logDebug("Processing location message")
		result, err = d.decodeLocationMessage(payload)
	case statusMsg:
		d.logDebug("Processing status message")
		result, err = d.decodeStatusMessage(payload)
	case alarmMsg:
		d.logDebug("Processing alarm message")
		result, err = d.decodeAlarmMessage(payload)
	}

	if err != nil {
		return nil, fmt.Errorf("error decoding message type 0x%02x: %w", protocolNumber, err)
	}

	return result, nil
}

func calculateChecksum(data []byte) uint16 {
	var checksum uint16
	for _, b := range data {
		checksum ^= uint16(b)
	}
	return checksum
}

func (d *Decoder) decodeLocationMessage(data []byte) (*GT06Data, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("location message too short")
	}

	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	// Read GPS status byte
	statusByte := data[0]
	result.GPSValid = (statusByte&0x01) == 0x01
	result.Satellites = (statusByte >> 2) & 0x0F

	d.logDebug("GPS Valid: %v, Satellites: %d", result.GPSValid, result.Satellites)

	// Extract coordinates
	rawLat := binary.BigEndian.Uint32(data[1:5])
	rawLon := binary.BigEndian.Uint32(data[5:9])

	var err error
	if result.Latitude, err = bcdToFloat(rawLat); err != nil {
		return nil, fmt.Errorf("invalid latitude (0x%08x): %w", rawLat, err)
	}
	if result.Longitude, err = bcdToFloat(rawLon); err != nil {
		return nil, fmt.Errorf("invalid longitude (0x%08x): %w", rawLon, err)
	}

	d.logDebug("Position: %.6f, %.6f", result.Latitude, result.Longitude)

	// Extract speed and course
	result.Speed = float64(data[9])
	result.Course = float64(binary.BigEndian.Uint16(data[10:12]))

	d.logDebug("Speed: %.1f, Course: %.1f", result.Speed, result.Course)

	// Parse timestamp if present
	if len(data) >= 18 {
		timeData := data[12:18]
		if ts, err := d.parseTimestamp(bytes.NewReader(timeData)); err == nil {
			result.Timestamp = ts
			d.logDebug("Timestamp: %v", result.Timestamp)
		} else {
			d.logDebug("Failed to parse timestamp: %v", err)
		}
	}

	return result, nil
}

type GT06Data struct {
	Latitude    float64
	Longitude   float64
	Speed       float64
	Course      float64
	Timestamp   time.Time
	Valid       bool
	GPSValid    bool
	Satellites  uint8
	PowerLevel  uint8
	GSMSignal   uint8
	Alarm       string
	Status      map[string]interface{}
}

func (d *Decoder) GenerateResponse(msgType uint8, deviceID string) []byte {
	switch msgType {
	case loginMsg:
		return d.generateLoginResponse(deviceID)
	case locationMsg:
		return d.generateLocationResponse()
	case alarmMsg:
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
	resp = append(resp, startByte1, startByte2)

	// Packet length
	resp = append(resp, byte(respLen))

	// Protocol number (login response)
	resp = append(resp, loginResp)

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
	resp = append(resp, endByte1, endByte2)

	return resp
}

func (d *Decoder) generateLocationResponse() []byte {
	resp := []byte{
		startByte1, startByte2, // Start bytes
		0x05,                   // Packet length
		locationResp,           // Protocol number (location response)
		0x00, 0x01,            // Serial number
		0x00, 0x01,            // CRC
		endByte1, endByte2,    // End bytes
	}
	return resp
}

func (d *Decoder) generateAlarmResponse() []byte {
	resp := []byte{
		startByte1, startByte2, // Start bytes
		0x05,                   // Packet length
		alarmResp,              // Protocol number (alarm response)
		0x00, 0x01,            // Serial number
		0x00, 0x01,            // CRC
		endByte1, endByte2,    // End bytes
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

func (d *Decoder) decodeAlarmMessage(data []byte) (*GT06Data, error) {
	// Alarm message contains location data followed by alarm type
	// Length should be at least location data (12 bytes) + alarm type (1 byte)
	if len(data) < 13 {
		return nil, fmt.Errorf("alarm message too short: need 13 bytes, got %d", len(data))
	}

	// Location data is all bytes except the last one (alarm type)
	locationData, err := d.decodeLocationMessage(data[:len(data)-1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode location data in alarm message: %w", err)
	}

	// Last byte is alarm type
	alarmType := data[len(data)-1]
	switch alarmType {
	case sosAlarm:
		locationData.Alarm = "sos"
	case powerCutAlarm:
		locationData.Alarm = "powerCut"
	case vibrationAlarm:
		locationData.Alarm = "vibration"
	case fenceInAlarm:
		locationData.Alarm = "geofenceEnter"
	case fenceOutAlarm:
		locationData.Alarm = "geofenceExit"
	case lowBatteryAlarm:
		locationData.Alarm = "lowBattery"
	case overspeedAlarm:
		locationData.Alarm = "overspeed"
	default:
		locationData.Alarm = fmt.Sprintf("unknown_%02x", alarmType)
	}

	// Add alarm to status map
	if locationData.Status == nil {
		locationData.Status = make(map[string]interface{})
	}
	locationData.Status["alarm"] = locationData.Alarm

	return locationData, nil
}

func (d *Decoder) decodeStatusMessage(data []byte) (*GT06Data, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("status message too short")
	}

	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	statusByte := data[0]
	result.PowerLevel = (statusByte >> 4) & 0x0F
	result.GSMSignal = statusByte & 0x0F

	result.Status["powerLevel"] = result.PowerLevel
	result.Status["gsmSignal"] = result.GSMSignal
	result.Status["charging"] = (statusByte&0x20 != 0)
	result.Status["engineOn"] = (statusByte&0x40 != 0)

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

func bcdToFloat(bcd uint32) (float64, error) {
	// Convert BCD to binary values for each digit
	d1 := (bcd >> 28) & 0x0F
	d2 := (bcd >> 24) & 0x0F
	d3 := (bcd >> 20) & 0x0F
	d4 := (bcd >> 16) & 0x0F
	d5 := (bcd >> 12) & 0x0F
	d6 := (bcd >> 8) & 0x0F
	d7 := (bcd >> 4) & 0x0F
	d8 := bcd & 0x0F

	// Validate BCD digits
	for _, digit := range []uint32{d1, d2, d3, d4, d5, d6, d7, d8} {
		if digit > 9 {
			return 0, ErrInvalidCoordinate
		}
	}

	// Combine BCD digits into degrees and minutes
	degrees := float64(d1*10 + d2)
	minutes := float64(d3*10+d4) + float64(d5)/10.0 + float64(d6)/100.0 +
		float64(d7)/1000.0 + float64(d8)/10000.0

	// Validate ranges
	if degrees > 90 || minutes >= 60 {
		return 0, ErrInvalidCoordinate
	}

	// Convert to decimal degrees
	return degrees + minutes/60.0, nil
}

func (d *Decoder) parseTimestamp(reader *bytes.Reader) (time.Time, error) {
	var timeBytes [6]byte
	if _, err := reader.Read(timeBytes[:]); err != nil {
		return time.Time{}, err
	}

	// Extract each component from BCD bytes
	// Year is stored as two BCD digits (e.g., 23 for 2023)
	year := 2000 + ((int(timeBytes[0])>>4)*10 + int(timeBytes[0]&0x0F))

	// Month is stored as two BCD digits (01-12)
	month := (int(timeBytes[1])>>4)*10 + int(timeBytes[1]&0x0F)

	// Day is stored as two BCD digits (01-31)
	day := (int(timeBytes[2])>>4)*10 + int(timeBytes[2]&0x0F)

	// Hours is stored as two BCD digits (00-23)
	hour := (int(timeBytes[3])>>4)*10 + int(timeBytes[3]&0x0F)

	// Minutes is stored as two BCD digits (00-59)
	minute := (int(timeBytes[4])>>4)*10 + int(timeBytes[4]&0x0F)

	// Seconds is stored as two BCD digits (00-59)
	second := (int(timeBytes[5])>>4)*10 + int(timeBytes[5]&0x0F)

	// Validate ranges
	if month < 1 || month > 12 || day < 1 || day > 31 ||
		hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, ErrInvalidTimestamp
	}

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC), nil
}

func (d *Decoder) ToPosition(deviceID string, data *GT06Data) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Valid = data.GPSValid
	position.Timestamp = data.Timestamp
	position.Protocol = "gt06"
	position.Satellites = data.Satellites

	// Add additional status information
	if data.Alarm != "" {
		position.Status = map[string]interface{}{
			"alarm":      data.Alarm,
			"powerLevel": data.PowerLevel,
			"gsmSignal":  data.GSMSignal,
		}
		// Add all status fields
		for k, v := range data.Status {
			position.Status[k] = v
		}
	}

	return position
}