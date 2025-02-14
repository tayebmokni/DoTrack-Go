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
	OrganizationID string `json:"organizationId,omitempty"`
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user claims from JWT token
	claims, err := util.GetUserClaims(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	// Check organization access if creating for an organization
	if req.OrganizationID != "" {
		if !util.CanAccessOrganization(claims.Role, claims.OrganizationID, req.OrganizationID) {
			http.Error(w, "Unauthorized access to organization", http.StatusForbidden)
			return
		}
	}

	device, err := h.deviceService.CreateDevice(req.Name, req.UniqueID, claims.UserID, req.OrganizationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

func (h *DeviceHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	claims, err := util.GetUserClaims(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	orgID := r.URL.Query().Get("organizationId")

	// If requesting organization devices, verify access
	if orgID != "" {
		if !util.CanAccessOrganization(claims.Role, claims.OrganizationID, orgID) {
			http.Error(w, "Unauthorized access to organization", http.StatusForbidden)
			return
		}
		devices, err := h.deviceService.GetOrganizationDevices(orgID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(devices)
		return
	}

	// Get user's devices
	devices, err := h.deviceService.GetUserDevices(claims.UserID)
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

	claims, err := util.GetUserClaims(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	// Validate user has access to this device
	if err := h.deviceService.ValidateDeviceAccess(deviceID, claims.UserID); err != nil {
		http.Error(w, "Unauthorized access to device", http.StatusForbidden)
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