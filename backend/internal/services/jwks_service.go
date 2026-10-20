package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"sync"
	"time"

	"sparkly/backend/internal/domain"

	"golang.org/x/sync/singleflight"
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

	rotationGroup singleflight.Group
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

func NewJWKSService(repo JWKSRepo) (*JWKSService, error) {
	s := &JWKSService{
		repo:         repo,
		inMemoryKeys: make(map[string]domain.JWK),
	}

	ctx := context.Background()
	keys, err := repo.GetActualKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %v", err)
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
		err := s.generateNewKey(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate initial key: %w", err)
		}
	}
	return s, nil
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

func (s *JWKSService) GetCurrSignKey(ctx context.Context) (*rsa.PrivateKey, string, error) {
	s.mu.RLock()
	currentKey, exists := s.inMemoryKeys[s.currentKID]
	if exists && time.Now().Before(currentKey.SignUntil) {
		key := currentKey.Private
		kid := s.currentKID
		s.mu.RUnlock()
		return key, kid, nil
	}
	expiredKID := s.currentKID
	s.mu.RUnlock()

	res, err, _ := s.rotationGroup.Do(expiredKID, func() (any, error) {
		s.mu.RLock()
		if curr, ok := s.inMemoryKeys[s.currentKID]; ok && time.Now().Before(curr.SignUntil) {
			s.mu.RUnlock()
			return curr, nil
		}
		s.mu.RUnlock()

		newRSAKey, err := rsa.GenerateKey(rand.Reader, domain.KEY_SIZE)
		if err != nil {
			return nil, fmt.Errorf("failed to generate rsa key: %w", err)
		}

		savedKey, err := s.repo.SaveKey(ctx, newRSAKey)
		if err != nil {
			return nil, fmt.Errorf("failed to save key: %w", err)
		}

		s.mu.Lock()
		s.inMemoryKeys[savedKey.KID] = savedKey
		s.currentKID = savedKey.KID
		s.mu.Unlock()
		return savedKey, nil
	})
	if err != nil {
		return nil, "", err
	}
	activeKey := res.(domain.JWK)
	return activeKey.Private, activeKey.KID, nil
}
