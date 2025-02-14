package gt06

import (
	"testing"
	"time"
)

func TestGT06Decoder(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *GT06Data
		wantErr bool
	}{
		{
			name: "valid location packet",
			data: []byte{
				0x78, 0x78, // Start bytes
				0x11,       // Packet length
				0x12,       // Protocol number (location)
				0x0F,       // GPS status
				0x0C, 0x46, 0x58, 0x1E, // Latitude
				0x6E, 0x31, 0x22, 0x50, // Longitude
				0x28,             // Speed
				0x01, 0x44,       // Course
				0x23, 0x02, 0x14, // Date (YY-MM-DD)
				0x12, 0x15, 0x13, // Time (HH-MM-SS)
				0x06, 0x0D,       // Checksum
				0x0D, 0x0A,       // End bytes
			},
			want: &GT06Data{
				Valid:      true,
				GPSValid:   true,
				Satellites: 3,
				Speed:      40.0,
				Course:    324.0,
			},
			wantErr: false,
		},
		{
			name: "invalid header",
			data: []byte{0x77, 0x77, 0x00},
			want: nil,
			wantErr: true,
		},
		{
			name: "packet too short",
			data: []byte{0x78, 0x78},
			want: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder()
			decoder.EnableDebug(true) // Enable debug logging for tests

			got, err := decoder.Decode(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.Valid != tt.want.Valid {
				t.Errorf("Valid = %v, want %v", got.Valid, tt.want.Valid)
			}
			if got.GPSValid != tt.want.GPSValid {
				t.Errorf("GPSValid = %v, want %v", got.GPSValid, tt.want.GPSValid)
			}
			if got.Satellites != tt.want.Satellites {
				t.Errorf("Satellites = %v, want %v", got.Satellites, tt.want.Satellites)
			}
			if got.Speed != tt.want.Speed {
				t.Errorf("Speed = %v, want %v", got.Speed, tt.want.Speed)
			}
			if got.Course != tt.want.Course {
				t.Errorf("Course = %v, want %v", got.Course, tt.want.Course)
			}
		})
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
			bcd:     0x12345678,
			want:    12.5845,
			wantErr: false,
		},
		{
			name:    "invalid degrees",
			bcd:     0x95345678,
			wantErr: true,
		},
		{
			name:    "invalid minutes",
			bcd:     0x12645678,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bcdToFloat(tt.bcd)
			if (err != nil) != tt.wantErr {
				t.Errorf("bcdToFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("bcdToFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}
