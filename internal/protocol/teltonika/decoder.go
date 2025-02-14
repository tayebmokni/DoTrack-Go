package teltonika

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

type TeltonikaData struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	Speed     float64
	Course    float64
	Timestamp time.Time
	Valid     bool
}

func (d *Decoder) Decode(data []byte) (*TeltonikaData, error) {
	if len(data) < 16 {
		return nil, errors.New("data too short for Teltonika protocol")
	}

	reader := bytes.NewReader(data)
	result := &TeltonikaData{
		Timestamp: time.Now(),
		Valid:     true,
	}

	// Teltonika uses IEEE 754 double-precision format for coordinates
	if err := binary.Read(reader, binary.BigEndian, &result.Latitude); err != nil {
		return nil, err
	}

	if err := binary.Read(reader, binary.BigEndian, &result.Longitude); err != nil {
		return nil, err
	}

	// Read optional fields if available
	if reader.Len() >= 4 {
		var altitude float32
		if err := binary.Read(reader, binary.BigEndian, &altitude); err != nil {
			return nil, err
		}
		result.Altitude = float64(altitude)
	}

	if reader.Len() >= 2 {
		var speed uint16
		if err := binary.Read(reader, binary.BigEndian, &speed); err != nil {
			return nil, err
		}
		result.Speed = float64(speed) / 10.0 // Convert to km/h
	}

	if reader.Len() >= 2 {
		var course uint16
		if err := binary.Read(reader, binary.BigEndian, &course); err != nil {
			return nil, err
		}
		result.Course = float64(course)
	}

	// Validate coordinates
	if !isValidCoordinate(result.Latitude, result.Longitude) {
		return nil, errors.New("invalid coordinates")
	}

	return result, nil
}

func isValidCoordinate(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func (d *Decoder) ToPosition(deviceID string, data *TeltonikaData) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Altitude = data.Altitude
	position.Protocol = "teltonika"
	return position
}