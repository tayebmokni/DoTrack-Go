package model

import (
	"time"
	"tracking/internal/core/util"
)

type Position struct {
	ID         string                 `json:"id"`
	DeviceID   string                 `json:"deviceId"`
	Timestamp  time.Time             `json:"timestamp"`
	Latitude   float64               `json:"latitude"`
	Longitude  float64               `json:"longitude"`
	Altitude   float64               `json:"altitude"`
	Speed      float64               `json:"speed"`
	Course     float64               `json:"course"`
	Address    string                `json:"address,omitempty"`
	Protocol   string                `json:"protocol"`
	Valid      bool                  `json:"valid"`       // GPS fix validity
	Satellites uint8                 `json:"satellites"`  // Number of satellites used for fix
	Status     map[string]interface{} `json:"status,omitempty"` // Additional status information
}

func NewPosition(deviceID string, lat, lon float64) *Position {
	return &Position{
		ID:        util.GenerateID(),
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Latitude:  lat,
		Longitude: lon,
		Protocol:  "unknown",
		Valid:     true,
		Status:    make(map[string]interface{}),
	}
}

func GenerateID() string {
	//Implementation for GenerateID() would go here.  This is assumed to exist in the original file.
	return "" //Placeholder - Replace with actual implementation.
}