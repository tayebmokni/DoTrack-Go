package util

import (
	"context"
	"errors"
	"net/http"
)

type UserClaims struct {
	UserID         string `json:"sub"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	OrganizationID string `json:"organization_id,omitempty"`
}

type contextKey string

const userClaimsKey contextKey = "userClaims"

// WithUserClaims adds UserClaims to the context
func WithUserClaims(ctx context.Context, claims *UserClaims) context.Context {
	return context.WithValue(ctx, userClaimsKey, claims)
}

// GetUserClaims extracts UserClaims from the context
func GetUserClaims(r *http.Request) (*UserClaims, error) {
	claims, ok := r.Context().Value(userClaimsKey).(*UserClaims)
	if !ok {
		return nil, errors.New("no user claims found in context")
	}
	return claims, nil
}

// IsAdmin checks if the user has admin role
func IsAdmin(role string) bool {
	return role == "admin"
}

// IsOrganizationAdmin checks if the user has organization_admin role
func IsOrganizationAdmin(role string) bool {
	return role == "organization_admin"
}

// CanAccessOrganization checks if user has access to an organization
func CanAccessOrganization(userRole string, userOrgID, targetOrgID string) bool {
	if IsAdmin(userRole) {
		return true
	}

	if userOrgID == "" || targetOrgID == "" {
		return false
	}

	return userOrgID == targetOrgID &&
		(IsOrganizationAdmin(userRole) || userRole == "organization_member")
}