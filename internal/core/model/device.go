package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"
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
		ID:         GenerateID(),
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

func (d *Device) SetOwnership(userID, organizationID string) {
	d.UserID = userID
	d.OrganizationID = organizationID
}

func GenerateID() string {
	return time.Now().Format("20060102150405")
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