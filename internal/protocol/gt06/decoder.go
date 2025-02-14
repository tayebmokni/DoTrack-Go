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
	PowerLevel uint8
	GSMSignal  uint8
	Alarm      string
	Status     map[string]interface{}
}

// GT06 protocol constants
const (
	startByte1 = 0x78
	startByte2 = 0x78
	minLength  = 15
	endByte1   = 0x0D // Split endByte into two constants
	endByte2   = 0x0A

	// Command types
	loginMsg    = 0x01
	locationMsg = 0x12
	statusMsg   = 0x13
	alarmMsg    = 0x16

	// Response types
	loginResp    = 0x05
	locationResp = 0x13
	alarmResp    = 0x15

	// Alarm types
	sosAlarm          = 0x01
	powerCutAlarm     = 0x02
	vibrationAlarm    = 0x04
	fenceInAlarm      = 0x10
	fenceOutAlarm     = 0x11
	lowBatteryAlarm   = 0x20
	overspeedAlarm    = 0x40
)

// GenerateResponse generates the appropriate response packet based on the received message type
func (d *Decoder) GenerateResponse(msgType uint8, deviceID string) []byte {
	switch msgType {
	case loginMsg:
		return d.generateLoginResponse(deviceID)
	case locationMsg:
		return d.generateLocationResponse()
	case alarmMsg:
		return d.generateAlarmResponse()
	default:
		return d.generateLocationResponse() // Default to location response
	}
}

func (d *Decoder) generateLoginResponse(deviceID string) []byte {
	// Login response format:
	// Start(2) + PackLen(1) + ProtocolNo(1) + DeviceID(n) + Time(2) + SerialNo(2) + Error(2) + CRC(2) + Stop(2)
	respLen := len(deviceID) + 10 // Add length of fixed fields
	resp := make([]byte, 0, respLen+5)

	// Start bytes
	resp = append(resp, startByte1, startByte2)

	// Packet length
	resp = append(resp, byte(respLen))

	// Protocol number (login response)
	resp = append(resp, loginResp)

	// Device ID (copy from request)
	resp = append(resp, []byte(deviceID)...)

	// Current time (UTC)
	now := time.Now().UTC()
	resp = append(resp, byte(now.Hour()), byte(now.Minute()))

	// Serial number (0x0001)
	resp = append(resp, 0x00, 0x01)

	// Error code (0x0000 = success)
	resp = append(resp, 0x00, 0x00)

	// Calculate CRC
	crc := calculateCRC(resp[2:])
	resp = append(resp, byte(crc>>8), byte(crc))

	// End bytes
	resp = append(resp, endByte1, endByte2)

	return resp
}

func (d *Decoder) generateLocationResponse() []byte {
	resp := []byte{
		startByte1, startByte2, // Start bytes
		0x05,                   // Packet length
		locationResp,           // Protocol number (location response)
		0x00, 0x01,            // Serial number
		0x00, 0x01,            // CRC
		endByte1, endByte2,    // End bytes
	}
	return resp
}

func (d *Decoder) generateAlarmResponse() []byte {
	resp := []byte{
		startByte1, startByte2, // Start bytes
		0x05,                   // Packet length
		alarmResp,              // Protocol number (alarm response)
		0x00, 0x01,            // Serial number
		0x00, 0x01,            // CRC
		endByte1, endByte2,    // End bytes
	}
	return resp
}

func calculateCRC(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b)
	}
	return crc
}

func (d *Decoder) Decode(data []byte) (*GT06Data, error) {
	if len(data) < minLength {
		return nil, errors.New("data too short for GT06 protocol")
	}

	if data[0] != startByte1 || data[1] != startByte2 {
		return nil, errors.New("invalid GT06 protocol header")
	}

	if !d.validateChecksum(data) {
		return nil, errors.New("invalid checksum")
	}

	reader := bytes.NewReader(data[3:])
	var protocolNumber uint8
	if err := binary.Read(reader, binary.BigEndian, &protocolNumber); err != nil {
		return nil, err
	}

	// Process based on message type
	switch protocolNumber {
	case loginMsg:
		return d.decodeLoginMessage(reader)
	case locationMsg:
		return d.decodeLocationMessage(reader)
	case statusMsg:
		return d.decodeStatusMessage(reader)
	case alarmMsg:
		return d.decodeAlarmMessage(reader)
	default:
		return nil, fmt.Errorf("unsupported message type: %02x", protocolNumber)
	}
}

func (d *Decoder) decodeLocationMessage(reader *bytes.Reader) (*GT06Data, error) {
	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	var statusByte uint8
	if err := binary.Read(reader, binary.BigEndian, &statusByte); err != nil {
		return nil, err
	}

	result.GPSValid = (statusByte&0x01) == 0x01
	result.Satellites = (statusByte >> 2) & 0x0F

	var rawLat, rawLon uint32
	if err := binary.Read(reader, binary.BigEndian, &rawLat); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &rawLon); err != nil {
		return nil, err
	}

	var err error
	if result.Latitude, err = bcdToFloat(rawLat); err != nil {
		return nil, fmt.Errorf("invalid latitude: %v", err)
	}
	if result.Longitude, err = bcdToFloat(rawLon); err != nil {
		return nil, fmt.Errorf("invalid longitude: %v", err)
	}

	var speed uint8
	if err := binary.Read(reader, binary.BigEndian, &speed); err != nil {
		return nil, err
	}
	result.Speed = float64(speed)

	var course uint16
	if err := binary.Read(reader, binary.BigEndian, &course); err != nil {
		return nil, err
	}
	result.Course = float64(course)

	result.Timestamp, err = d.parseTimestamp(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %v", err)
	}

	return result, nil
}

func (d *Decoder) decodeAlarmMessage(reader *bytes.Reader) (*GT06Data, error) {
	locationData, err := d.decodeLocationMessage(reader)
	if err != nil {
		return nil, err
	}

	var alarmType uint8
	if err := binary.Read(reader, binary.BigEndian, &alarmType); err != nil {
		return nil, err
	}

	// Set alarm type based on the received code
	switch alarmType {
	case sosAlarm:
		locationData.Alarm = "sos"
	case powerCutAlarm:
		locationData.Alarm = "powerCut"
	case vibrationAlarm:
		locationData.Alarm = "vibration"
	case fenceInAlarm:
		locationData.Alarm = "geofenceEnter"
	case fenceOutAlarm:
		locationData.Alarm = "geofenceExit"
	case lowBatteryAlarm:
		locationData.Alarm = "lowBattery"
	case overspeedAlarm:
		locationData.Alarm = "overspeed"
	default:
		locationData.Alarm = fmt.Sprintf("unknown_%02x", alarmType)
	}

	return locationData, nil
}

func (d *Decoder) decodeStatusMessage(reader *bytes.Reader) (*GT06Data, error) {
	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	var statusByte uint8
	if err := binary.Read(reader, binary.BigEndian, &statusByte); err != nil {
		return nil, err
	}

	result.PowerLevel = (statusByte >> 4) & 0x0F
	result.GSMSignal = statusByte & 0x0F

	result.Status["powerLevel"] = result.PowerLevel
	result.Status["gsmSignal"] = result.GSMSignal
	result.Status["charging"] = (statusByte&0x20 != 0)
	result.Status["engineOn"] = (statusByte&0x40 != 0)

	return result, nil
}

func (d *Decoder) decodeLoginMessage(reader *bytes.Reader) (*GT06Data, error) {
	result := &GT06Data{
		Valid:  true,
		Status: make(map[string]interface{}),
	}

	// Read IMEI (8 bytes)
	imei := make([]byte, 8)
	if _, err := reader.Read(imei); err != nil {
		return nil, err
	}
	result.Status["imei"] = fmt.Sprintf("%x", imei)

	return result, nil
}

func (d *Decoder) validateChecksum(data []byte) bool {
	if len(data) < 3 {
		return false
	}

	length := int(data[2])
	if len(data) < length+5 {
		return false
	}

	checksum := uint16(0)
	for i := 2; i < length+3; i++ {
		checksum ^= uint16(data[i])
	}

	packetChecksum := binary.BigEndian.Uint16(data[length+3 : length+5])
	return checksum == packetChecksum
}

func bcdToFloat(bcd uint32) (float64, error) {
	deg := float64((bcd>>20)&0xF)*10 + float64((bcd>>16)&0xF)
	min := float64((bcd>>12)&0xF)*10 + float64((bcd>>8)&0xF) +
		(float64((bcd>>4)&0xF)*10 + float64(bcd&0xF))/100.0

	if deg > 90 || min >= 60 {
		return 0, errors.New("invalid BCD coordinate value")
	}

	return deg + (min / 60.0), nil
}

func (d *Decoder) parseTimestamp(reader *bytes.Reader) (time.Time, error) {
	var timeBytes [6]byte
	if _, err := reader.Read(timeBytes[:]); err != nil {
		return time.Time{}, err
	}

	year := 2000 + ((int(timeBytes[0])>>4)*10 + int(timeBytes[0]&0x0F))
	month := (int(timeBytes[1])>>4)*10 + int(timeBytes[1]&0x0F)
	day := (int(timeBytes[2])>>4)*10 + int(timeBytes[2]&0x0F)
	hour := (int(timeBytes[3])>>4)*10 + int(timeBytes[3]&0x0F)
	minute := (int(timeBytes[4])>>4)*10 + int(timeBytes[4]&0x0F)
	second := (int(timeBytes[5])>>4)*10 + int(timeBytes[5]&0x0F)

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

	// Add additional status information
	if data.Alarm != "" {
		position.Status = map[string]interface{}{
			"alarm":      data.Alarm,
			"powerLevel": data.PowerLevel,
			"gsmSignal":  data.GSMSignal,
		}
		// Add all status fields
		for k, v := range data.Status {
			position.Status[k] = v
		}
	}

	return position
}