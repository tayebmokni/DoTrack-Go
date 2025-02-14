package model

import (
	"time"
)

type Device struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	UniqueID    string    `json:"uniqueId"`
	Status      string    `json:"status"`
	LastUpdate  time.Time `json:"lastUpdate"`
	PositionID  string    `json:"positionId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	Protocol    string    `json:"protocol"`
}

func NewDevice(name, uniqueID string) *Device {
	return &Device{
		ID:         GenerateID(),
		Name:       name,
		UniqueID:   uniqueID,
		Status:     "inactive",
		LastUpdate: time.Now(),
		CreatedAt:  time.Now(),
		Protocol:   "teltonika",
	}
}

func GenerateID() string {
	return time.Now().Format("20060102150405")
}