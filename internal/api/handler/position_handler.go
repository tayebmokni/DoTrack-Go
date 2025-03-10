package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"tracking/internal/api/util"
	"tracking/internal/core/service"
)

type PositionHandler struct {
	positionService service.PositionService
}

func NewPositionHandler(positionService service.PositionService) *PositionHandler {
	return &PositionHandler{
		positionService: positionService,
	}
}

type addPositionRequest struct {
	DeviceID  string  `json:"deviceId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type rawDataRequest struct {
	DeviceID string `json:"deviceId"`
	RawData  string `json:"rawData"` // Base64 encoded raw data
}

func (h *PositionHandler) AddPosition(w http.ResponseWriter, r *http.Request) {
	var req addPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, err := util.GetUserClaims(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	position, err := h.positionService.AddPosition(req.DeviceID, req.Latitude, req.Longitude, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(position)
}

func (h *PositionHandler) GetPositions(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("deviceId")
	if deviceID == "" {
		http.Error(w, "Device ID required", http.StatusBadRequest)
		return
	}

	claims, err := util.GetUserClaims(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	positions, err := h.positionService.GetDevicePositions(deviceID, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(positions)
}

func (h *PositionHandler) GetLatestPosition(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("deviceId")
	if deviceID == "" {
		http.Error(w, "Device ID required", http.StatusBadRequest)
		return
	}

	claims, err := util.GetUserClaims(r)
	if err != nil {
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	position, err := h.positionService.GetLatestPosition(deviceID, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if position == nil {
		http.Error(w, "No position found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(position)
}

func (h *PositionHandler) ProcessRawData(w http.ResponseWriter, r *http.Request) {
	// Add debug logging
	fmt.Printf("Received raw data request: %s %s\n", r.Method, r.URL.Path)

	var req rawDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, err := util.GetUserClaims(r)
	if err != nil {
		fmt.Printf("Error getting user claims: %v\n", err)
		http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
		return
	}

	// Decode base64 data
	rawData, err := base64.StdEncoding.DecodeString(req.RawData)
	if err != nil {
		fmt.Printf("Error decoding base64 data: %v\n", err)
		http.Error(w, "Invalid raw data format", http.StatusBadRequest)
		return
	}

	fmt.Printf("Processing raw data for device: %s, data length: %d bytes\n", req.DeviceID, len(rawData))
	position, err := h.positionService.ProcessRawData(req.DeviceID, rawData, claims.UserID)
	if err != nil {
		fmt.Printf("Error processing raw data: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(position)
}