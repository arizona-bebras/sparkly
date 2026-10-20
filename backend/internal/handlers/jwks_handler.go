package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sparkly/backend/internal/services"
)

type JWKSHadnelr struct {
	service *services.JWKSService
}

func NewJWKSHandler(s *services.JWKSService) *JWKSHadnelr {
	return &JWKSHadnelr{
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

func (h *JWKSHadnelr) GetJWKS(w http.ResponseWriter, r *http.Request) {
	inMemoryKeys := h.service.GetKeys()

	publicKeys := make([]jwk, 0, len(inMemoryKeys))
	for _, key := range inMemoryKeys {
		publicKeys = append(publicKeys, jwk{
			Kty: "RSA",
			Use: "sig",
			Alg: "RS256",
			Kid: key.KID,
			N:   key.N,
			E:   key.E,
		})
	}

	if err := json.NewEncoder(w).Encode(publicKeys); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf(".well-known/jwks.json error: %w", err)
		return
	}
}
