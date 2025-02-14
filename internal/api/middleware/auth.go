package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"tracking/internal/api/util"
)

type Claims struct {
	jwt.RegisteredClaims
	Email          string `json:"email"`
	Role           string `json:"role"`
	OrganizationID string `json:"organization_id,omitempty"`
}

type AuthMiddleware struct {
	accessSecret string
}

func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{
		accessSecret: os.Getenv("JWT_ACCESS_SECRET"),
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		// Extract the token from the "Bearer <token>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Parse and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.accessSecret), nil
		})

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Verify expiration
		if !claims.ExpiresAt.Time.After(time.Now()) {
			http.Error(w, "Token has expired", http.StatusUnauthorized)
			return
		}

		// Create UserClaims from JWT claims
		userClaims := &util.UserClaims{
			UserID:         claims.Subject,
			Email:          claims.Email,
			Role:           claims.Role,
			OrganizationID: claims.OrganizationID,
		}

		// Add claims to request context
		ctx := util.WithUserClaims(r.Context(), userClaims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
