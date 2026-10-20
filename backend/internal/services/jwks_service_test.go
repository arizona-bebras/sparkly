package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"sparkly/backend/internal/domain"

	"github.com/google/uuid"
)

type MockJWKSRepo struct {
	mu               sync.RWMutex
	keys             map[string]domain.JWK
	saveKeyErr       error
	getActualKeysErr error
	saveKeyCalls     int
}

func (r *MockJWKSRepo) SaveKeyCalls() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.saveKeyCalls
}

func (r *MockJWKSRepo) SetSaveKeyErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveKeyErr = err
}

func (r *MockJWKSRepo) SaveKey(_ context.Context, key *rsa.PrivateKey) (domain.JWK, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.saveKeyCalls++
	if r.saveKeyErr != nil {
		return domain.JWK{}, r.saveKeyErr
	}

	id, err := uuid.NewV7()
	if err != nil {
		return domain.JWK{}, err
	}

	publicKey := key.Public().(*rsa.PublicKey)
	eBytes := big.NewInt(int64(publicKey.E)).Bytes()
	kid := "key-" + id.String()
	now := time.Now()
	jwk := domain.JWK{
		KID:       kid,
		N:         base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
		E:         base64.RawURLEncoding.EncodeToString(eBytes),
		CreatedAt: now,
		ExpiresAt: now.Add(domain.PRIVATE_KEY_LIFETIME),
		SignUntil: now.Add(domain.PRIVATE_KEY_SIGNING_PERIOD),
		Private:   key,
	}

	if r.keys == nil {
		r.keys = make(map[string]domain.JWK)
	}
	r.keys[kid] = jwk

	return jwk, nil
}

func (r *MockJWKSRepo) GetActualKeys(_ context.Context) ([]domain.JWK, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.getActualKeysErr != nil {
		return nil, r.getActualKeysErr
	}

	keys := make([]domain.JWK, 0, len(r.keys))
	for _, key := range r.keys {
		keys = append(keys, key)
	}

	return keys, nil
}

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test rsa key: %v", err)
	}
	return k
}

func TestJWKSService_Public(t *testing.T) {
	expiredKID := "key-EXPIRED"
	expiredPrivateKey := generateTestKey(t)
	expiredKey := domain.JWK{
		KID:       expiredKID,
		ExpiresAt: time.Now().Add(-time.Hour * 24),
		Private:   expiredPrivateKey,
	}

	rotatingKID := "key-ROTATING"
	rotatingPrivateKey := generateTestKey(t)
	rotatingKey := domain.JWK{
		KID:       rotatingKID,
		SignUntil: time.Now().Add(-time.Hour * 24),
		ExpiresAt: time.Now().Add(time.Hour * 24),
		Private:   rotatingPrivateKey,
	}

	tests := []struct {
		Name     string
		Repo     *MockJWKSRepo
		Validate func(s *JWKSService) error
	}{
		{
			Name: "first key generation",
			Repo: &MockJWKSRepo{
				keys: make(map[string]domain.JWK),
			},
			Validate: func(s *JWKSService) error {
				keys := s.GetPublicKeys()
				if len(keys) != 1 {
					return fmt.Errorf("must be single key after service creation, keys count: %v", len(keys))
				}
				return nil
			},
		},
		{
			Name: "omit expired keys",
			Repo: &MockJWKSRepo{
				keys: map[string]domain.JWK{
					expiredKID: expiredKey,
				},
			},
			Validate: func(s *JWKSService) error {
				keys := s.GetPublicKeys()
				if len(keys) != 1 {
					return fmt.Errorf("must be only 1 key (the generated active key), got count: %d", len(keys))
				}
				if keys[0].KID == expiredKID {
					return fmt.Errorf("expired key %s was not omitted from public keys", expiredKID)
				}
				return nil
			},
		},
		{
			Name: "includes rotating keys still within verification grace period",
			Repo: &MockJWKSRepo{
				keys: map[string]domain.JWK{
					rotatingKID: rotatingKey,
				},
			},
			Validate: func(s *JWKSService) error {
				keys := s.GetPublicKeys()
				if len(keys) != 2 {
					return fmt.Errorf("must have 2 keys (active + rotating), got: %d", len(keys))
				}
				foundRotating := false
				for _, k := range keys {
					if k.KID == rotatingKID {
						foundRotating = true
						break
					}
				}
				if !foundRotating {
					return fmt.Errorf("rotating key %s missing from public keys", rotatingKID)
				}
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			service, err := NewJWKSService(test.Repo)
			if err != nil {
				t.Fatalf("service init error: %v", err)
			}

			err = test.Validate(service)
			if err != nil {
				t.Errorf("%v", err)
			}
		})
	}
}

func TestJWKSService_GetCurrSignKey_Rotation(t *testing.T) {
	initialKID := "key-ACTIVE-INITIAL"
	initialPrivateKey := generateTestKey(t)
	now := time.Now()
	initialKey := domain.JWK{
		KID:       initialKID,
		CreatedAt: now.Add(-time.Hour * 24),
		SignUntil: now.Add(time.Hour * 24 * 30),
		ExpiresAt: now.Add(time.Hour * 24 * 60),
		Private:   initialPrivateKey,
	}

	mockRepo := &MockJWKSRepo{
		keys: map[string]domain.JWK{
			initialKID: initialKey,
		},
	}

	ctx := context.Background()
	service, err := NewJWKSService(mockRepo)
	if err != nil {
		t.Fatalf("failed to init service: %v", err)
	}

	key, kid, err := service.GetCurrSignKey(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting current sign key: %v", err)
	}
	if kid != initialKID {
		t.Fatalf("expected initial kid %s, got %s", initialKID, kid)
	}
	if key != initialPrivateKey {
		t.Fatalf("expected initial private key pointer, got different key")
	}
	if mockRepo.SaveKeyCalls() != 0 {
		t.Fatalf("expected 0 SaveKey calls, got %d", mockRepo.SaveKeyCalls())
	}

	service.mu.Lock()
	expiredKey := service.inMemoryKeys[initialKID]
	expiredKey.SignUntil = time.Now().Add(-time.Hour)
	service.inMemoryKeys[initialKID] = expiredKey
	service.mu.Unlock()

	rotatedKey, rotatedKID, err := service.GetCurrSignKey(ctx)
	if err != nil {
		t.Fatalf("failed to rotate key: %v", err)
	}
	if rotatedKID == initialKID {
		t.Fatalf("expected new rotated KID, but still got initial KID %s", initialKID)
	}
	if rotatedKey == initialPrivateKey {
		t.Fatalf("expected new private key, but got initial private key")
	}
	if mockRepo.SaveKeyCalls() != 1 {
		t.Fatalf("expected SaveKey to be called once during rotation, got %d", mockRepo.SaveKeyCalls())
	}

	publicKeys := service.GetPublicKeys()
	if len(publicKeys) != 2 {
		t.Fatalf("expected 2 public keys after rotation, got %d", len(publicKeys))
	}
	kids := make(map[string]bool)
	for _, pk := range publicKeys {
		kids[pk.KID] = true
	}
	if !kids[initialKID] || !kids[rotatedKID] {
		t.Fatalf("expected public keys to contain both %s and %s, got %v", initialKID, rotatedKID, kids)
	}

	subsequentKey, subsequentKID, err := service.GetCurrSignKey(ctx)
	if err != nil {
		t.Fatalf("failed on subsequent sign key call: %v", err)
	}
	if subsequentKID != rotatedKID || subsequentKey != rotatedKey {
		t.Fatalf("expected same rotated key to be reused, got %s", subsequentKID)
	}
	if mockRepo.SaveKeyCalls() != 1 {
		t.Fatalf("expected SaveKey calls to remain 1, got %d", mockRepo.SaveKeyCalls())
	}
}

func TestJWKSService_Initialization(t *testing.T) {
	t.Run("selects latest active key when multiple exist", func(t *testing.T) {
		now := time.Now()
		olderKey := domain.JWK{
			KID:       "key-OLDER",
			CreatedAt: now.Add(-time.Hour * 24 * 10),
			SignUntil: now.Add(time.Hour * 24 * 50),
			ExpiresAt: now.Add(time.Hour * 24 * 80),
			Private:   generateTestKey(t),
		}
		newerKey := domain.JWK{
			KID:       "key-NEWER",
			CreatedAt: now.Add(-time.Hour * 24 * 2),
			SignUntil: now.Add(time.Hour * 24 * 58),
			ExpiresAt: now.Add(time.Hour * 24 * 88),
			Private:   generateTestKey(t),
		}

		mockRepo := &MockJWKSRepo{
			keys: map[string]domain.JWK{
				olderKey.KID: olderKey,
				newerKey.KID: newerKey,
			},
		}

		service, err := NewJWKSService(mockRepo)
		if err != nil {
			t.Fatalf("unexpected error initializing service: %v", err)
		}

		_, kid, err := service.GetCurrSignKey(context.Background())
		if err != nil {
			t.Fatalf("unexpected error getting current sign key: %v", err)
		}
		if kid != newerKey.KID {
			t.Errorf("expected latest active key %s, got %s", newerKey.KID, kid)
		}
		if mockRepo.SaveKeyCalls() != 0 {
			t.Errorf("expected 0 SaveKey calls, got %d", mockRepo.SaveKeyCalls())
		}
	})

	t.Run("fails when repo.GetActualKeys returns error", func(t *testing.T) {
		mockRepo := &MockJWKSRepo{
			getActualKeysErr: errors.New("db connection failure"),
		}
		_, err := NewJWKSService(mockRepo)
		if err == nil {
			t.Fatal("expected error on failed GetActualKeys, got nil")
		}
	})

	t.Run("fails when repo.SaveKey fails during initial key generation", func(t *testing.T) {
		mockRepo := &MockJWKSRepo{
			keys:       make(map[string]domain.JWK),
			saveKeyErr: errors.New("db write failure"),
		}
		_, err := NewJWKSRepoInit(mockRepo)
		if err == nil {
			t.Fatal("expected error on failed initial SaveKey, got nil")
		}
	})
}

func NewJWKSRepoInit(mockRepo *MockJWKSRepo) (*JWKSService, error) {
	return NewJWKSService(mockRepo)
}

func TestJWKSService_GetCurrSignKey_RotationError(t *testing.T) {
	now := time.Now()
	initialKID := "key-EXPIRED-SIGNING"
	initialKey := domain.JWK{
		KID:       initialKID,
		CreatedAt: now.Add(-time.Hour * 24 * 65),
		SignUntil: now.Add(time.Hour * 24 * 5), // Valid initially
		ExpiresAt: now.Add(time.Hour * 24 * 25),
		Private:   generateTestKey(t),
	}

	mockRepo := &MockJWKSRepo{
		keys: map[string]domain.JWK{
			initialKID: initialKey,
		},
	}

	service, err := NewJWKSService(mockRepo)
	if err != nil {
		t.Fatalf("failed to init service: %v", err)
	}

	service.mu.Lock()
	k := service.inMemoryKeys[initialKID]
	k.SignUntil = time.Now().Add(-time.Hour)
	service.inMemoryKeys[initialKID] = k
	service.mu.Unlock()

	expectedErr := errors.New("database disk full")
	mockRepo.SetSaveKeyErr(expectedErr)

	_, _, err = service.GetCurrSignKey(context.Background())
	if err == nil {
		t.Fatal("expected error on failed rotation SaveKey, got nil")
	}
}

func TestJWKSService_Concurrency(t *testing.T) {
	mockRepo := &MockJWKSRepo{
		keys: make(map[string]domain.JWK),
	}
	service, err := NewJWKSService(mockRepo)
	if err != nil {
		t.Fatalf("failed to init service: %v", err)
	}

	initialKID := service.currentKID
	initialSaveCalls := mockRepo.SaveKeyCalls() // Should be 1 (initial key)

	service.mu.Lock()
	k := service.inMemoryKeys[initialKID]
	k.SignUntil = time.Now().Add(-time.Hour)
	service.inMemoryKeys[initialKID] = k
	service.mu.Unlock()

	const readers = 25
	const signers = 25
	var wg sync.WaitGroup
	errs := make(chan error, (readers+signers)*2)

	for i := 0; i < signers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, kid, err := service.GetCurrSignKey(context.Background())
			if err != nil {
				errs <- fmt.Errorf("GetCurrSignKey error: %w", err)
				return
			}
			if key == nil {
				errs <- fmt.Errorf("nil private key returned")
				return
			}
			if kid == "" || kid == initialKID {
				errs <- fmt.Errorf("expected new rotated KID, got %s", kid)
				return
			}
		}()
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			keys := service.GetPublicKeys()
			if len(keys) == 0 {
				errs <- fmt.Errorf("GetPublicKeys returned 0 keys")
				return
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}

	if total := mockRepo.SaveKeyCalls(); total != initialSaveCalls+1 {
		t.Errorf("expected exactly 1 rotation call via singleflight, got total SaveKey calls: %d (initial: %d)", total, initialSaveCalls)
	}
}
