package gt06

import (
	"bytes"
	"encoding/binary"
	"errors"
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
	Timestamp int64
}

// The GT06 protocol typically starts with 0x78 0x78
const (
	startByte1 = 0x78
	startByte2 = 0x78
)

func (d *Decoder) Decode(data []byte) (*GT06Data, error) {
	if len(data) < 15 {
		return nil, errors.New("data too short for GT06 protocol")
	}

	// Verify protocol header
	if data[0] != startByte1 || data[1] != startByte2 {
		return nil, errors.New("invalid GT06 protocol header")
	}

	reader := bytes.NewReader(data[3:]) // Skip header and length byte
	
	var result GT06Data
	
	// This is a simplified decoder. In production, implement full GT06 protocol
	err := binary.Read(reader, binary.BigEndian, &result.Latitude)
	if err != nil {
		return nil, err
	}
	
	err = binary.Read(reader, binary.BigEndian, &result.Longitude)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (d *Decoder) ToPosition(deviceID string, data *GT06Data) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Protocol = "gt06"
	return position
}
