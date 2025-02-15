// Package gt06 implements the GT06 protocol decoder (alternate implementation)
package gt06

import (
	"bytes"
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

func (d *DecoderV2) logPacket(data []byte, prefix string) {
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

func (d *DecoderV2) validatePacket(header *PacketHeader, data []byte) error {
	var minLength int
	switch header.Protocol {
	case LoginMsg:
		minLength = MinLoginLength
	case LocationMsg:
		minLength = MinLocationLength
	case StatusMsg:
		minLength = MinStatusLength
	case AlarmMsg:
		minLength = MinAlarmLength
	default:
		return fmt.Errorf("%w: 0x%02x", ErrInvalidMessageType, header.Protocol)
	}

	if len(data) < minLength {
		return fmt.Errorf("%w: got %d bytes, need at least %d",
			ErrPacketTooShort, len(data), minLength)
	}

	checksumPos := len(data) - 4
	calcChecksum := CalculateChecksum(data[2:checksumPos])
	recvChecksum := uint16(data[checksumPos])<<8 | uint16(data[checksumPos+1])

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

	if len(data) < MinPacketLength {
		return nil, fmt.Errorf("%w: need at least %d bytes",
			ErrPacketTooShort, MinPacketLength)
	}

	if data[0] != StartByte1 || data[1] != StartByte2 {
		return nil, fmt.Errorf("%w: expected 0x%02x%02x, got 0x%02x%02x",
			ErrInvalidHeader, StartByte1, StartByte2, data[0], data[1])
	}

	header := &PacketHeader{
		Length:    data[2],
		Protocol:  data[3],
		TotalSize: 2 + 1 + int(data[2]) + 2 + 2, // start(2) + len(1) + content(length) + checksum(2) + end(2)
	}

	if err := d.validatePacket(header, data); err != nil {
		return nil, fmt.Errorf("packet validation failed: %w", err)
	}

	payloadStart := 4
	payloadEnd := len(data) - 4
	payload := data[payloadStart:payloadEnd]

	d.logDebug("Processing payload: %d bytes, protocol=0x%02x",
		len(payload), header.Protocol)

	var result *GT06Data
	var err error

	switch header.Protocol {
	case LoginMsg:
		result, err = d.decodeLoginMessage(payload)
	case LocationMsg:
		result, err = d.decodeLocationMessage(payload)
	case StatusMsg:
		result, err = d.decodeStatusMessage(payload)
	case AlarmMsg:
		result, err = d.decodeAlarmMessage(payload)
	default:
		return nil, fmt.Errorf("unsupported protocol: 0x%02x", header.Protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode %s message: %w",
			GetMessageTypeName(header.Protocol), err)
	}

	return result, nil
}

func (d *DecoderV2) decodeLocationMessage(data []byte) (*GT06Data, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("location message too short: got %d bytes, need 10", len(data))
	}

	result := &GT06Data{
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
	locationData.Alarm = GetAlarmName(alarmType)
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
