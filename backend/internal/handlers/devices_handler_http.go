package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"
)

type DevicesRegHandler struct {
	service *services.DeviceService
}

func NewDeviceHandler(s *services.DeviceService) *DevicesRegHandler {
	return &DevicesRegHandler{
		service: s,
	}
}

type parkToken struct {
	Token string `json:"park_token"`
}

type deviceTokenResponse struct {
	Token string `json:"device_token"`
}

type errorResponse struct {
	StatusCode int    `json:"status_code"`
	Msg        string `json:"message"`
}

func (h *DevicesRegHandler) RegDevice(w http.ResponseWriter, r *http.Request) {
	var parkToken parkToken
	if err := json.NewDecoder(r.Body).Decode(&parkToken); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(errorResponse{
			StatusCode: http.StatusBadRequest,
			Msg:        err.Error(),
		})
		if encodeErr != nil {
			log.Printf("/devices/reg error: %v", err)
		}
		return
	}

	deviceToken, err := h.service.RegDevice(r.Context(), parkToken.Token)
	if err != nil {
		if errors.Is(err, domain.ErrParkNotFound) {
			w.WriteHeader(http.StatusNotFound)
			encodeErr := json.NewEncoder(w).Encode(errorResponse{
				StatusCode: http.StatusNotFound,
				Msg:        err.Error(),
			})
			if encodeErr != nil {
				log.Printf("/devices/reg error: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("/devices/reg error: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(deviceTokenResponse{
		Token: deviceToken,
	}); err != nil {
		log.Printf("/devices/reg error: %v", err)
		return
	}
}
