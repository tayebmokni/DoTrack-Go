package gt06

import (
	"fmt"
	"testing"
	"time"
)

const (
	StartByte1 = 0x78
	StartByte2 = 0x78
	EndByte1   = 0x0D
	EndByte2   = 0x0A
	LocationMsg = 0x12
	StatusMsg   = 0x13
	AlarmMsg    = 0x16
	SosAlarm    = 0x01
)

func TestCompareDecoders(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "valid location packet",
			data: []byte{
				StartByte1, StartByte2, // Start bytes
				0x11,                   // Packet length
				LocationMsg,            // Protocol number (location)
				0x0F,                   // GPS status
				0x12, 0x34, 0x56, 0x78, // Latitude
				0x09, 0x10, 0x20, 0x30, // Longitude
				0x28,                   // Speed
				0x01, 0x44,             // Course
				0x23, 0x02, 0x14,       // Date
				0x12, 0x15, 0x13,       // Time
				0x00, 0x12,             // Checksum
				EndByte1, EndByte2,     // End bytes
			},
			wantErr: false,
		},
		{
			name: "valid status message",
			data: []byte{
				StartByte1, StartByte2, // Start bytes
				0x0A,                   // Packet length
				StatusMsg,              // Protocol number (status)
				0x45,                   // Status (Power=4, GSM=5)
				0x00, 0x01,             // Serial number
				0x00, 0x01,             // Error check
				0x00, 0x46,             // Checksum
				EndByte1, EndByte2,     // End bytes
			},
			wantErr: false,
		},
		{
			name: "valid alarm message",
			data: []byte{
				StartByte1, StartByte2, // Start bytes
				0x11,                   // Packet length
				AlarmMsg,               // Protocol number (alarm)
				0x0F,                   // GPS status
				0x12, 0x34, 0x56, 0x78, // Latitude
				0x09, 0x10, 0x20, 0x30, // Longitude
				0x28,                   // Speed
				0x01, 0x44,             // Course
				0x23, 0x02, 0x14,       // Date
				0x12, 0x15, 0x13,       // Time
				SosAlarm,               // Alarm type
				0x00, 0x13,             // Checksum
				EndByte1, EndByte2,     // End bytes
			},
			wantErr: false,
		},
	}

	v1Decoder := NewDecoder()
	v1Decoder.EnableDebug(true)

	v2Decoder := NewDecoderV2()
	v2Decoder.EnableDebug(true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Decode with both implementations
			v1Result, v1Err := v1Decoder.Decode(tt.data)
			v2Result, v2Err := v2Decoder.Decode(tt.data)

			// Compare error conditions
			if (v1Err != nil) != tt.wantErr {
				t.Errorf("V1 Decode() error = %v, wantErr %v", v1Err, tt.wantErr)
			}
			if (v2Err != nil) != tt.wantErr {
				t.Errorf("V2 Decode() error = %v, wantErr %v", v2Err, tt.wantErr)
			}

			// If we don't expect an error, compare results
			if !tt.wantErr {
				if v1Err != nil || v2Err != nil {
					t.Errorf("Unexpected decode error: v1=%v, v2=%v", v1Err, v2Err)
					return
				}

				compareGT06Data(t, v1Result, v2Result)
			}
		})
	}
}

func compareGT06Data(t *testing.T, v1, v2 *GT06Data) {
	if v1.Valid != v2.Valid {
		t.Errorf("Valid mismatch: v1=%v, v2=%v", v1.Valid, v2.Valid)
	}
	if v1.GPSValid != v2.GPSValid {
		t.Errorf("GPSValid mismatch: v1=%v, v2=%v", v1.GPSValid, v2.GPSValid)
	}
	if v1.Satellites != v2.Satellites {
		t.Errorf("Satellites mismatch: v1=%d, v2=%d", v1.Satellites, v2.Satellites)
	}
	if !almostEqual(v1.Latitude, v2.Latitude, 0.0001) {
		t.Errorf("Latitude mismatch: v1=%v, v2=%v", v1.Latitude, v2.Latitude)
	}
	if !almostEqual(v1.Longitude, v2.Longitude, 0.0001) {
		t.Errorf("Longitude mismatch: v1=%v, v2=%v", v1.Longitude, v2.Longitude)
	}
	if !almostEqual(v1.Speed, v2.Speed, 0.1) {
		t.Errorf("Speed mismatch: v1=%v, v2=%v", v1.Speed, v2.Speed)
	}
	if !almostEqual(v1.Course, v2.Course, 0.1) {
		t.Errorf("Course mismatch: v1=%v, v2=%v", v1.Course, v2.Course)
	}

	// Compare non-zero timestamps
	if !v1.Timestamp.IsZero() && !v2.Timestamp.IsZero() {
		if !v1.Timestamp.Equal(v2.Timestamp) {
			t.Errorf("Timestamp mismatch: v1=%v, v2=%v",
				v1.Timestamp.Format(time.RFC3339),
				v2.Timestamp.Format(time.RFC3339))
		}
	}

	// Compare status maps
	compareStatusMaps(t, v1.Status, v2.Status)
}

func compareStatusMaps(t *testing.T, v1, v2 map[string]interface{}) {
	// Check all keys in v1 exist in v2 with same values
	for k, v1Val := range v1 {
		v2Val, exists := v2[k]
		if !exists {
			t.Errorf("Status key %q missing in v2", k)
			continue
		}

		if fmt.Sprintf("%v", v1Val) != fmt.Sprintf("%v", v2Val) {
			t.Errorf("Status[%q] mismatch: v1=%v, v2=%v", k, v1Val, v2Val)
		}
	}

	// Check for extra keys in v2
	for k := range v2 {
		if _, exists := v1[k]; !exists {
			t.Errorf("Extra status key %q in v2", k)
		}
	}
}

func almostEqual(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < tolerance
}

type GT06Data struct {
	Valid       bool
	GPSValid    bool
	Satellites  int
	Latitude    float64
	Longitude   float64
	Speed       float64
	Course      float64
	PowerLevel  int
	GSMSignal   int
	Alarm       string
	Timestamp   time.Time
	Status      map[string]interface{}
}

func NewDecoder() *Decoder { return &Decoder{} }
func NewDecoderV2() *DecoderV2 { return &DecoderV2{} }
type Decoder struct{}
type DecoderV2 struct{}

func (d *Decoder) Decode(data []byte) (*GT06Data, error) {
	//Implement your decode logic here
	return &GT06Data{}, nil
}
func (d *DecoderV2) Decode(data []byte) (*GT06Data, error) {
	//Implement your decode logic here
	return &GT06Data{}, nil
}
func (d *Decoder) EnableDebug(debug bool) {}
func (d *DecoderV2) EnableDebug(debug bool) {}