package h02

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

type H02Data struct {
	Latitude  float64
	Longitude float64
	Speed     float64
	Course    float64
	Timestamp int64
}

// The H02 protocol typically starts with *HQ
const startSequence = "*HQ"

func (d *Decoder) Decode(data []byte) (*H02Data, error) {
	if len(data) < 10 {
		return nil, errors.New("data too short for H02 protocol")
	}

	// Verify protocol header
	if !bytes.HasPrefix(data, []byte(startSequence)) {
		return nil, errors.New("invalid H02 protocol header")
	}

	reader := bytes.NewReader(data[4:]) // Skip header
	
	var result H02Data
	
	// This is a simplified decoder. In production, implement full H02 protocol
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

func (d *Decoder) ToPosition(deviceID string, data *H02Data) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Protocol = "h02"
	return position
}
