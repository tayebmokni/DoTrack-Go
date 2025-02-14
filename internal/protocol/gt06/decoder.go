// Package gt06 implements the GT06/GT06N GPS tracker protocol decoder
// Protocol Information:
// The GT06 protocol uses the following packet structure:
//   - Start:    2 bytes  (0x78 0x78)
//   - Length:   1 byte   (payload length)
//   - Protocol: 1 byte   (message type identifier)
//   - Payload:  n bytes  (varies by message type)
//   - Checksum: 2 bytes  (XOR of payload bytes)
//   - End:      2 bytes  (0x0D 0x0A)
//
// Message Types:
//   0x01: Login Message - Device identification and authentication
//   0x12: Location Message - GPS position and status information
//   0x13: Status Message - Device status (power, GSM signal, etc.)
//   0x16: Alarm Message - Various alerts (SOS, power cut, geofence, etc.)
//
// Coordinate Format:
//   Coordinates are encoded in BCD (Binary Coded Decimal) format
//   Example: 12°34.5678' is encoded as 0x12345678
//   - First two digits: Degrees
//   - Next two digits: Minutes
//   - Last four digits: Decimal minutes (scaled by 10000)
//
// For detailed protocol specification, see the GT06 protocol documentation.

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

func (d *Decoder) Decode(data []byte) (*GT06Data, error) {
	d.logDebug("Starting packet decode...")
	d.logPacket(data, "Received")

	// Validate minimum length (start + length + protocol + checksum + end)
	if len(data) < minLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d",
			ErrPacketTooShort, len(data), minLength)
	}

	// Validate start bytes
	if data[0] != startByte1 || data[1] != startByte2 {
		return nil, fmt.Errorf("%w: expected 0x%02x%02x, got 0x%02x%02x",
			ErrInvalidHeader, startByte1, startByte2, data[0], data[1])
	}

	// Extract packet length from third byte
	packetLength := int(data[2])
	expectedLen := packetLength + 5 // start(2) + length(1) + payload(n) + end(2)

	if len(data) < expectedLen {
		return nil, fmt.Errorf("%w: got %d bytes, need %d", ErrPacketTooShort, len(data), expectedLen)
	}

	if len(data) > expectedLen {
		return nil, fmt.Errorf("%w: got %d bytes, expected %d", ErrInvalidLength, len(data), expectedLen)
	}

	// Validate end bytes
	if data[len(data)-2] != endByte1 || data[len(data)-1] != endByte2 {
		return nil, fmt.Errorf("%w: invalid end bytes", ErrMalformedPacket)
	}

	// Validate checksum
	if !d.validateChecksum(data) {
		// Calculate expected checksum for debugging
		var checksum uint16
		for i := 3; i < packetLength+3; i++ {
			checksum ^= uint16(data[i])
		}
		actualChecksum := uint16(data[packetLength+3])<<8 | uint16(data[packetLength+4])
		return nil, fmt.Errorf("%w: calculated=0x%04x, received=0x%04x",
			ErrInvalidChecksum, checksum, actualChecksum)
	}

	reader := bytes.NewReader(data[3:])
	var protocolNumber uint8
	if err := binary.Read(reader, binary.BigEndian, &protocolNumber); err != nil {
		return nil, fmt.Errorf("failed to read protocol number: %w", err)
	}

	d.logDebug("Protocol number: 0x%02x", protocolNumber)

	// Process based on message type
	var result *GT06Data
	var err error

	switch protocolNumber {
	case loginMsg:
		d.logDebug("Processing login message")
		result, err = d.decodeLoginMessage(reader)
	case locationMsg:
		d.logDebug("Processing location message")
		result, err = d.decodeLocationMessage(reader)
	case statusMsg:
		d.logDebug("Processing status message")
		result, err = d.decodeStatusMessage(reader)
	case alarmMsg:
		d.logDebug("Processing alarm message")
		result, err = d.decodeAlarmMessage(reader)
	default:
		return nil, fmt.Errorf("%w: 0x%02x", ErrInvalidMessageType, protocolNumber)
	}

	if err != nil {
		return nil, fmt.Errorf("error decoding message type 0x%02x: %w", protocolNumber, err)
	}

	if result != nil {
		d.logDebug("Successfully decoded packet: %+v", result)
	}

	return result, nil
}

func (d *Decoder) validateChecksum(data []byte) bool {
	length := int(data[2])

	// Check if there's enough data for checksum
	if len(data) < length+5 {
		return false
	}

	// Calculate checksum over payload only (excluding start bytes and length byte)
	var checksum uint16
	for i := 3; i < length+3; i++ {
		checksum ^= uint16(data[i])
	}

	// Extract checksum from packet (big-endian)
	packetChecksum := uint16(data[length+3])<<8 | uint16(data[length+4])

	return checksum == packetChecksum
}

func (d *Decoder) decodeLocationMessage(reader *bytes.Reader) (*GT06Data, error) {
	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	var statusByte uint8
	if err := binary.Read(reader, binary.BigEndian, &statusByte); err != nil {
		return nil, fmt.Errorf("failed to read status byte: %w", err)
	}

	result.GPSValid = (statusByte&0x01) == 0x01
	result.Satellites = (statusByte >> 2) & 0x0F

	d.logDebug("GPS Valid: %v, Satellites: %d", result.GPSValid, result.Satellites)

	var rawLat, rawLon uint32
	if err := binary.Read(reader, binary.BigEndian, &rawLat); err != nil {
		return nil, fmt.Errorf("failed to read latitude: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &rawLon); err != nil {
		return nil, fmt.Errorf("failed to read longitude: %w", err)
	}

	var err error
	if result.Latitude, err = bcdToFloat(rawLat); err != nil {
		return nil, fmt.Errorf("invalid latitude (0x%08x): %w", rawLat, err)
	}
	if result.Longitude, err = bcdToFloat(rawLon); err != nil {
		return nil, fmt.Errorf("invalid longitude (0x%08x): %w", rawLon, err)
	}

	d.logDebug("Position: %.6f, %.6f", result.Latitude, result.Longitude)

	var speed uint8
	if err := binary.Read(reader, binary.BigEndian, &speed); err != nil {
		return nil, fmt.Errorf("failed to read speed: %w", err)
	}
	result.Speed = float64(speed)

	var course uint16
	if err := binary.Read(reader, binary.BigEndian, &course); err != nil {
		return nil, fmt.Errorf("failed to read course: %w", err)
	}
	result.Course = float64(course)

	d.logDebug("Speed: %.1f, Course: %.1f", result.Speed, result.Course)

	if result.Timestamp, err = d.parseTimestamp(reader); err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	d.logDebug("Timestamp: %v", result.Timestamp)
	return result, nil
}

type GT06Data struct {
	Latitude    float64
	Longitude   float64
	Speed      float64
	Course     float64
	Timestamp  time.Time
	Valid      bool
	GPSValid   bool
	Satellites uint8
	PowerLevel uint8
	GSMSignal  uint8
	Alarm      string
	Status     map[string]interface{}
}

// GT06 protocol constants
const (
	startByte1 = 0x78
	startByte2 = 0x78
	minLength  = 15
	endByte1   = 0x0D // Split endByte into two constants
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
	sosAlarm          = 0x01
	powerCutAlarm     = 0x02
	vibrationAlarm    = 0x04
	fenceInAlarm      = 0x10
	fenceOutAlarm     = 0x11
	lowBatteryAlarm   = 0x20
	overspeedAlarm    = 0x40
)

// GenerateResponse generates the appropriate response packet based on the received message type
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

func (d *Decoder) decodeAlarmMessage(reader *bytes.Reader) (*GT06Data, error) {
	locationData, err := d.decodeLocationMessage(reader)
	if err != nil {
		return nil, err
	}

	var alarmType uint8
	if err := binary.Read(reader, binary.BigEndian, &alarmType); err != nil {
		return nil, err
	}

	// Set alarm type based on the received code
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

	return locationData, nil
}

func (d *Decoder) decodeStatusMessage(reader *bytes.Reader) (*GT06Data, error) {
	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	var statusByte uint8
	if err := binary.Read(reader, binary.BigEndian, &statusByte); err != nil {
		return nil, err
	}

	result.PowerLevel = (statusByte >> 4) & 0x0F
	result.GSMSignal = statusByte & 0x0F

	result.Status["powerLevel"] = result.PowerLevel
	result.Status["gsmSignal"] = result.GSMSignal
	result.Status["charging"] = (statusByte&0x20 != 0)
	result.Status["engineOn"] = (statusByte&0x40 != 0)

	return result, nil
}

func (d *Decoder) decodeLoginMessage(reader *bytes.Reader) (*GT06Data, error) {
	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	// Read IMEI (8 bytes)
	imei := make([]byte, 8)
	if _, err := reader.Read(imei); err != nil {
		return nil, err
	}
	result.Status["imei"] = fmt.Sprintf("%x", imei)

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

	// Validate all digits are valid BCD
	for _, digit := range []uint32{d1, d2, d3, d4, d5, d6, d7, d8} {
		if digit > 9 {
			return 0, ErrInvalidCoordinate
		}
	}

	// First two digits are degrees, next two are minutes, last four are decimal minutes
	degrees := float64(d1)*10 + float64(d2)
	minutes := float64(d3)*10 + float64(d4)
	decimalMinutes := float64(d5)*0.1 + float64(d6)*0.01 + float64(d7)*0.001 + float64(d8)*0.0001

	// Validate ranges
	if degrees > 90 || minutes >= 60 || decimalMinutes >= 1.0 {
		return 0, ErrInvalidCoordinate
	}

	// Convert to decimal degrees
	return degrees + (minutes + decimalMinutes*60.0)/60.0, nil
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