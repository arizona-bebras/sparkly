package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"
)

type DevicesRegHandler struct {
	service *services.DeviceService
}

func NewDeviceRegHandler(s *services.DeviceService) *DevicesRegHandler {
	return &DevicesRegHandler{
		service: s,
	}

}

type deviceEnrollmentRequest struct {
	Token string `json:"park_token"`
	CSR   string `json:"csr_pem"`
}

type deviceEnrollmentResponse struct {
	DeviceID            uuid.UUID `json:"device_id"`
	ParkID              uuid.UUID `json:"park_id"`
	IdentityURI         string    `json:"identity_uri"`
	CertificateChainPEM string    `json:"certificate_chain_pem"`
	ExpiresAt           time.Time `json:"expires_at"`
}

type errorResponse struct {
	StatusCode int    `json:"status_code"`
	Msg        string `json:"message"`
}

func (h *DevicesRegHandler) RegDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	writeError := func(status int, message string) {
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(errorResponse{StatusCode: status, Msg: message}); err != nil {
			log.Printf("/devices/reg response error: %v", err)
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request deviceEnrollmentRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(http.StatusBadRequest, "invalid enrollment request")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(http.StatusBadRequest, "expected a single JSON object")
		return
	}
	enrollment, err := h.service.RegDevice(r.Context(), request.Token, request.CSR)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidEnrollment):
			writeError(http.StatusBadRequest, domain.ErrInvalidEnrollment.Error())
		case errors.Is(err, domain.ErrParkNotFound), errors.Is(err, domain.ErrTokenNotFound):
			writeError(http.StatusUnauthorized, "invalid enrollment token")
		case errors.Is(err, domain.ErrCertificateIssuance):
			log.Printf("/devices/reg signing error: %v", err)
			writeError(http.StatusServiceUnavailable, "certificate issuance unavailable")
		default:
			log.Printf("/devices/reg error: %v", err)
			writeError(http.StatusInternalServerError, "device enrollment failed")
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
	response := deviceEnrollmentResponse{
		DeviceID:            enrollment.Identity.DeviceID(),
		ParkID:              enrollment.Identity.ParkID(),
		IdentityURI:         enrollment.Identity.URI(),
		CertificateChainPEM: enrollment.Certificate.ChainPEM,
		ExpiresAt:           enrollment.Certificate.ExpiresAt,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("/devices/reg response error: %v", err)
	}
}
