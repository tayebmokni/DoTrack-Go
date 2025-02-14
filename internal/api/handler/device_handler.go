package handler

import (
	"encoding/json"
	"net/http"
	"tracking/internal/api/util"
	"tracking/internal/core/service"
)

type DeviceHandler struct {
	deviceService service.DeviceService
}

func NewDeviceHandler(deviceService service.DeviceService) *DeviceHandler {
	return &DeviceHandler{
		deviceService: deviceService,
	}
}

type createDeviceRequest struct {
	Name           string `json:"name"`
	UniqueID       string `json:"uniqueId"`
	UserID         string `json:"userId,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
}


func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user ID from JWT token
	userID, err := util.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	device, err := h.deviceService.CreateDevice(req.Name, req.UniqueID, userID, req.OrganizationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

func (h *DeviceHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	userID, err := util.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	orgID := r.URL.Query().Get("organizationId")

	var devices interface{}
	if orgID != "" {
		devices, err = h.deviceService.GetOrganizationDevices(orgID)
	} else {
		devices, err = h.deviceService.GetUserDevices(userID)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (h *DeviceHandler) GetDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("id")
	if deviceID == "" {
		http.Error(w, "Device ID required", http.StatusBadRequest)
		return
	}

	userID, err := util.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	// Validate user has access to this device
	if err := h.deviceService.ValidateDeviceAccess(deviceID, userID); err != nil {
		http.Error(w, "Unauthorized access to device", http.StatusUnauthorized)
		return
	}

	device, err := h.deviceService.GetDevice(deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if device == nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}