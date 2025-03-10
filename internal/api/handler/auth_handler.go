package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	accessSecret  string
	refreshSecret string
}

func NewAuthHandler() *AuthHandler {
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		accessSecret = "test_jwt_secret_key_123" // Default secret for development
	}

	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = "test_jwt_refresh_key_123" // Default secret for development
	}

	return &AuthHandler{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// TestLogin is a temporary endpoint for testing JWT authentication
func (h *AuthHandler) TestLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For testing, accept any credentials
	accessToken, err := h.generateAccessToken(req.Email)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.generateRefreshToken(req.Email)
	if err != nil {
		http.Error(w, "Error generating refresh token", http.StatusInternalServerError)
		return
	}

	resp := loginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) generateAccessToken(email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   "test-user-id",
		"email": email,
		"role":  "admin", // For testing purposes
		"exp":   now.Add(15 * time.Minute).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.accessSecret))
}

func (h *AuthHandler) generateRefreshToken(email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   "test-user-id",
		"email": email,
		"exp":   now.Add(7 * 24 * time.Hour).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.refreshSecret))
}