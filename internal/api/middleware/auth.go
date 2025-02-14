package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"tracking/internal/api/util"
)

type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AuthMiddleware struct {
	accessSecret string
}

func NewAuthMiddleware() *AuthMiddleware {
	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		secret = "test_jwt_secret_key_123" // Default secret for development
		log.Printf("Warning: Using default JWT secret for development")
	}

	return &AuthMiddleware{
		accessSecret: secret,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		log.Printf("Processing token: %s", tokenString[:10]) // Log first 10 chars for debugging

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.accessSecret), nil
		})

		if err != nil {
			log.Printf("Token validation error: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			log.Printf("Invalid token claims or token not valid")
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Verify expiration
		if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
			log.Printf("Token expired at: %v", claims.ExpiresAt.Time)
			http.Error(w, "Token has expired", http.StatusUnauthorized)
			return
		}

		log.Printf("Successfully validated token for user: %s with role: %s", claims.Email, claims.Role)

		// Create UserClaims from JWT claims
		userClaims := &util.UserClaims{
			UserID: claims.Subject,
			Email:  claims.Email,
			Role:   claims.Role,
		}

		// Add claims to request context
		ctx := util.WithUserClaims(r.Context(), userClaims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}