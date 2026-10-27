package interceptors

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"sparkly/backend/internal/domain"
)

type KeyManager interface {
	GetCurrSignKey(ctx context.Context) (*rsa.PrivateKey, string, error)
}

type RefreshTokenRepo interface {
	GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (domain.User, domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error
}

type TokenRefresher struct {
	keyManager KeyManager
	repo       RefreshTokenRepo
}

func NewTokenRefresher(KeyManager KeyManager, repo RefreshTokenRepo) *TokenRefresher {
	return &TokenRefresher{
		keyManager: KeyManager,
		repo:       repo,
	}
}

func (r *TokenRefresher) genNewAccessToken(ctx context.Context, userID uuid.UUID, email string) (string, error) {
	signKey, kid, err := r.keyManager.GetCurrSignKey(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get private key: %w", err)
	}

	now := time.Now()

	accessClaims := jwt.MapClaims{
		"id":    userID.String(),
		"email": email,
		"exp":   now.Add(domain.ACCESS_TOKEN_LIFETIME).Unix(),
		"iat":   now.Unix(),
		"type":  "access",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessToken.Header["kid"] = kid
	accessString, err := accessToken.SignedString(signKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return accessString, nil
}

func (r *TokenRefresher) RefreshToken(ctx context.Context, tokenIDStr string) (domain.User, string, error) {
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("failed to parse token id: %w", err)
	}

	user, refreshToken, err := r.repo.GetRefreshToken(ctx, tokenID)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("failed to verify token: %w", err)
	}

	if refreshToken.ExpiresAt.Before(time.Now()) {
		return domain.User{}, "", fmt.Errorf("failed to continue user session: %w", domain.ErrExpiredRefreshToken)
	}

	if refreshToken.Revoked {
		return domain.User{}, "", fmt.Errorf("failed to continue user session: %w", domain.ErrRevokedRefreshToken)
	}

	newAccessToken, err := r.genNewAccessToken(ctx, user.ID, user.Email)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	return user, newAccessToken, nil
}
