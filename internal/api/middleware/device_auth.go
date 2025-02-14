package middleware

import (
	"context"
	"net/http"
	"tracking/internal/core/service"
)

type DeviceAuthMiddleware struct {
	deviceService service.DeviceService
}

func NewDeviceAuthMiddleware(deviceService service.DeviceService) *DeviceAuthMiddleware {
	return &DeviceAuthMiddleware{
		deviceService: deviceService,
	}
}

func (m *DeviceAuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-Device-API-Key")
		apiSecret := r.Header.Get("X-Device-API-Secret")

		if apiKey == "" || apiSecret == "" {
			http.Error(w, "Device authentication required", http.StatusUnauthorized)
			return
		}

		// Get device ID from request parameters or body
		deviceID := r.URL.Query().Get("deviceId")
		if deviceID == "" {
			deviceID = r.URL.Query().Get("id")
		}

		if deviceID == "" {
			http.Error(w, "Device ID required", http.StatusBadRequest)
			return
		}

		// Verify device credentials
		device, err := m.deviceService.GetDevice(deviceID)
		if err != nil {
			http.Error(w, "Error verifying device credentials", http.StatusInternalServerError)
			return
		}

		if device == nil {
			http.Error(w, "Device not found", http.StatusNotFound)
			return
		}

		if !device.ValidateCredentials(apiKey, apiSecret) {
			http.Error(w, "Invalid device credentials", http.StatusUnauthorized)
			return
		}

		// Add device to context
		ctx := context.WithValue(r.Context(), "device", device)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
