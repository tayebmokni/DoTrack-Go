package router

import (
	"encoding/json"
	"net/http"
	"tracking/internal/api/handler"
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

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Device routes
	mux.HandleFunc("/api/devices", deviceHandler.Create)
	mux.HandleFunc("/api/devices/list", deviceHandler.GetDevices)
	mux.HandleFunc("/api/devices/get", deviceHandler.GetDevice)

	// Position routes
	mux.HandleFunc("/api/positions", positionHandler.AddPosition)
	mux.HandleFunc("/api/positions/list", positionHandler.GetPositions)
	mux.HandleFunc("/api/positions/latest", positionHandler.GetLatestPosition)

	return mux
}