package repos

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/domain"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	KEY_SIZE       = 2048
	ALG            = "RS256"
	TOKEN_LIFETIME = time.Hour * 24 * 30
)

type inMemoryKey struct {
	key       *rsa.PrivateKey
	signUntil time.Time
	expiresAt time.Time
}

type JWKSRepo struct {
	mu           sync.RWMutex
	masterKey    []byte
	db           *gorm.DB
	currentKID   string
	inMemoryKeys map[string]*inMemoryKey
}

func (km *JWKSRepo) encryptPrivateKey(privKey *rsa.PrivateKey) ([]byte, error) {
	derBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	block, err := aes.NewCipher(km.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := gcm.Seal(nonce, nonce, derBytes, nil)
	return encrypted, nil
}

func (km *JWKSRepo) decryptPrivateKey(payload []byte) (*rsa.PrivateKey, error) {
	block, err := aes.NewCipher(km.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(payload) < nonceSize {
		return nil, errors.New("ciphertext payload too short")
	}
	nonce, ciphertext := payload[:nonceSize], payload[nonceSize:]
	derBytes, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key (auth tag failed or wrong master key): %w", err)
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decrypted private key: %w", err)
	}
	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("decrypted key is not an RSA private key")
	}
	return rsaKey, nil
}

func NewJWKSRepo(db *gorm.DB) (*JWKSRepo, error) {
	masterKey := os.Getenv("JWKS_MASTER_KEY")
	if len(masterKey) != 32 {
		return nil, errors.New("JWKS_MASTER_KEY must be 32 bytes")
	}
	r := &JWKSRepo{
		masterKey:    []byte(masterKey),
		db:           db,
		inMemoryKeys: make(map[string]*inMemoryKey),
	}
	var keys []models.SigningKey
	if err := db.Where("expires_at > ?", time.Now()).Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("failed to get keys from db: %w", err)
	}
	now := time.Now()
	var latestActive *models.SigningKey
	for _, k := range keys {
		decryptedKey, err := r.decryptPrivateKey(k.EncryptedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt key %s: %w", k.KID, err)
		}
		signUntil := k.CreatedAt.Add(domain.PRIVATE_KEY_SIGNING_PERIOD)
		r.inMemoryKeys[k.KID] = &inMemoryKey{
			key:       decryptedKey,
			signUntil: signUntil,
			expiresAt: k.ExpiresAt,
		}

		if signUntil.After(now) {
			if latestActive == nil || signUntil.After(latestActive.CreatedAt.Add(domain.PRIVATE_KEY_SIGNING_PERIOD)) {
				keyCopy := k
				latestActive = &keyCopy
			}
		}
	}
	if latestActive != nil {
		r.currentKID = latestActive.KID
	} else {
		if _, err := r.generateNewKey(); err != nil {
			return nil, fmt.Errorf("failed to generate initial key: %w", err)
		}
	}
	return r, nil
}

func (r *JWKSRepo) generateNewKey() (string, error) {
	newKey, err := rsa.GenerateKey(rand.Reader, KEY_SIZE)
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	encryptedKey, err := r.encryptPrivateKey(newKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	signUntil := time.Now().Add(domain.PRIVATE_KEY_SIGNING_PERIOD)

	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate id: %w", err)
	}

	key := &models.SigningKey{
		ID:           id,
		KID:          fmt.Sprintf("key-%s", id.String()),
		EncryptedKey: encryptedKey,
		Algorithm:    ALG,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(domain.PRIVATE_KEY_LIFETIME),
	}

	if err = r.db.Create(key).Error; err != nil {
		return "", fmt.Errorf("failed to write generated key %w", err)
	}

	r.inMemoryKeys[key.KID] = &inMemoryKey{
		key:       newKey,
		signUntil: signUntil,
		expiresAt: key.ExpiresAt,
	}
	newKID := key.KID


	return newKID, nil
}

func (r *JWKSRepo) GetPublicKeys() []domain.JWK {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	keys := make([]domain.JWK, 0, len(r.inMemoryKeys))
	for kid, inMemoryKey := range r.inMemoryKeys {
		if inMemoryKey.expiresAt.Before(time.Now()) {
			continue
		}

		publicKey := inMemoryKey.key.Public().(*rsa.PublicKey)
		eBytes := big.NewInt(int64(inMemoryKey.key.E)).Bytes()
		keys = append(keys, domain.JWK{
			KID: kid,
			N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(eBytes),
		})
	}

	return keys
}

func (r *JWKSRepo) GetPrivateKey() (*rsa.PrivateKey, error) {
	r.mu.RLock()
	currentKey, exists := r.inMemoryKeys[r.currentKID]

	if exists && time.Now().Before(currentKey.signUntil) {
		key := currentKey.key
		r.mu.RUnlock()
		return key, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if currentKey, exists = r.inMemoryKeys[r.currentKID]; exists && time.Now().Before(currentKey.signUntil) {
		return currentKey.key, nil
	}

	newKID, err := r.generateNewKey()
	if err != nil {
		return nil,  fmt.Errorf("failed to rotate key: %w", err)
	}
	return r.inMemoryKeys[newKID].key, nil
}
