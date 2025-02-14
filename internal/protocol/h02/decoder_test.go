package h02

import (
	"strings"
	"testing"
	"time"
)

func TestH02Decoder(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *H02Data
		wantErr error
	}{
		{
			name: "valid info report",
			data: []byte("*HQ,V1,123456789012345,A,2237.7514,N,11408.6214,E,6,2,151022,10,1,6#"),
			want: &H02Data{
				Valid:      true,
				Latitude:   22.62919,
				Longitude:  114.14369,
				Speed:     11.112,  // 6 knots * 1.852
				Course:    2.0,
				Timestamp: time.Date(2022, 10, 15, 0, 0, 0, 0, time.UTC),
				PowerLevel: 10,
				Status: map[string]interface{}{
					"powerLevel": uint8(10),
				},
			},
			wantErr: nil,
		},
		{
			name: "valid alarm report",
			data: []byte("*HQ,V2,123456789012345,A,2237.7514,N,11408.6214,E,6,2,151022,0#"),
			want: &H02Data{
				Valid:      true,
				Latitude:   22.62919,
				Longitude:  114.14369,
				Speed:     11.112,
				Course:    2.0,
				Timestamp: time.Date(2022, 10, 15, 0, 0, 0, 0, time.UTC),
				Alarm:     "sos",
				Status: map[string]interface{}{
					"alarm": "sos",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid status report",
			data: []byte("*HQ,V3,123456789012345,45,5,CE#"),
			want: &H02Data{
				Valid:      true,
				PowerLevel: 45,
				GSMSignal:  5,
				Status: map[string]interface{}{
					"powerLevel": uint8(45),
					"gsmSignal":  uint8(5),
					"charging":   true,
					"engineOn":   true,
				},
			},
			wantErr: nil,
		},
		{
			name: "invalid header",
			data: []byte("*XX,V1,123456789012345"),
			want: nil,
			wantErr: ErrInvalidHeader,
		},
		{
			name: "packet too short",
			data: []byte("*HQ"),
			want: nil,
			wantErr: ErrPacketTooShort,
		},
		{
			name: "invalid message type",
			data: []byte("*HQ,V9,123456789012345,A,2237.7514,N,11408.6214,E#"),
			want: nil,
			wantErr: ErrInvalidMessageType,
		},
		{
			name: "invalid coordinate format",
			data: []byte("*HQ,V1,123456789012345,A,INVALID,N,11408.6214,E,6,2,151022#"),
			want: nil,
			wantErr: ErrInvalidCoordinate,
		},
		{
			name: "invalid latitude range",
			data: []byte("*HQ,V1,123456789012345,A,9237.7514,N,11408.6214,E,6,2,151022#"),
			want: nil,
			wantErr: ErrInvalidCoordinate,
		},
		{
			name: "invalid longitude range",
			data: []byte("*HQ,V1,123456789012345,A,2237.7514,N,19908.6214,E,6,2,151022#"),
			want: nil,
			wantErr: ErrInvalidCoordinate,
		},
		{
			name: "malformed packet",
			data: []byte("*HQ,V1,123456789012345"),
			want: nil,
			wantErr: ErrInvalidFormat,
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

			compareH02Data(t, got, tt.want)
		})
	}
}

func compareH02Data(t *testing.T, got, want *H02Data) {
	if got.Valid != want.Valid {
		t.Errorf("Valid = %v, want %v", got.Valid, want.Valid)
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
	if got.PowerLevel != want.PowerLevel {
		t.Errorf("PowerLevel = %v, want %v", got.PowerLevel, want.PowerLevel)
	}
	if got.GSMSignal != want.GSMSignal {
		t.Errorf("GSMSignal = %v, want %v", got.GSMSignal, want.GSMSignal)
	}
	if got.Alarm != want.Alarm {
		t.Errorf("Alarm = %v, want %v", got.Alarm, want.Alarm)
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
