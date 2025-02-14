package model

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
	"tracking/internal/core/util"
)

type Device struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	UniqueID       string    `json:"uniqueId"`
	Status         string    `json:"status"`
	LastUpdate     time.Time `json:"lastUpdate"`
	PositionID     string    `json:"positionId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	Protocol       string    `json:"protocol"`
	ApiKey         string    `json:"apiKey,omitempty"`
	ApiSecret      string    `json:"-"` // Not included in JSON responses
	OrganizationID string    `json:"organizationId,omitempty"`
	UserID         string    `json:"userId,omitempty"`
}

func NewDevice(name, uniqueID string) *Device {
	apiKey, _ := generateRandomKey(32)
	apiSecret, _ := generateRandomKey(32)

	return &Device{
		ID:         util.GenerateID(),
		Name:       name,
		UniqueID:   uniqueID,
		Status:     "inactive",
		LastUpdate: time.Now(),
		CreatedAt:  time.Now(),
		Protocol:   "teltonika",
		ApiKey:     apiKey,
		ApiSecret:  apiSecret,
	}
}

// NewTestDevice creates a new test device instance
func NewTestDevice(uniqueID string) *Device {
	return &Device{
		ID:         uniqueID,
		Name:       "Test Device",
		UniqueID:   uniqueID,
		Status:     "active",
		LastUpdate: time.Now(),
		CreatedAt:  time.Now(),
		Protocol:   "test",
	}
}

func (d *Device) SetOwnership(userID, organizationID string) {
	d.UserID = userID
	d.OrganizationID = organizationID
}

func generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (d *Device) ValidateCredentials(apiKey, apiSecret string) bool {
	return d.ApiKey == apiKey && d.ApiSecret == apiSecret
}

// IsTestDevice checks if this is a test device
func (d *Device) IsTestDevice() bool {
	return strings.HasPrefix(d.UniqueID, "test-") || strings.HasPrefix(d.UniqueID, "demo-")
}