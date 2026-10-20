package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sparkly/backend/internal/services"
)

type JWKSHandler struct {
	service *services.JWKSService
}

func NewJWKSHandler(s *services.JWKSService) *JWKSHandler {
	return &JWKSHandler{
		service: s,
	}
}

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

func (h *JWKSHandler) GetJWKS(w http.ResponseWriter, r *http.Request) {
	inMemoryKeys := h.service.GetPublicKeys()

	publicKeys := jwks{
		Keys: make([]jwk, 0, len(inMemoryKeys)),
	}
	for _, key := range inMemoryKeys {
		publicKeys.Keys = append(publicKeys.Keys, jwk{
			Kty: "RSA",
			Use: "sig",
			Alg: "RS256",
			Kid: key.KID,
			N:   key.N,
			E:   key.E,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(publicKeys); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf(".well-known/jwks.json error: %v", err)
		return
	}
}
