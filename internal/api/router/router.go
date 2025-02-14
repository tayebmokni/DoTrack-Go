package router

import (
	"encoding/json"
	"net/http"
	"tracking/internal/api/handler"
	"tracking/internal/api/middleware"
	"tracking/internal/core/service"
)

func NewRouter(
	deviceService service.DeviceService,
	positionService service.PositionService,
) http.Handler {
	// Initialize handlers
	deviceHandler := handler.NewDeviceHandler(deviceService)
	positionHandler := handler.NewPositionHandler(positionService)
	authHandler := handler.NewAuthHandler()

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware()

	// Create router
	mux := http.NewServeMux()

	// Add middleware chain
	withMiddleware := func(handler http.Handler) http.Handler {
		return middleware.CORSMiddleware(
			middleware.LoggingMiddleware(
				authMiddleware.Authenticate(
					handler,
				),
			),
		)
	}

	// Health check endpoint (no auth required)
	mux.Handle("/health", middleware.CORSMiddleware(
		middleware.LoggingMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"status":   "ok",
					"database": "connected",
				})
			}),
		),
	))

	// Test login endpoint (unprotected)
	mux.Handle("/api/auth/test-login", middleware.CORSMiddleware(
		middleware.LoggingMiddleware(
			http.HandlerFunc(authHandler.TestLogin),
		),
	))

	// Protected routes
	mux.Handle("/api/devices", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			deviceHandler.Create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/devices/list", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		deviceHandler.GetDevices(w, r)
	})))

	mux.Handle("/api/devices/get", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		deviceHandler.GetDevice(w, r)
	})))

	// Position routes with authentication
	mux.Handle("/api/positions", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			positionHandler.AddPosition(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/positions/list", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		positionHandler.GetPositions(w, r)
	})))

	mux.Handle("/api/positions/latest", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		positionHandler.GetLatestPosition(w, r)
	})))

	mux.Handle("/api/positions/raw", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			positionHandler.ProcessRawData(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	return mux
}