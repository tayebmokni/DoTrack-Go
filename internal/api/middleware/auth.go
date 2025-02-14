package middleware

import (
    "context"
    "net/http"
    "strings"
    "tracking/internal/core/service"
)

type AuthMiddleware struct {
    userService service.UserService
}

func NewAuthMiddleware(userService service.UserService) *AuthMiddleware {
    return &AuthMiddleware{
        userService: userService,
    }
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        if auth == "" {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }

        parts := strings.Split(auth, " ")
        if len(parts) != 2 || parts[0] != "Basic" {
            http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
            return
        }

        // In production, implement proper token validation
        // For now, just pass the token as user ID
        ctx := context.WithValue(r.Context(), "userID", parts[1])
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
