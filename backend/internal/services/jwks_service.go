package services

import (
	"sparkly/backend/internal/domain"
)

type JWKSRepo interface {
	GetPublicKeys() []domain.JWK
}

type JWKSService struct {
	repo JWKSRepo
}

func NewJWKSService (repo JWKSRepo) *JWKSService {
	return &JWKSService{
		repo: repo,
	}
}

func (s *JWKSService) GetKeys() []domain.JWK {
	return s.repo.GetPublicKeys()
}