package gt06

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"time"
	"tracking/internal/core/model"
)

// DecoderV2 represents an alternate implementation of the GT06 protocol decoder
type DecoderV2 struct {
	debug bool
}

func NewDecoderV2() *DecoderV2 {
	return &DecoderV2{debug: false}
}

func (d *DecoderV2) EnableDebug(enable bool) {
	d.debug = enable
}

func (d *DecoderV2) logDebug(format string, v ...interface{}) {
	if d.debug {
		log.Printf("[GT06v2] "+format, v...)
	}
}

type packetHeader struct {
	length    byte
	protocol  byte
	totalSize int
}

func (d *DecoderV2) validatePacket(header *packetHeader, data []byte) error {
	var minLength int
	switch header.protocol {
	case LoginMsg:
		minLength = 15
	case LocationMsg:
		minLength = 26
	case StatusMsg:
		minLength = 13
	case AlarmMsg:
		minLength = 27
	default:
		return fmt.Errorf("%w: 0x%02x", ErrInvalidMessageType, header.protocol)
	}

	if len(data) < minLength {
		return fmt.Errorf("%w: got %d bytes, need at least %d",
			ErrPacketTooShort, len(data), minLength)
	}

	checksumPos := len(data) - 4
	calcChecksum := calculateChecksum(data[2:checksumPos])
	recvChecksum := binary.BigEndian.Uint16(data[checksumPos:])

	if calcChecksum != recvChecksum {
		return fmt.Errorf("%w: calc=0x%04x, recv=0x%04x",
			ErrInvalidChecksum, calcChecksum, recvChecksum)
	}

	if data[len(data)-2] != EndByte1 || data[len(data)-1] != EndByte2 {
		return fmt.Errorf("%w: invalid end bytes", ErrMalformedPacket)
	}

	return nil
}

func (d *DecoderV2) Decode(data []byte) (*GT06Data, error) {
	d.logDebug("Starting packet decode (v2)...")

	if len(data) < 4 {
		return nil, fmt.Errorf("%w: need at least 4 bytes", ErrPacketTooShort)
	}

	if data[0] != StartByte1 || data[1] != StartByte2 {
		return nil, fmt.Errorf("%w: expected 0x%02x%02x, got 0x%02x%02x",
			ErrInvalidHeader, StartByte1, StartByte2, data[0], data[1])
	}

	header := &packetHeader{
		length:    data[2],
		protocol:  data[3],
		totalSize: 2 + 1 + int(data[2]) + 2 + 2, // start(2) + len(1) + content(length) + checksum(2) + end(2)
	}

	if err := d.validatePacket(header, data); err != nil {
		return nil, fmt.Errorf("packet validation failed: %w", err)
	}

	payloadStart := 4
	payloadEnd := len(data) - 4
	payload := data[payloadStart:payloadEnd]

	d.logDebug("Processing payload: %d bytes, protocol=0x%02x", len(payload), header.protocol)

	var result *GT06Data
	var err error

	switch header.protocol {
	case LoginMsg:
		result, err = d.decodeLoginMessage(payload)
	case LocationMsg:
		result, err = d.decodeLocationMessage(payload)
	case StatusMsg:
		result, err = d.decodeStatusMessage(payload)
	case AlarmMsg:
		result, err = d.decodeAlarmMessage(payload)
	default:
		return nil, fmt.Errorf("unsupported protocol: 0x%02x", header.protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode %s message: %w",
			getMessageTypeName(header.protocol), err)
	}

	return result, nil
}

func (d *DecoderV2) decodeLocationMessage(data []byte) (*GT06Data, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("location message too short: need 10 bytes")
	}

	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	statusByte := data[0]
	result.GPSValid = (statusByte&0x01) == 0x01
	result.Satellites = int((statusByte >> 2) & 0x0F)

	var err error
	if result.Latitude, err = bcdToFloat(binary.BigEndian.Uint32(data[1:5])); err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}
	if result.Longitude, err = bcdToFloat(binary.BigEndian.Uint32(data[5:9])); err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}

	result.Speed = float64(data[9])
	if len(data) >= 11 {
		result.Course = float64(binary.BigEndian.Uint16(data[10:12]))
	}

	if len(data) >= 16 {
		reader := bytes.NewReader(data[12:18])
		if ts, err := parseTimestamp(reader); err == nil {
			result.Timestamp = ts
		}
	}

	return result, nil
}

func (d *DecoderV2) decodeStatusMessage(data []byte) (*GT06Data, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("status message too short: need 4 bytes")
	}

	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	statusByte := data[0]
	result.PowerLevel = int((statusByte >> 4) & 0x0F)
	result.GSMSignal = int(statusByte & 0x0F)

	if result.PowerLevel > 15 {
		return nil, fmt.Errorf("invalid power level: %d", result.PowerLevel)
	}

	result.Status["powerLevel"] = result.PowerLevel
	result.Status["gsmSignal"] = result.GSMSignal

	if len(data) > 1 {
		result.Status["charging"] = (data[1]&0x20 != 0)
		result.Status["engineOn"] = (data[1]&0x40 != 0)
	}

	return result, nil
}

func (d *DecoderV2) decodeLoginMessage(data []byte) (*GT06Data, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("login message too short: need 8 bytes")
	}

	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	result.Status["imei"] = fmt.Sprintf("%x", data[:8])
	return result, nil
}

func (d *DecoderV2) decodeAlarmMessage(data []byte) (*GT06Data, error) {
	locationData, err := d.decodeLocationMessage(data[:len(data)-1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode location part: %w", err)
	}

	alarmType := data[len(data)-1]
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
		locationData.Alarm = fmt.Sprintf("unknown_%02x", alarmType)
	}

	locationData.Status["alarm"] = locationData.Alarm
	return locationData, nil
}

func (d *DecoderV2) ToPosition(deviceID string, data *GT06Data) *model.Position {
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

// Helper functions
func parseTimestamp(reader *bytes.Reader) (time.Time, error) {
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

func bcdToDec(b byte) int {
	return int(b>>4)*10 + int(b&0x0F)
}

func bcdToFloat(bcd uint32) (float64, error) {
	degrees := float64(bcdToDec(byte(bcd>>24)))*10 +
		float64(bcdToDec(byte((bcd>>16)&0xFF)))/60 +
		float64(bcdToDec(byte((bcd>>8)&0xFF)))/3600
	return degrees, nil
}

func getAlarmName(alarmType byte) string {
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

func getMessageTypeName(messageType byte) string {
	switch messageType {
	case LoginMsg:
		return "login"
	case LocationMsg:
		return "location"
	case StatusMsg:
		return "status"
	case AlarmMsg:
		return "alarm"
	default:
		return fmt.Sprintf("unknown(0x%02x)", messageType)
	}
}

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

const (
	StartByte1     = 0x78
	StartByte2     = 0x78
	EndByte1       = 0x0D
	EndByte2       = 0x0A
	LoginMsg       = 0x01
	LocationMsg    = 0x12
	StatusMsg      = 0x13
	AlarmMsg       = 0x16
	SosAlarm       = 0x01
	PowerCutAlarm  = 0x02
	VibrationAlarm = 0x04
	FenceInAlarm   = 0x08
	FenceOutAlarm  = 0x10
	LowBatteryAlarm = 0x20
	OverspeedAlarm  = 0x40
)

var ErrInvalidHeader = fmt.Errorf("invalid packet header")
var ErrPacketTooShort = fmt.Errorf("packet too short")
var ErrInvalidMessageType = fmt.Errorf("invalid message type")
var ErrInvalidLength = fmt.Errorf("invalid packet length")
var ErrInvalidChecksum = fmt.Errorf("invalid checksum")
var ErrMalformedPacket = fmt.Errorf("malformed packet")
var ErrInvalidTimestamp = fmt.Errorf("invalid timestamp")

func calculateChecksum(data []byte) uint16 {
	sum := uint16(0)
	for _, b := range data {
		sum += uint16(b)
	}
	return sum
}