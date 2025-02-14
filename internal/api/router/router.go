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
	authMiddleware := middleware.NewAuthMiddleware()

	// Create router
	mux := http.NewServeMux()

	// Add middleware chain
	withMiddleware := func(handler http.Handler) http.Handler {
		return middleware.CORSMiddleware(
			middleware.LoggingMiddleware(
				authMiddleware.Authenticate(handler),
			),
		)
	}

	// Test endpoint
	mux.Handle("/test", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "API is working correctly",
		})
	})))

	// Health check endpoint
	mux.Handle("/health", withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "ok",
			"database": "connected",
		})
	})))

	// Device routes with method handling
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

	// Position routes - with device validation in service layer
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