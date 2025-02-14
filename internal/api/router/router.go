package router

import (
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

	// Create router
	mux := http.NewServeMux()

	// Add logging middleware
	loggingMiddleware := middleware.LoggingMiddleware

	// Health check endpoint
	mux.Handle("/health", loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})))

	// Device routes with method handling
	mux.Handle("/api/devices", loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			deviceHandler.Create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/devices/list", loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		deviceHandler.GetDevices(w, r)
	})))

	mux.Handle("/api/devices/get", loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		deviceHandler.GetDevice(w, r)
	})))

	// Position routes
	mux.Handle("/api/positions", loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			positionHandler.AddPosition(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/positions/list", loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		positionHandler.GetPositions(w, r)
	})))

	mux.Handle("/api/positions/latest", loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		positionHandler.GetLatestPosition(w, r)
	})))

	return mux
}