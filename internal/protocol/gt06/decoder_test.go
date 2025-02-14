package gt06

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestGT06Decoder(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *GT06Data
		wantErr error
	}{
		{
			name: "valid location packet",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x11,       // Packet length
				0x12,       // Protocol number (location)
				0x0F,       // GPS status
				0x12, 0x34, 0x56, 0x78, // Latitude
				0x09, 0x10, 0x20, 0x30, // Longitude
				0x28,             // Speed
				0x01, 0x44,       // Course
				0x23, 0x02, 0x14, // Date
				0x12, 0x15, 0x13, // Time
				0x00, 0x12,       // Checksum
				0x0D, 0x0A,       // End bytes
			},
			want: &GT06Data{
				Valid:      true,
				GPSValid:   true,
				Satellites: 3,
				Latitude:   12.5761333,
				Longitude:  91.0338333,
				Speed:     40.0,
				Course:    324.0,
				Timestamp: time.Date(2023, 2, 14, 12, 15, 13, 0, time.UTC),
				Status:    make(map[string]interface{}),
			},
			wantErr: nil,
		},
		{
			name: "valid status message",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x0A,       // Packet length
				0x13,       // Protocol number (status)
				0x45,       // Status (Power=4, GSM=5)
				0x00, 0x01, // Serial number
				0x00, 0x01, // Error check
				0x00, 0x46, // Checksum
				0x0D, 0x0A, // End bytes
			},
			want: &GT06Data{
				Valid:      true,
				PowerLevel: 4,
				GSMSignal:  5,
				Status: map[string]interface{}{
					"powerLevel": uint8(4),
					"gsmSignal":  uint8(5),
					"charging":   false,
					"engineOn":   false,
				},
			},
			wantErr: nil,
		},
		{
			name: "valid alarm message",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x11,       // Packet length
				0x16,       // Protocol number (alarm)
				0x0F,       // GPS status
				0x12, 0x34, 0x56, 0x78, // Latitude
				0x09, 0x10, 0x20, 0x30, // Longitude
				0x28,             // Speed
				0x01, 0x44,       // Course
				0x23, 0x02, 0x14, // Date
				0x12, 0x15, 0x13, // Time
				0x01,             // Alarm type (SOS)
				0x00, 0x13,       // Checksum
				0x0D, 0x0A,       // End bytes
			},
			want: &GT06Data{
				Valid:      true,
				GPSValid:   true,
				Satellites: 3,
				Latitude:   12.5761333,
				Longitude:  91.0338333,
				Speed:     40.0,
				Course:    324.0,
				Timestamp: time.Date(2023, 2, 14, 12, 15, 13, 0, time.UTC),
				Alarm:     "sos",
				Status:    make(map[string]interface{}),
			},
			wantErr: nil,
		},
		{
			name: "invalid header",
			data: []byte{0x77, 0x77, 0x00},
			want: nil,
			wantErr: ErrInvalidHeader,
		},
		{
			name: "packet too short",
			data: []byte{0x78, 0x78},
			want: nil,
			wantErr: ErrPacketTooShort,
		},
		{
			name: "invalid length",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x20,       // Incorrect length
				0x12,       // Protocol number
				0x00,       // Data byte
				0x00, 0x00, // Checksum
				0x0D, 0x0A, // End bytes
			},
			want: nil,
			wantErr: ErrInvalidLength,
		},
		{
			name: "invalid checksum",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x11,       // Packet length
				0x12,       // Protocol number
				0x0F,       // GPS status
				0x12, 0x34, 0x56, 0x78, // Latitude
				0x09, 0x10, 0x20, 0x30, // Longitude
				0x28,             // Speed
				0x01, 0x44,       // Course
				0x23, 0x02, 0x14, // Date
				0x12, 0x15, 0x13, // Time
				0xFF, 0xFF,       // Invalid checksum
				0x0D, 0x0A,       // End bytes
			},
			want: nil,
			wantErr: ErrInvalidChecksum,
		},
		{
			name: "malformed end bytes",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x11,       // Length
				0x12,       // Protocol (location)
				0x0F,       // GPS status
				0x12, 0x34, 0x56, 0x78, // Latitude
				0x09, 0x10, 0x20, 0x30, // Longitude
				0x28,             // Speed
				0x01, 0x44,       // Course
				0x23, 0x02, 0x14, // Date
				0x12, 0x15, 0x13, // Time
				0x00, 0x12,       // Checksum
				0x0D, 0x0C,       // Invalid end bytes
			},
			want:    nil,
			wantErr: ErrMalformedPacket,
		},
		{
			name: "invalid protocol number",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x05,       // Length
				0xFF,       // Invalid protocol number
				0x00, 0x00, // Data
				0xFF, 0x00, // Checksum
				0x0D, 0x0A, // End bytes
			},
			want:    nil,
			wantErr: ErrInvalidMessageType,
		},
		{
			name: "status message with invalid power level",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x0A,       // Length
				0x13,       // Protocol (status)
				0xF5,       // Invalid status (power=15, GSM=5)
				0x00, 0x01, // Serial
				0x00, 0x01, // Error check
				0x00, 0xF6, // Checksum
				0x0D, 0x0A, // End bytes
			},
			want:    nil,
			wantErr: ErrMalformedPacket,
		},
		{
			name: "alarm message with unknown type",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x11,       // Length
				0x16,       // Protocol (alarm)
				0x0F,       // GPS status
				0x12, 0x34, 0x56, 0x78, // Latitude
				0x09, 0x10, 0x20, 0x30, // Longitude
				0x28,             // Speed
				0x01, 0x44,       // Course
				0x23, 0x02, 0x14, // Date
				0x12, 0x15, 0x13, // Time
				0xFF,             // Unknown alarm type
				0x00, 0x1C,       // Checksum
				0x0D, 0x0A,       // End bytes
			},
			want: &GT06Data{
				Valid:      true,
				GPSValid:   true,
				Satellites: 3,
				Latitude:   12.5761333,
				Longitude:  91.0338333,
				Speed:     40.0,
				Course:    324.0,
				Timestamp: time.Date(2023, 2, 14, 12, 15, 13, 0, time.UTC),
				Alarm:     "unknown_ff",
				Status:    make(map[string]interface{}),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder()
			decoder.EnableDebug(true)

			got, err := decoder.Decode(tt.data)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Decode() expected error %v, got nil", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("Decode() expected error containing %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Decode() unexpected error: %v", err)
				return
			}

			compareGT06Data(t, got, tt.want)
		})
	}
}

func compareGT06Data(t *testing.T, got, want *GT06Data) {
	if got.Valid != want.Valid {
		t.Errorf("Valid = %v, want %v", got.Valid, want.Valid)
	}
	if got.GPSValid != want.GPSValid {
		t.Errorf("GPSValid = %v, want %v", got.GPSValid, want.GPSValid)
	}
	if got.Satellites != want.Satellites {
		t.Errorf("Satellites = %v, want %v", got.Satellites, want.Satellites)
	}
	if !almostEqual(got.Latitude, want.Latitude, 0.0001) {
		t.Errorf("Latitude = %v, want %v", got.Latitude, want.Latitude)
	}
	if !almostEqual(got.Longitude, want.Longitude, 0.0001) {
		t.Errorf("Longitude = %v, want %v", got.Longitude, want.Longitude)
	}
	if !almostEqual(got.Speed, want.Speed, 0.1) {
		t.Errorf("Speed = %v, want %v", got.Speed, want.Speed)
	}
	if !almostEqual(got.Course, want.Course, 0.1) {
		t.Errorf("Course = %v, want %v", got.Course, want.Course)
	}
	if want.Timestamp.IsZero() && !got.Timestamp.IsZero() {
		t.Errorf("Expected zero timestamp, got %v", got.Timestamp)
	} else if !want.Timestamp.IsZero() && !got.Timestamp.Equal(want.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", got.Timestamp, want.Timestamp)
	}
	if got.Alarm != want.Alarm {
		t.Errorf("Alarm = %v, want %v", got.Alarm, want.Alarm)
	}
}

func TestBCDToFloat(t *testing.T) {
	tests := []struct {
		name    string
		bcd     uint32
		want    float64
		wantErr bool
	}{
		{
			name:    "valid coordinate",
			bcd:     0x12345678, // 12°34.5678'
			want:    12.5761333,
			wantErr: false,
		},
		{
			name:    "invalid BCD digit",
			bcd:     0xA2345678, // First digit is A (invalid BCD)
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid degrees",
			bcd:     0x95345678, // 95 degrees is invalid
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid minutes",
			bcd:     0x12645678, // 64 minutes is invalid
			want:    0,
			wantErr: true,
		},
		{
			name:    "valid coordinate near limit",
			bcd:     0x89595959, // 89°59.5959'
			want:    89.9932639,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bcdToFloat(tt.bcd)
			if (err != nil) != tt.wantErr {
				t.Errorf("bcdToFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !almostEqual(got, tt.want, 0.0001) {
				t.Errorf("bcdToFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for floating point comparison
func almostEqual(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    time.Time
		wantErr bool
	}{
		{
			name: "valid timestamp",
			data: []byte{0x23, 0x02, 0x14, 0x12, 0x15, 0x13}, // 2023-02-14 12:15:13
			want: time.Date(2023, 2, 14, 12, 15, 13, 0, time.UTC),
			wantErr: false,
		},
		{
			name: "invalid month",
			data: []byte{0x23, 0x13, 0x14, 0x12, 0x15, 0x13}, // Month 13 is invalid
			want: time.Time{},
			wantErr: true,
		},
		{
			name: "invalid hour",
			data: []byte{0x23, 0x02, 0x14, 0x24, 0x15, 0x13}, // Hour 24 is invalid
			want: time.Time{},
			wantErr: true,
		},
	}

	decoder := NewDecoder()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			got, err := decoder.parseTimestamp(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}