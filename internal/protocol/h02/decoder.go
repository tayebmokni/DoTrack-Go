// Package h02 implements H02 GPS protocol decoder
package h02

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"tracking/internal/core/model"
)

// Common H02 errors
var (
	ErrInvalidHeader      = errors.New("invalid H02 protocol header")
	ErrPacketTooShort     = errors.New("data too short for H02 protocol")
	ErrInvalidFormat      = errors.New("invalid H02 data format")
	ErrInvalidCoordinate  = errors.New("invalid coordinate value")
	ErrInvalidMessageType = errors.New("unsupported message type")
	ErrMalformedPacket    = errors.New("malformed packet structure")
)

// H02 protocol constants
const (
	startSequence = "*HQ"
	minLength     = 20

	// Message types
	infoReport   = "V1"
	alarmReport  = "V2"
	statusReport = "V3"

	// Alarm types
	sosAlarm        = "0"
	powerCutAlarm   = "1"
	lowBatteryAlarm = "2"
	overspeedAlarm  = "3"
	geoFenceAlarm   = "4"
)

type Decoder struct {
	debug bool
}

func NewDecoder() *Decoder {
	return &Decoder{
		debug: false,
	}
}

func (d *Decoder) EnableDebug(enable bool) {
	d.debug = enable
}

func (d *Decoder) logDebug(format string, v ...interface{}) {
	if d.debug {
		log.Printf("[H02] "+format, v...)
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

func (d *Decoder) Decode(data []byte) (*H02Data, error) {
	d.logDebug("Starting packet decode...")
	d.logPacket(data, "Received")

	if len(data) < minLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d",
			ErrPacketTooShort, len(data), minLength)
	}

	dataStr := strings.TrimSpace(string(data))
	if !strings.HasPrefix(dataStr, "*HQ,") {
		return nil, fmt.Errorf("%w: expected *HQ,, got %s",
			ErrInvalidHeader, dataStr[:4])
	}

	// Remove start marker and split into fields
	dataStr = strings.TrimPrefix(dataStr, "*HQ,")
	dataStr = strings.TrimSuffix(dataStr, "#")
	parts := strings.Split(dataStr, ",")

	if len(parts) < 3 {
		return nil, fmt.Errorf("%w: insufficient fields", ErrInvalidFormat)
	}

	// Parse message type
	msgType := parts[0]
	d.logDebug("Message type: %s", msgType)

	switch msgType {
	case infoReport:
		return d.decodeInfoReport(parts[1:])
	case alarmReport:
		return d.decodeAlarmReport(parts[1:])
	case statusReport:
		return d.decodeStatusReport(parts[1:])
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidMessageType, msgType)
	}
}

func (d *Decoder) decodeInfoReport(parts []string) (*H02Data, error) {
	if len(parts) < 10 {
		return nil, fmt.Errorf("%w: info report requires at least 10 fields", ErrInvalidFormat)
	}

	result := &H02Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	// Parse GPS fix status
	if parts[1] != "A" {
		result.Valid = false
		d.logDebug("Invalid GPS fix status: %s", parts[1])
	}

	// Parse coordinates with validation
	var err error
	if result.Latitude, err = d.parseCoordinate(parts[2], parts[3]); err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}
	d.logDebug("Parsed coordinate %s%s to %.6f", parts[2], parts[3], result.Latitude)

	if result.Longitude, err = d.parseCoordinate(parts[4], parts[5]); err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}
	d.logDebug("Parsed coordinate %s%s to %.6f", parts[4], parts[5], result.Longitude)

	// Parse speed (convert knots to km/h)
	if speed, err := strconv.ParseFloat(parts[6], 64); err == nil {
		result.Speed = speed * 1.852 // Convert knots to km/h
	}

	// Parse course
	if course, err := strconv.ParseFloat(parts[7], 64); err == nil {
		result.Course = course
	}

	// Parse timestamp
	if ts, err := d.parseTimestamp(parts[8]); err == nil {
		result.Timestamp = ts
	} else {
		d.logDebug("Failed to parse timestamp: %v", err)
	}

	// Parse power level if available
	if len(parts) > 9 {
		if power, err := strconv.ParseUint(parts[9], 10, 8); err == nil {
			result.PowerLevel = uint8(power)
			result.Status["powerLevel"] = result.PowerLevel
		}
	}

	return result, nil
}

func (d *Decoder) decodeStatusReport(parts []string) (*H02Data, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("%w: status report requires at least 3 fields", ErrInvalidFormat)
	}

	result := &H02Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	// First field is power level
	if power, err := strconv.ParseUint(parts[1], 10, 8); err == nil {
		result.PowerLevel = uint8(power)
		result.Status["powerLevel"] = result.PowerLevel
	}

	// Second field is GSM signal
	if len(parts) > 2 && parts[2] != "" {
		if signal, err := strconv.ParseUint(parts[2], 10, 8); err == nil {
			result.GSMSignal = uint8(signal)
			result.Status["gsmSignal"] = result.GSMSignal
		}
	}

	// Parse status flags if present
	if len(parts) > 3 {
		statusFlags := parts[3]
		result.Status["charging"] = strings.Contains(statusFlags, "C")
		result.Status["engineOn"] = strings.Contains(statusFlags, "E")
	}

	return result, nil
}

func (d *Decoder) decodeAlarmReport(parts []string) (*H02Data, error) {
	// First parse location data
	result, err := d.decodeInfoReport(parts)
	if err != nil {
		return nil, err
	}

	// Parse alarm type (last field)
	if len(parts) > 0 {
		alarmCode := parts[len(parts)-1]
		d.logDebug("Alarm code: %s", alarmCode)

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

func (d *Decoder) parseCoordinate(coord, dir string) (float64, error) {
	val, err := strconv.ParseFloat(coord, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid format", ErrInvalidCoordinate)
	}

	// Extract degrees and minutes
	degrees := float64(int(val / 100))
	minutes := val - (degrees * 100)

	// Validate minutes
	if minutes >= 60 {
		return 0, fmt.Errorf("%w: invalid minutes value", ErrInvalidCoordinate)
	}

	// Convert to decimal degrees
	result := degrees + (minutes / 60.0)

	// Apply direction and validate range
	if dir == "S" || dir == "W" {
		result = -result
	}

	// Validate final range
	if (dir == "N" || dir == "S") && (result < -90 || result > 90) {
		return 0, fmt.Errorf("%w: latitude out of range", ErrInvalidCoordinate)
	}
	if (dir == "E" || dir == "W") && (result < -180 || result > 180) {
		return 0, fmt.Errorf("%w: longitude out of range", ErrInvalidCoordinate)
	}

	return result, nil
}

func (d *Decoder) parseTimestamp(date string) (time.Time, error) {
	if len(date) != 6 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", date)
	}

	// Parse date string in format DDMMYY
	day, err := strconv.Atoi(date[0:2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %s", date[0:2])
	}
	month, err := strconv.Atoi(date[2:4])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %s", date[2:4])
	}
	year, err := strconv.Atoi("20" + date[4:6])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year: %s", date[4:6])
	}

	// Validate ranges
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("invalid date values: day=%d, month=%d", day, month)
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
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

func (d *Decoder) ToPosition(deviceID string, data *H02Data) *model.Position {
	position := model.NewPosition(deviceID, data.Latitude, data.Longitude)
	position.Speed = data.Speed
	position.Course = data.Course
	position.Timestamp = data.Timestamp
	position.Protocol = "h02"

	// Add status information
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

func parsePowerLevel(power string) uint8 {
	if val, err := strconv.ParseUint(power, 10, 8); err == nil {
		if val > 100 {
			val = 100
		}
		return uint8(val)
	}
	return 0
}

func parseGSMSignal(signal string) uint8 {
	if val, err := strconv.ParseUint(signal, 10, 8); err == nil {
		if val > 31 {
			val = 31
		}
		return uint8(val)
	}
	return 0
}