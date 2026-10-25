package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sparkly/backend/internal/services"
)

type DevicesHandler struct {
	service *services.DeviceService
}

func NewDeviceHandler(s *services.DeviceService) *DevicesHandler {
	return &DevicesHandler{
		service: s,
	}
}

type parkToken struct {
	Token string `json:"park_token"`
}

type deviceTokenResponse struct {
	Token string `json:"device_token"`
}

func (h *DevicesHandler) RegDevice(w http.ResponseWriter, r *http.Request) {
	var parkToken parkToken
	if err := json.NewDecoder(r.Body).Decode(&parkToken); err != nil {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte("invalid body"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		return
	}

	deviceToken, err := h.service.RegDevice(r.Context(), parkToken.Token)
	if err != nil {
		log.Printf("/devices/reg error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deviceTokenResponse{
		Token: deviceToken,
	}); err != nil {
		log.Printf("/devices/reg error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
