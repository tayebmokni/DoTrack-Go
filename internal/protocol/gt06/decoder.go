package gt06

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
	"tracking/internal/core/model"
)

type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
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
}

// GT06 protocol constants
const (
	startByte1 = 0x78
	startByte2 = 0x78
	minLength  = 15
	endByte    = 0x0D0A
)

func (d *Decoder) Decode(data []byte) (*GT06Data, error) {
	if len(data) < minLength {
		return nil, errors.New("data too short for GT06 protocol")
	}

	// Verify protocol header
	if data[0] != startByte1 || data[1] != startByte2 {
		return nil, errors.New("invalid GT06 protocol header")
	}

	// Verify checksum
	if !d.validateChecksum(data) {
		return nil, errors.New("invalid checksum")
	}

	// Skip header (2 bytes) and packet length (1 byte)
	reader := bytes.NewReader(data[3:])
	result := &GT06Data{
		Valid: true,
	}

	// Read protocol number (1 byte)
	var protocolNumber uint8
	if err := binary.Read(reader, binary.BigEndian, &protocolNumber); err != nil {
		return nil, err
	}

	// Read GPS status byte
	var statusByte uint8
	if err := binary.Read(reader, binary.BigEndian, &statusByte); err != nil {
		return nil, err
	}

	// Parse GPS status
	result.GPSValid = (statusByte & 0x01) == 0x01
	result.Satellites = (statusByte >> 2) & 0x0F

	// Parse location data
	var rawLat, rawLon uint32
	if err := binary.Read(reader, binary.BigEndian, &rawLat); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &rawLon); err != nil {
		return nil, err
	}

	var err error
	result.Latitude, err = bcdToFloat(rawLat)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude: %v", err)
	}

	result.Longitude, err = bcdToFloat(rawLon)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude: %v", err)
	}

	// Read speed (1 byte)
	var speed uint8
	if err := binary.Read(reader, binary.BigEndian, &speed); err != nil {
		return nil, err
	}
	result.Speed = float64(speed)

	// Read course (2 bytes)
	var course uint16
	if err := binary.Read(reader, binary.BigEndian, &course); err != nil {
		return nil, err
	}
	result.Course = float64(course)

	// Parse timestamp (6 bytes, BCD format)
	result.Timestamp, err = d.parseTimestamp(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %v", err)
	}

	return result, nil
}

// validateChecksum verifies the data integrity
func (d *Decoder) validateChecksum(data []byte) bool {
	if len(data) < 3 {
		return false
	}

	length := int(data[2])
	if len(data) < length+5 { // Header(2) + Length(1) + Data(length) + Checksum(2)
		return false
	}

	checksum := uint16(0)
	for i := 2; i < length+3; i++ {
		checksum ^= uint16(data[i])
	}

	packetChecksum := binary.BigEndian.Uint16(data[length+3:length+5])
	return checksum == packetChecksum
}

// bcdToFloat converts BCD-encoded coordinates to decimal degrees
func bcdToFloat(bcd uint32) (float64, error) {
	// Extract degrees and minutes from BCD
	deg := float64((bcd>>20)&0xF)*10 + float64((bcd>>16)&0xF)
	min := float64((bcd>>12)&0xF)*10 + float64((bcd>>8)&0xF) +
		(float64((bcd>>4)&0xF)*10 + float64(bcd&0xF)) / 100.0

	if deg > 90 || min >= 60 {
		return 0, errors.New("invalid BCD coordinate value")
	}

	return deg + (min / 60.0), nil
}

// parseTimestamp extracts timestamp from BCD encoded bytes
func (d *Decoder) parseTimestamp(reader *bytes.Reader) (time.Time, error) {
	var timeBytes [6]byte
	if _, err := reader.Read(timeBytes[:]); err != nil {
		return time.Time{}, err
	}

	// Convert BCD to integers
	year := 2000 + ((int(timeBytes[0])>>4)*10 + int(timeBytes[0]&0x0F))
	month := (int(timeBytes[1])>>4)*10 + int(timeBytes[1]&0x0F)
	day := (int(timeBytes[2])>>4)*10 + int(timeBytes[2]&0x0F)
	hour := (int(timeBytes[3])>>4)*10 + int(timeBytes[3]&0x0F)
	minute := (int(timeBytes[4])>>4)*10 + int(timeBytes[4]&0x0F)
	second := (int(timeBytes[5])>>4)*10 + int(timeBytes[5]&0x0F)

	// Validate time components
	if month < 1 || month > 12 || day < 1 || day > 31 ||
		hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, errors.New("invalid timestamp values")
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
	return position
}