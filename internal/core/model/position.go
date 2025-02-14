package model

import (
    "time"
)

type Position struct {
    ID        string    `json:"id"`
    DeviceID  string    `json:"deviceId"`
    Timestamp time.Time `json:"timestamp"`
    Latitude  float64   `json:"latitude"`
    Longitude float64   `json:"longitude"`
    Altitude  float64   `json:"altitude"`
    Speed     float64   `json:"speed"`
    Course    float64   `json:"course"`
    Address   string    `json:"address,omitempty"`
    Protocol  string    `json:"protocol"`
}

func NewPosition(deviceID string, lat, lon float64) *Position {
    return &Position{
        ID:        GenerateID(),
        DeviceID:  deviceID,
        Timestamp: time.Now(),
        Latitude:  lat,
        Longitude: lon,
        Protocol:  "teltonika",
    }
}
