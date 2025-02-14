package util

import (
	"errors"
	"net/http"
	"strings"
)

type UserClaims struct {
	UserID         string `json:"sub"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// GetUserClaims extracts all user claims from the JWT token
func GetUserClaims(r *http.Request) (*UserClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("no authorization header")
	}

	// Extract the token from the "Bearer <token>" format
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, errors.New("invalid authorization header format")
	}

	// Note: Token validation is handled by external auth service
	// Here we just extract the claims assuming the token is valid

	// For development/testing, we'll extract user info from token
	// In production, these would be properly decoded from the JWT
	token := parts[1]

	// Extract user ID from token subject claim
	claims := &UserClaims{
		UserID: token, // In production, this would be decoded from JWT
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