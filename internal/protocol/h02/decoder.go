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
	ErrInvalidHeader     = errors.New("invalid H02 protocol header")
	ErrPacketTooShort    = errors.New("data too short for H02 protocol")
	ErrInvalidFormat     = errors.New("invalid H02 data format")
	ErrInvalidCoordinate = errors.New("invalid coordinate value")
	ErrInvalidMessageType = errors.New("unsupported message type")
	ErrMalformedPacket   = errors.New("malformed packet structure")
)

type Decoder struct {
	debug bool
}

func NewDecoder() *Decoder {
	return &Decoder{
		debug: false,
	}
}

// EnableDebug enables detailed logging for protocol parsing
func (d *Decoder) EnableDebug(enable bool) {
	d.debug = enable
}

// logDebug logs debug messages if debug mode is enabled
func (d *Decoder) logDebug(format string, v ...interface{}) {
	if d.debug {
		log.Printf("[H02] "+format, v...)
	}
}

// logPacket logs packet details in hexadecimal format
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

	// Convert to string and split into fields
	dataStr := strings.TrimSpace(string(data))
	if !strings.HasPrefix(dataStr, "*HQ,") {
		return nil, fmt.Errorf("%w: expected *HQ, got %s",
			ErrInvalidHeader, dataStr[:4])
	}

	// Remove start and end markers
	dataStr = strings.TrimPrefix(dataStr, "*HQ,")
	dataStr = strings.TrimSuffix(dataStr, "#")

	parts := strings.Split(dataStr, ",")
	if len(parts) < 3 { // Need at least message type and device ID
		return nil, fmt.Errorf("%w: insufficient fields (%d)",
			ErrInvalidFormat, len(parts))
	}

	// Parse message type (first field)
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
	if len(parts) < 12 {
		return nil, fmt.Errorf("%w: info report requires at least 12 fields", ErrInvalidFormat)
	}

	result := &H02Data{
		Valid:     true,
		Status:    make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	// Parse coordinates
	var err error
	if result.Latitude, err = d.parseCoordinate(parts[4], parts[5]); err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}

	if result.Longitude, err = d.parseCoordinate(parts[6], parts[7]); err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}

	// Parse speed (knots to km/h)
	if speed, err := strconv.ParseFloat(parts[8], 64); err == nil {
		result.Speed = speed * 1.852 // Convert knots to km/h
	}

	// Parse course (heading)
	if course, err := strconv.ParseFloat(parts[9], 64); err == nil {
		result.Course = course
	}

	// Parse timestamp if present
	if len(parts) > 10 {
		if ts, err := d.parseTimestamp(parts[10]); err == nil {
			result.Timestamp = ts
		}
	}

	// Parse additional status fields
	if len(parts) > 11 {
		result.PowerLevel = parsePowerLevel(parts[11])
		result.Status["powerLevel"] = result.PowerLevel
	}

	return result, nil
}

func (d *Decoder) parseCoordinate(coord, dir string) (float64, error) {
	val, err := strconv.ParseFloat(coord, 64)
	if err != nil {
		return 0, err
	}

	// Convert DDMM.MMMM to decimal degrees
	degrees := float64(int(val/100))
	minutes := val - degrees*100

	result := degrees + minutes/60.0

	// Apply direction
	if dir == "S" || dir == "W" {
		result = -result
	}

	// Validate ranges
	if (dir == "N" || dir == "S") && (result < -90 || result > 90) {
		return 0, ErrInvalidCoordinate
	}
	if (dir == "E" || dir == "W") && (result < -180 || result > 180) {
		return 0, ErrInvalidCoordinate
	}

	return result, nil
}

func (d *Decoder) parseTimestamp(date string) (time.Time, error) {
	// Parse date string in format YYMMDDHHMMSS
	if len(date) != 12 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", date)
	}

	year, _ := strconv.Atoi("20" + date[0:2])
	month, _ := strconv.Atoi(date[2:4])
	day, _ := strconv.Atoi(date[4:6])
	hour, _ := strconv.Atoi(date[6:8])
	minute, _ := strconv.Atoi(date[8:10])
	second, _ := strconv.Atoi(date[10:12])

	// Validate ranges
	if month < 1 || month > 12 || day < 1 || day > 31 ||
		hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, fmt.Errorf("invalid date/time values")
	}

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC), nil
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
		result.PowerLevel = parsePowerLevel(parts[3])

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

	// Add all remaining status fields
	for k, v := range data.Status {
		if _, exists := position.Status[k]; !exists {
			position.Status[k] = v
		}
	}

	return position
}