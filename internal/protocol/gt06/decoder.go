package gt06

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"
	"tracking/internal/core/model"
)

type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

type GT06Data struct {
	Latitude  float64
	Longitude float64
	Speed     float64
	Course    float64
	Timestamp time.Time
	Valid     bool
}

// GT06 protocol constants
const (
	startByte1 = 0x78
	startByte2 = 0x78
	minLength  = 15
)

func (d *Decoder) Decode(data []byte) (*GT06Data, error) {
	if len(data) < minLength {
		return nil, errors.New("data too short for GT06 protocol")
	}

	// Verify protocol header
	if data[0] != startByte1 || data[1] != startByte2 {
		return nil, errors.New("invalid GT06 protocol header")
	}

	// Skip header (2 bytes) and packet length (1 byte)
	reader := bytes.NewReader(data[3:])
	result := &GT06Data{
		Timestamp: time.Now(), // Default to current time if parsing fails
		Valid:     true,
	}

	// Read protocol number (1 byte)
	var protocolNumber uint8
	if err := binary.Read(reader, binary.BigEndian, &protocolNumber); err != nil {
		return nil, err
	}

	// Parse location data
	// In GT06, latitude and longitude are stored as binary-coded decimal (BCD)
	// 4 bytes each, representing degrees and decimal minutes
	var rawLat, rawLon uint32
	if err := binary.Read(reader, binary.BigEndian, &rawLat); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &rawLon); err != nil {
		return nil, err
	}

	// Convert BCD coordinates to decimal degrees
	result.Latitude = bcdToFloat(rawLat)
	result.Longitude = bcdToFloat(rawLon)

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

	return result, nil
}

// bcdToFloat converts BCD-encoded coordinates to decimal degrees
func bcdToFloat(bcd uint32) float64 {
	// Extract degrees and minutes from BCD
	deg := float64((bcd >> 16) & 0xFF)
	min := float64(bcd & 0xFFFF) / 100.0
	return deg + (min / 60.0)
}

func (d *Decoder) ToPosition(deviceID string, data *GT06Data) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Protocol = "gt06"
	return position
}