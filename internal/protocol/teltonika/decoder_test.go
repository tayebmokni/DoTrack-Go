package teltonika

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestTeltonikaDecoder(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *TeltonikaData
		wantErr error
	}{
		{
			name: "valid packet with all fields",
			data: func() []byte {
				buf := new(bytes.Buffer)
				// Latitude: 37.7749° N
				binary.Write(buf, binary.BigEndian, 37.7749)
				// Longitude: -122.4194° W
				binary.Write(buf, binary.BigEndian, -122.4194)
				// Altitude: 100.5 meters
				binary.Write(buf, binary.BigEndian, float32(100.5))
				// Speed: 45.5 km/h (455 deci-km/h)
				binary.Write(buf, binary.BigEndian, uint16(455))
				// Course: 180.0 degrees
				binary.Write(buf, binary.BigEndian, uint16(180))
				return buf.Bytes()
			}(),
			want: &TeltonikaData{
				Valid:     true,
				Latitude:  37.7749,
				Longitude: -122.4194,
				Altitude:  100.5,
				Speed:    45.5,
				Course:   180.0,
				Status: map[string]interface{}{
					"altitude": float64(100.5),
					"speed":    float64(45.5),
					"course":   float64(180.0),
				},
			},
			wantErr: nil,
		},
		{
			name: "valid packet with minimum fields",
			data: func() []byte {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.BigEndian, 37.7749)
				binary.Write(buf, binary.BigEndian, -122.4194)
				return buf.Bytes()
			}(),
			want: &TeltonikaData{
				Valid:     true,
				Latitude:  37.7749,
				Longitude: -122.4194,
				Status:    map[string]interface{}{},
			},
			wantErr: nil,
		},
		{
			name: "packet too short",
			data: make([]byte, 8),
			want: nil,
			wantErr: ErrPacketTooShort,
		},
		{
			name: "invalid coordinates",
			data: func() []byte {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.BigEndian, 91.0)  // Invalid latitude
				binary.Write(buf, binary.BigEndian, 0.0)
				return buf.Bytes()
			}(),
			want: nil,
			wantErr: ErrInvalidCoordinate,
		},
		{
			name: "invalid course value",
			data: func() []byte {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.BigEndian, 37.7749)
				binary.Write(buf, binary.BigEndian, -122.4194)
				binary.Write(buf, binary.BigEndian, float32(100.5))
				binary.Write(buf, binary.BigEndian, uint16(455))
				binary.Write(buf, binary.BigEndian, uint16(361)) // Invalid course
				return buf.Bytes()
			}(),
			want: nil,
			wantErr: ErrInvalidValue,
		},
		{
			name: "NaN coordinates",
			data: func() []byte {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.BigEndian, math.NaN())
				binary.Write(buf, binary.BigEndian, -122.4194)
				return buf.Bytes()
			}(),
			want: nil,
			wantErr: ErrInvalidCoordinate,
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
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("Decode() expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Decode() unexpected error: %v", err)
				return
			}

			compareTeltonikaData(t, got, tt.want)
		})
	}
}

func compareTeltonikaData(t *testing.T, got, want *TeltonikaData) {
	if got.Valid != want.Valid {
		t.Errorf("Valid = %v, want %v", got.Valid, want.Valid)
	}
	if !almostEqual(got.Latitude, want.Latitude, 0.0001) {
		t.Errorf("Latitude = %v, want %v", got.Latitude, want.Latitude)
	}
	if !almostEqual(got.Longitude, want.Longitude, 0.0001) {
		t.Errorf("Longitude = %v, want %v", got.Longitude, want.Longitude)
	}
	if !almostEqual(got.Altitude, want.Altitude, 0.1) {
		t.Errorf("Altitude = %v, want %v", got.Altitude, want.Altitude)
	}
	if !almostEqual(got.Speed, want.Speed, 0.1) {
		t.Errorf("Speed = %v, want %v", got.Speed, want.Speed)
	}
	if !almostEqual(got.Course, want.Course, 0.1) {
		t.Errorf("Course = %v, want %v", got.Course, want.Course)
	}

	// Compare status fields
	for k, wantV := range want.Status {
		if gotV, ok := got.Status[k]; !ok {
			t.Errorf("Status missing key %s", k)
		} else {
			switch v := wantV.(type) {
			case float64:
				if !almostEqual(gotV.(float64), v, 0.1) {
					t.Errorf("Status[%s] = %v, want %v", k, gotV, v)
				}
			default:
				if gotV != v {
					t.Errorf("Status[%s] = %v, want %v", k, gotV, v)
				}
			}
		}
	}
}

// Helper function for floating point comparison
func almostEqual(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon || (math.IsNaN(a) && math.IsNaN(b))
}