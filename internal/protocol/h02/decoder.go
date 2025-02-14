package h02

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"time"
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
	Timestamp time.Time
	Valid     bool
}

// H02 protocol constants
const (
	startSequence = "*HQ"
	minLength     = 20
)

func (d *Decoder) Decode(data []byte) (*H02Data, error) {
	if len(data) < minLength {
		return nil, errors.New("data too short for H02 protocol")
	}

	// Verify protocol header
	if !bytes.HasPrefix(data, []byte(startSequence)) {
		return nil, errors.New("invalid H02 protocol header")
	}

	// Convert to string for easier parsing since H02 uses ASCII format
	dataStr := string(data)
	parts := strings.Split(dataStr, ",")
	if len(parts) < 7 {
		return nil, errors.New("invalid H02 data format")
	}

	result := &H02Data{
		Timestamp: time.Now(), // Default to current time if parsing fails
		Valid:     true,
	}

	// Parse latitude (format: DDMM.MMMM)
	if lat, err := parseCoordinate(parts[2]); err == nil {
		result.Latitude = lat
	} else {
		return nil, errors.New("invalid latitude format")
	}

	// Parse longitude (format: DDDMM.MMMM)
	if lon, err := parseCoordinate(parts[3]); err == nil {
		result.Longitude = lon
	} else {
		return nil, errors.New("invalid longitude format")
	}

	// Parse speed (in knots, convert to km/h)
	if speed, err := strconv.ParseFloat(parts[4], 64); err == nil {
		result.Speed = speed * 1.852 // Convert knots to km/h
	}

	// Parse course (heading)
	if course, err := strconv.ParseFloat(parts[5], 64); err == nil {
		result.Course = course
	}

	return result, nil
}

// parseCoordinate converts DDMM.MMMM format to decimal degrees
func parseCoordinate(coord string) (float64, error) {
	if len(coord) < 6 {
		return 0, errors.New("coordinate string too short")
	}

	// Split into degrees and minutes
	degLen := 2
	if len(coord) > 7 { // Longitude has 3 degree digits
		degLen = 3
	}

	degrees, err := strconv.ParseFloat(coord[:degLen], 64)
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.ParseFloat(coord[degLen:], 64)
	if err != nil {
		return 0, err
	}

	return degrees + (minutes / 60.0), nil
}

func (d *Decoder) ToPosition(deviceID string, data *H02Data) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Protocol = "h02"
	return position
}