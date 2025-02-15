// Package gt06 implements the GT06 protocol decoder
package gt06

import (
	"bytes"
	"fmt"
	"log"
	"time"
	"tracking/internal/core/model"
)

// Response types (only used in decoder.go)
const (
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

func (d *Decoder) Decode(data []byte) (*model.GT06Data, error) { // Changed to use model.GT06Data
	d.logDebug("Starting packet decode...")
	d.logPacket(data, "Received")

	if len(data) < MinPacketLength {
		return nil, fmt.Errorf("%w: need at least %d bytes",
			ErrPacketTooShort, MinPacketLength)
	}

	if data[0] != StartByte1 || data[1] != StartByte2 {
		return nil, fmt.Errorf("%w: expected 0x%02x%02x, got 0x%02x%02x",
			ErrInvalidHeader, StartByte1, StartByte2, data[0], data[1])
	}

	protocolNumber := data[3]
	d.logDebug("Protocol number: 0x%02x", protocolNumber)

	var minLength int
	switch protocolNumber {
	case LoginMsg:
		minLength = MinLoginLength
	case LocationMsg:
		minLength = MinLocationLength
	case StatusMsg:
		minLength = MinStatusLength
	case AlarmMsg:
		minLength = MinAlarmLength
	default:
		return nil, fmt.Errorf("%w: 0x%02x", ErrInvalidMessageType, protocolNumber)
	}

	if len(data) < minLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d for protocol 0x%02x",
			ErrPacketTooShort, len(data), minLength, protocolNumber)
	}

	declaredLen := int(data[2])
	expectedLen := len(data) - 5 // subtract start(2), len(1), checksum(2)
	if declaredLen != expectedLen {
		return nil, fmt.Errorf("%w: declared=%d, actual=%d",
			ErrInvalidLength, declaredLen, expectedLen)
	}

	checksumPos := len(data) - 4
	calcChecksum := CalculateChecksum(data[2:checksumPos])
	recvChecksum := uint16(data[checksumPos])<<8 | uint16(data[checksumPos+1])

	if calcChecksum != recvChecksum {
		return nil, fmt.Errorf("%w: calc=0x%04x, recv=0x%04x",
			ErrInvalidChecksum, calcChecksum, recvChecksum)
	}

	if data[len(data)-2] != EndByte1 || data[len(data)-1] != EndByte2 {
		return nil, fmt.Errorf("%w: invalid end bytes", ErrMalformedPacket)
	}

	content := data[4:checksumPos]
	d.logDebug("Content length: %d bytes", len(content))

	var result *model.GT06Data // Changed to use model.GT06Data
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
			GetMessageTypeName(protocolNumber), err)
	}

	return result, nil
}

func (d *Decoder) decodeLocationMessage(data []byte) (*model.GT06Data, error) { // Changed to use model.GT06Data
	if len(data) < 10 {
		return nil, fmt.Errorf("location message too short: got %d bytes, need 10", len(data))
	}

	result := &model.GT06Data{ // Changed to use model.GT06Data
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	statusByte := data[0]
	result.GPSValid = (statusByte&0x01) == 0x01
	result.Satellites = int((statusByte >> 2) & 0x0F)

	var err error
	if result.Latitude, err = BcdToFloat(uint32(data[1])<<24 | uint32(data[2])<<16 | uint32(data[3])<<8 | uint32(data[4])); err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}
	if result.Longitude, err = BcdToFloat(uint32(data[5])<<24 | uint32(data[6])<<16 | uint32(data[7])<<8 | uint32(data[8])); err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}

	if err := ValidateCoordinates(result.Latitude, result.Longitude); err != nil {
		return nil, err
	}

	result.Speed = float64(data[9])
	if len(data) >= 11 {
		result.Course = float64(uint16(data[10])<<8 | uint16(data[11]))
	}

	if len(data) >= 16 {
		if ts, err := ParseTimestamp(bytes.NewReader(data[12:18])); err == nil {
			result.Timestamp = ts
		}
	}

	return result, nil
}

func (d *Decoder) decodeStatusMessage(data []byte) (*model.GT06Data, error) { // Changed to use model.GT06Data
	if len(data) < 4 {
		return nil, fmt.Errorf("%w: status message too short", ErrInvalidLength)
	}

	result := &model.GT06Data{ // Changed to use model.GT06Data
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	statusByte := data[0]
	result.PowerLevel = int((statusByte >> 4) & 0x0F)
	result.GSMSignal = int(statusByte & 0x0F)

	if result.PowerLevel > 15 {
		return nil, fmt.Errorf("%w: power level %d exceeds maximum of 15",
			ErrMalformedPacket, result.PowerLevel)
	}

	result.Status["powerLevel"] = result.PowerLevel
	result.Status["gsmSignal"] = result.GSMSignal

	if len(data) > 1 {
		result.Status["charging"] = (data[1]&0x20 != 0)
		result.Status["engineOn"] = (data[1]&0x40 != 0)
	}

	return result, nil
}

func (d *Decoder) decodeLoginMessage(data []byte) (*model.GT06Data, error) { // Changed to use model.GT06Data
	if len(data) < 8 {
		return nil, fmt.Errorf("login message too short")
	}

	result := &model.GT06Data{ // Changed to use model.GT06Data
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	result.Status["imei"] = fmt.Sprintf("%x", data[:8])
	return result, nil
}

func (d *Decoder) decodeAlarmMessage(data []byte) (*model.GT06Data, error) { // Changed to use model.GT06Data
	locationData, err := d.decodeLocationMessage(data[:len(data)-1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode location part: %w", err)
	}

	alarmType := data[len(data)-1]
	locationData.Alarm = GetAlarmName(alarmType)
	locationData.Status["alarm"] = locationData.Alarm

	return locationData, nil
}

func (d *Decoder) ToPosition(deviceID string, data *model.GT06Data) *model.Position { // Changed to use model.GT06Data and model.Position
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Valid = data.GPSValid
	position.Protocol = "gt06"
	position.Satellites = data.Satellites
	position.Timestamp = data.Timestamp

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
	respLen := len(deviceID) + 10 // Add length of fixed fields
	resp := make([]byte, 0, respLen+5)

	resp = append(resp, StartByte1, StartByte2)
	resp = append(resp, byte(respLen))
	resp = append(resp, LoginResp)
	resp = append(resp, []byte(deviceID)...)

	now := time.Now().UTC()
	resp = append(resp, byte(now.Hour()), byte(now.Minute()))
	resp = append(resp, 0x00, 0x01) // Serial number
	resp = append(resp, 0x00, 0x00) // Error code (success)

	crc := CalculateChecksum(resp[2:])
	resp = append(resp, byte(crc>>8), byte(crc))
	resp = append(resp, EndByte1, EndByte2)

	return resp
}

func (d *Decoder) generateLocationResponse() []byte {
	resp := []byte{
		StartByte1, StartByte2,
		0x05,          // Packet length
		LocationResp,  // Protocol number
		0x00, 0x01,   // Serial number
		0x00, 0x01,   // CRC
		EndByte1, EndByte2,
	}
	return resp
}

func (d *Decoder) generateAlarmResponse() []byte {
	resp := []byte{
		StartByte1, StartByte2,
		0x05,         // Packet length
		AlarmResp,    // Protocol number
		0x00, 0x01,  // Serial number
		0x00, 0x01,  // CRC
		EndByte1, EndByte2,
	}
	return resp
}

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

func CalculateChecksum(data []byte) uint16 {
	var sum uint16
	for _, b := range data {
		sum ^= uint16(b)
	}
	return sum
}

func BcdToFloat(bcd uint32) (float64, error) {
	degrees := float64(bcdToDec(byte(bcd>>24)))*10 +
		float64(bcdToDec(byte((bcd>>16)&0xFF)))/60 +
		float64(bcdToDec(byte((bcd>>8)&0xFF)))/3600
	return degrees, nil
}

func bcdToDec(b byte) int {
	return int(b>>4)*10 + int(b&0x0F)
}

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

func ValidateCoordinates(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return ErrInvalidCoordinate
	}
	return nil
}

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


// Common GT06 errors
var (
	ErrInvalidHeader      = fmt.Errorf("invalid GT06 protocol header")
	ErrPacketTooShort     = fmt.Errorf("data too short for GT06 protocol")
	ErrInvalidChecksum    = fmt.Errorf("invalid checksum")
	ErrInvalidCoordinate  = fmt.Errorf("invalid BCD coordinate value")
	ErrInvalidTimestamp   = fmt.Errorf("invalid timestamp values")
	ErrInvalidLength      = fmt.Errorf("packet length mismatch")
	ErrInvalidMessageType = fmt.Errorf("unsupported message type")
	ErrMalformedPacket    = fmt.Errorf("malformed packet structure")
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

	MinPacketLength    = 7
	MinLoginLength     = 15
	MinLocationLength  = 26
	MinStatusLength    = 13
	MinAlarmLength     = 27
)