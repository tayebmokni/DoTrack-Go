package util

import (
	"net/http"
	"strings"
)

// GetUserIDFromToken extracts userID from Authorization header
// Note: This assumes the JWT token is verified by an external auth service
func GetUserIDFromToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", nil
	}

	// Extract the token from the "Bearer <token>" format
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", nil
	}

	// In production, this would verify the JWT token
	// For now, we'll assume the token itself is the user ID
	return parts[1], nil
}
