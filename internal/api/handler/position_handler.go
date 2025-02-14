package handler

import (
    "encoding/json"
    "net/http"
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

func (h *PositionHandler) AddPosition(w http.ResponseWriter, r *http.Request) {
    var req addPositionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    position, err := h.positionService.AddPosition(req.DeviceID, req.Latitude, req.Longitude)
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

    positions, err := h.positionService.GetDevicePositions(deviceID)
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

    position, err := h.positionService.GetLatestPosition(deviceID)
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
