package h02

import (
	"bytes"
	"errors"
	"fmt"
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
	Latitude   float64
	Longitude  float64
	Speed     float64
	Course    float64
	Timestamp time.Time
	Valid     bool
	PowerLevel uint8
	GSMSignal  uint8
	Alarm      string
	Status     map[string]interface{}
}

// H02 protocol constants
const (
	startSequence = "*HQ"
	minLength     = 20

	// H02 protocol message types
	infoReport   = "V1"
	alarmReport  = "V2"
	statusReport = "V3"

	// H02 alarm types
	sosAlarm        = "0"
	powerCutAlarm   = "1"
	lowBatteryAlarm = "2"
	overspeedAlarm  = "3"
	geoFenceAlarm   = "4"
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

	// Parse message type from parts[0] (e.g., "*HQ,V1,...")
	msgType := strings.TrimPrefix(parts[0], "*HQ,")

	switch msgType {
	case infoReport:
		return d.decodeInfoReport(parts)
	case alarmReport:
		return d.decodeAlarmReport(parts)
	case statusReport:
		return d.decodeStatusReport(parts)
	default:
		// Default to info report for backward compatibility
		return d.decodeInfoReport(parts)
	}
}

func (d *Decoder) decodeInfoReport(parts []string) (*H02Data, error) {
	result := &H02Data{
		Valid:     true,
		Status:    make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	// Parse latitude (format: DDMM.MMMM)
	if lat, err := parseCoordinate(parts[2]); err == nil {
		result.Latitude = lat
	} else {
		return nil, fmt.Errorf("invalid latitude format: %v", err)
	}

	// Parse longitude (format: DDDMM.MMMM)
	if lon, err := parseCoordinate(parts[3]); err == nil {
		result.Longitude = lon
	} else {
		return nil, fmt.Errorf("invalid longitude format: %v", err)
	}

	// Parse speed (in knots, convert to km/h)
	if speed, err := strconv.ParseFloat(parts[4], 64); err == nil {
		result.Speed = speed * 1.852 // Convert knots to km/h
	}

	// Parse course (heading)
	if course, err := strconv.ParseFloat(parts[5], 64); err == nil {
		result.Course = course
	}

	// Add additional status information if available
	if len(parts) > 6 {
		result.GSMSignal = parseGSMSignal(parts[6])
		result.Status["gsmSignal"] = result.GSMSignal
	}

	if len(parts) > 7 {
		result.PowerLevel = parseBatteryLevel(parts[7])
		result.Status["powerLevel"] = result.PowerLevel
	}

	return result, nil
}

func (d *Decoder) decodeAlarmReport(parts []string) (*H02Data, error) {
	result, err := d.decodeInfoReport(parts)
	if err != nil {
		return nil, err
	}

	// Parse alarm type if available
	if len(parts) > 8 {
		alarmCode := parts[8]
		switch alarmCode {
		case sosAlarm:
			result.Alarm = "sos"
		case powerCutAlarm:
			result.Alarm = "powerCut"
		case lowBatteryAlarm:
			result.Alarm = "lowBattery"
		case overspeedAlarm:
			result.Alarm = "overspeed"
		case geoFenceAlarm:
			result.Alarm = "geofence"
		default:
			result.Alarm = fmt.Sprintf("unknown_%s", alarmCode)
		}
		result.Status["alarm"] = result.Alarm
	}

	return result, nil
}

func (d *Decoder) decodeStatusReport(parts []string) (*H02Data, error) {
	result := &H02Data{
		Valid:     true,
		Status:    make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	if len(parts) > 3 {
		result.GSMSignal = parseGSMSignal(parts[2])
		result.PowerLevel = parseBatteryLevel(parts[3])

		result.Status["gsmSignal"] = result.GSMSignal
		result.Status["powerLevel"] = result.PowerLevel

		// Parse additional status flags if available
		if len(parts) > 4 {
			statusFlags := parts[4]
			result.Status["charging"] = strings.Contains(statusFlags, "C")
			result.Status["engineOn"] = strings.Contains(statusFlags, "E")
		}
	}

	return result, nil
}

// Parse GSM signal strength (0-31)
func parseGSMSignal(signal string) uint8 {
	if val, err := strconv.ParseUint(signal, 10, 8); err == nil {
		if val > 31 {
			val = 31
		}
		return uint8(val)
	}
	return 0
}

// Parse battery level (0-100)
func parseBatteryLevel(battery string) uint8 {
	if val, err := strconv.ParseUint(battery, 10, 8); err == nil {
		if val > 100 {
			val = 100
		}
		return uint8(val)
	}
	return 0
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

	// Add additional status information
	position.Status = make(map[string]interface{})
	position.Status["powerLevel"] = data.PowerLevel
	position.Status["gsmSignal"] = data.GSMSignal

	if data.Alarm != "" {
		position.Status["alarm"] = data.Alarm
	}

	// Add all remaining status fields
	for k, v := range data.Status {
		if _, exists := position.Status[k]; !exists {
			position.Status[k] = v
		}
	}

	return position
}