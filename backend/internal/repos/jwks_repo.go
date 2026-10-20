package repos

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/domain"
	"time"
	"math/big"

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
	masterKey    []byte
	db           *gorm.DB
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
	}

	return r, nil
}

func (r *JWKSRepo) GetActualKeys(ctx context.Context) ([]domain.JWK, error) {
	var encryptedKeys []models.SigningKey
	if err := r.db.WithContext(ctx).Where("expires_at > ?", time.Now()).Find(&encryptedKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get keys from db: %w", err)
	}

	keys := make([]domain.JWK, len(encryptedKeys))
	for i, key := range encryptedKeys {
		privateKey, err := r.decryptPrivateKey(key.EncryptedKey)
		if err != nil {
			return []domain.JWK{}, fmt.Errorf("failed to decrypt key: %w", err)
		}

		publicKey := privateKey.Public().(*rsa.PublicKey)
		eBytes := big.NewInt(int64(publicKey.E)).Bytes()
		keys[i] = domain.JWK{
			KID: key.KID,
			N: base64.URLEncoding.EncodeToString(publicKey.N.Bytes()),
			E: base64.RawURLEncoding.EncodeToString(eBytes),
			CreatedAt: key.CreatedAt,
			ExpiresAt: key.ExpiresAt,
			SignUntil: key.CreatedAt.Add(domain.PRIVATE_KEY_SIGNING_PERIOD),
			Private: privateKey,
		}
	}

	return keys, nil
}

func (r *JWKSRepo) SaveKey(ctx context.Context, key *rsa.PrivateKey) (domain.JWK, error) {
	encryptedKey, err := r.encryptPrivateKey(key)
	if err != nil {
		return domain.JWK{}, fmt.Errorf("failed to generate key: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return domain.JWK{}, fmt.Errorf("failed to generate id: %w", err)
	}

	savedKey := &models.SigningKey{
		ID:           id,
		KID:          fmt.Sprintf("key-%s", id.String()),
		EncryptedKey: encryptedKey,
		Algorithm:    ALG,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(domain.PRIVATE_KEY_LIFETIME),
	}

	if err = r.db.Create(savedKey).Error; err != nil {
		return domain.JWK{}, fmt.Errorf("failed to write generated key %w", err)
	}

	publicKey := key.Public().(*rsa.PublicKey)
	eBytes := big.NewInt(int64(publicKey.E)).Bytes()
	return domain.JWK{
		KID: savedKey.KID,
		N: base64.URLEncoding.EncodeToString(publicKey.N.Bytes()),
		E: base64.RawURLEncoding.EncodeToString(eBytes),
		CreatedAt: savedKey.CreatedAt,
		ExpiresAt: savedKey.ExpiresAt,
		SignUntil: savedKey.CreatedAt.Add(domain.PRIVATE_KEY_SIGNING_PERIOD),
		Private: key,
	}, nil
}
