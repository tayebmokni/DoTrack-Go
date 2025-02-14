package teltonika

import (
    "bytes"
    "encoding/binary"
    "errors"
    "tracking/internal/core/model"
)

type Decoder struct{}

func NewDecoder() *Decoder {
    return &Decoder{}
}

type TeltonikaData struct {
    Latitude  float64
    Longitude float64
    Altitude  float64
    Speed     float64
    Course    float64
    Timestamp int64
}

func (d *Decoder) Decode(data []byte) (*TeltonikaData, error) {
    if len(data) < 16 {
        return nil, errors.New("data too short")
    }

    reader := bytes.NewReader(data)
    
    var result TeltonikaData
    
    // This is a simplified decoder. In production, implement full Teltonika protocol
    err := binary.Read(reader, binary.BigEndian, &result.Latitude)
    if err != nil {
        return nil, err
    }
    
    err = binary.Read(reader, binary.BigEndian, &result.Longitude)
    if err != nil {
        return nil, err
    }

    return &result, nil
}

func (d *Decoder) ToPosition(deviceID string, data *TeltonikaData) *model.Position {
    return model.NewPosition(deviceID, data.Latitude, data.Longitude)
}
