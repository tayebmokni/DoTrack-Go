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

// GT06 protocol constants
const (
	startByte1 = 0x78
	startByte2 = 0x78
	minLength  = 10 // Minimum length: start(2) + length(1) + protocol(1) + content(1) + checksum(2) + end(2)
	endByte1   = 0x0D
	endByte2   = 0x0A

	// Message types
	loginMsg    = 0x01
	locationMsg = 0x12
	statusMsg   = 0x13
	alarmMsg    = 0x16

	// Minimum content lengths (excluding protocol number)
	loginMinLen    = 1  // IMEI (1+)
	locationMinLen = 10 // GPS data(10)
	statusMinLen   = 4  // Status(4)
	alarmMinLen    = 11 // GPS data(10) + Alarm type(1)
)

type Decoder struct {
	debug bool
}

func NewDecoder() *Decoder {
	return &Decoder{
		debug: false,
	}
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
	d.logPacket(data, "Received")

	// 1. Basic length check
	if len(data) < minLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d",
			ErrPacketTooShort, len(data), minLength)
	}

	// 2. Validate start bytes
	if data[0] != startByte1 || data[1] != startByte2 {
		return nil, fmt.Errorf("%w: expected 0x%02x%02x, got 0x%02x%02x",
			ErrInvalidHeader, startByte1, startByte2, data[0], data[1])
	}

	// 3. Get content length byte (includes protocol number)
	contentLen := int(data[2])
	d.logDebug("Content length byte: %d", contentLen)

	// 4. Calculate total packet length
	// Total = start(2) + length(1) + content(contentLen) + checksum(2) + end(2)
	expectedTotal := 2 + 1 + contentLen + 2 + 2

	// 5. Validate total length
	if len(data) != expectedTotal {
		return nil, fmt.Errorf("%w: got %d bytes, need %d",
			ErrInvalidLength, len(data), expectedTotal)
	}

	// 6. Get protocol number and validate minimum length
	protocolNumber := data[3]
	d.logDebug("Protocol number: 0x%02x", protocolNumber)

	// Calculate actual content length (excluding protocol byte)
	actualContentLen := contentLen - 1

	// 7. Validate protocol-specific minimum length
	var minContentLen int
	switch protocolNumber {
	case loginMsg:
		minContentLen = loginMinLen
	case locationMsg:
		minContentLen = locationMinLen
	case statusMsg:
		minContentLen = statusMinLen
	case alarmMsg:
		minContentLen = alarmMinLen
	default:
		return nil, fmt.Errorf("%w: 0x%02x", ErrInvalidMessageType, protocolNumber)
	}

	if actualContentLen < minContentLen {
		return nil, fmt.Errorf("%w: protocol 0x%02x requires at least %d content bytes, got %d",
			ErrInvalidLength, protocolNumber, minContentLen, actualContentLen)
	}

	// 8. Extract payload
	payloadStart := 4 // After start(2) + length(1) + protocol(1)
	payloadEnd := payloadStart + actualContentLen

	// 9. Validate payload boundaries
	if payloadEnd+4 > len(data) {
		return nil, fmt.Errorf("%w: invalid payload boundaries",
			ErrMalformedPacket)
	}

	payload := data[payloadStart:payloadEnd]
	d.logDebug("Extracted payload: %d bytes", len(payload))

	// 10. Validate checksum
	checksumData := data[2:payloadEnd]
	calculatedChecksum := calculateChecksum(checksumData)
	receivedChecksum := uint16(data[payloadEnd])<<8 | uint16(data[payloadEnd+1])

	if calculatedChecksum != receivedChecksum {
		return nil, fmt.Errorf("%w: calc=0x%04x, recv=0x%04x",
			ErrInvalidChecksum, calculatedChecksum, receivedChecksum)
	}

	// 11. Validate end bytes
	if data[payloadEnd+2] != endByte1 || data[payloadEnd+3] != endByte2 {
		return nil, fmt.Errorf("%w: invalid end bytes", ErrMalformedPacket)
	}

	// 12. Process message based on protocol
	var result *GT06Data
	var err error

	switch protocolNumber {
	case loginMsg:
		result, err = d.decodeLoginMessage(payload)
	case locationMsg:
		result, err = d.decodeLocationMessage(payload)
	case statusMsg:
		result, err = d.decodeStatusMessage(payload)
	case alarmMsg:
		result, err = d.decodeAlarmMessage(payload)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	d.logDebug("Successfully decoded %s message", getMessageTypeName(protocolNumber))
	return result, nil
}

func getMessageTypeName(protocolNumber byte) string {
	switch protocolNumber {
	case loginMsg:
		return "login"
	case locationMsg:
		return "location"
	case statusMsg:
		return "status"
	case alarmMsg:
		return "alarm"
	default:
		return fmt.Sprintf("unknown_0x%02x", protocolNumber)
	}
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
	// First decode location data
	locationData, err := d.decodeLocationMessage(data[:len(data)-1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode location data in alarm message: %w", err)
	}

	// Extract alarm type from last byte
	alarmType := data[len(data)-1]
	d.logDebug("Processing alarm type: 0x%02x", alarmType)

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
		// For unknown alarm types, use a consistent format
		locationData.Alarm = fmt.Sprintf("unknown_%02x", alarmType)
		d.logDebug("Unknown alarm type 0x%02x, using %s",
			alarmType, locationData.Alarm)
	}

	// Add alarm type to status map
	if locationData.Status == nil {
		locationData.Status = make(map[string]interface{})
	}
	locationData.Status["alarm"] = locationData.Alarm

	d.logDebug("Alarm message decoded: type=%s", locationData.Alarm)
	return locationData, nil
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
	result.PowerLevel = (statusByte >> 4) & 0x0F
	result.GSMSignal = statusByte & 0x0F

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

		d.logDebug("Status flags: charging=%v, engineOn=%v",
			result.Status["charging"], result.Status["engineOn"])
	}

	d.logDebug("Status: power=%d/15, gsm=%d/15",
		result.PowerLevel, result.GSMSignal)

	return result, nil
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
	// Extract BCD digits
	d1 := (bcd >> 28) & 0x0F
	d2 := (bcd >> 24) & 0x0F
	d3 := (bcd >> 20) & 0x0F
	d4 := (bcd >> 16) & 0x0F
	d5 := (bcd >> 12) & 0x0F
	d6 := (bcd >> 8) & 0x0F
	d7 := (bcd >> 4) & 0x0F
	d8 := bcd & 0x0F

	// Validate each BCD digit
	for i, digit := range []uint32{d1, d2, d3, d4, d5, d6, d7, d8} {
		if digit > 9 {
			return 0, fmt.Errorf("%w: invalid BCD digit %d at position %d",
				ErrInvalidCoordinate, digit, i+1)
		}
	}

	// Convert to degrees and minutes
	degrees := float64(d1*10 + d2)
	minutes := float64(d3*10) + float64(d4) +
		float64(d5)/10.0 + float64(d6)/100.0 +
		float64(d7)/1000.0 + float64(d8)/10000.0

	// Validate ranges
	if degrees > 90 || minutes >= 60 {
		return 0, fmt.Errorf("%w: invalid values (deg=%.0f, min=%.4f)",
			ErrInvalidCoordinate, degrees, minutes)
	}

	return degrees + (minutes / 60.0), nil
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

var (
	loginResp    = byte(0x01)
	locationResp = byte(0x12)
	alarmResp    = byte(0x16)
	sosAlarm     = byte(0x01)
	powerCutAlarm = byte(0x02)
	vibrationAlarm = byte(0x03)
	fenceInAlarm  = byte(0x04)
	fenceOutAlarm = byte(0x05)
	lowBatteryAlarm = byte(0x06)
	overspeedAlarm = byte(0x07)
)