package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"sparkly/backend/internal/domain"
	"sync"
	"time"
)

type JWKSRepo interface {
	SaveKey(context.Context, *rsa.PrivateKey) (domain.JWK, error)
	GetActualKeys(ctx context.Context) ([]domain.JWK, error)
}

type inMemoryKey struct {
	key       *rsa.PrivateKey
	signUntil time.Time
	expiresAt time.Time
}

type JWKSService struct {
	mu           sync.RWMutex
	currentKID   string
	inMemoryKeys map[string]domain.JWK
	repo         JWKSRepo
}

func NewJWKSService(repo JWKSRepo) (*JWKSService, error) {
	s := &JWKSService{
		repo:         repo,
		inMemoryKeys: make(map[string]domain.JWK),
	}
	
	ctx := context.Background()
	keys, err := repo.GetActualKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %v")
	}
	now := time.Now()
	var latestActive *domain.JWK
	for _, k := range keys {
		s.inMemoryKeys[k.KID] = k

		if k.SignUntil.After(now) {
			if latestActive == nil || k.SignUntil.After(latestActive.CreatedAt.Add(domain.PRIVATE_KEY_SIGNING_PERIOD)) {
				keyCopy := k
				latestActive = &keyCopy
			}
		}
	}
	if latestActive != nil {
		s.currentKID = latestActive.KID
	} else {
		err := s.generateNewKey(ctx); 
		if err != nil {
			return nil, fmt.Errorf("failed to generate initial key: %w", err)
		}
	}
	return s, nil
}

func (r *JWKSService) generateNewKey(ctx context.Context) error {
	newKey, err := rsa.GenerateKey(rand.Reader, domain.KEY_SIZE)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	key, err := r.repo.SaveKey(ctx, newKey)
	if err != nil {
		return fmt.Errorf("failed to save key: %w", err)
	}

	r.inMemoryKeys[key.KID] = key
	r.currentKID = key.KID

	return nil
}

func (s *JWKSService) GetPublicKeys() []domain.JWK {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]domain.JWK, 0, len(s.inMemoryKeys))
	for _, inMemoryKey := range s.inMemoryKeys {
		if inMemoryKey.ExpiresAt.Before(time.Now()) {
			continue
		}

		keys = append(keys, inMemoryKey)
	}

	return keys
}

func (s *JWKSService) GetCurrSignKey(ctx context.Context) (*rsa.PrivateKey, error) {
	s.mu.RLock()
	currentKey, exists := s.inMemoryKeys[s.currentKID]

	if exists && time.Now().Before(currentKey.SignUntil) {
		key := currentKey.Private
		s.mu.RUnlock()
		return key, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if currentKey, exists = s.inMemoryKeys[s.currentKID]; exists && time.Now().Before(currentKey.SignUntil) {
		return currentKey.Private, nil
	}

	err := s.generateNewKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate key: %w", err)
	}
	return s.inMemoryKeys[s.currentKID].Private, nil
}
