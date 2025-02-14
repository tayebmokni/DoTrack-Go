package model

import (
	"time"
)

type OrganizationMember struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	UserID         string    `json:"userId"`
	Role           string    `json:"role"` // admin, member
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func NewOrganizationMember(organizationID, userID string, role string) *OrganizationMember {
	return &OrganizationMember{
		ID:             GenerateID(),
		OrganizationID: organizationID,
		UserID:         userID,
		Role:           role,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}
