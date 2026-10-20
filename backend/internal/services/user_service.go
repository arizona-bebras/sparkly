package services

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"sparkly/backend/internal/domain"
)

type UserRepo interface {
	WriteNewUser(ctx context.Context, email, hashedPassword string) (domain.User, error)
	GetUser(ctx context.Context, email string) (domain.User, error)
	WriteRefreshToken(ctx context.Context, refreshString string, tokenID, userID uuid.UUID) error
	GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (domain.User, domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error
}

type KeyManager interface {
	GetCurrSignKey(ctx context.Context) (*rsa.PrivateKey, string, error)
}

type UserService struct {
	repo       UserRepo
	keyManager KeyManager
}

func NewUserService(userRepo UserRepo, KeyManager KeyManager) *UserService {
	return &UserService{
		repo:       userRepo,
		keyManager: KeyManager,
	}
}

type AuthToken struct {
	Access  string
	Refresh string
}

func (s *UserService) generateTokens(ctx context.Context, id uuid.UUID, email string) (AuthToken, error) {
	signKey, kid, err := s.keyManager.GetCurrSignKey(ctx)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to get private key: %w", err)
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
		return AuthToken{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	tokenID, err := uuid.NewV7()
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to gen token id: %w", err)
	}
	refreshClaims := jwt.MapClaims{
		"token_id": tokenID.String(),
		"user_id":  userID.String(),
		"exp":      now.Add(domain.REFRESH_TOKEN_LIFETIME).Unix(),
		"iat":      now.Unix(),
		"type":     "refresh",
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshToken.Header["kid"] = kid
	refreshString, err := refreshToken.SignedString(signKey)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	err = s.repo.WriteRefreshToken(ctx, refreshString, tokenID, userID)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to write token: %w", err)
	}

	return AuthToken{
		Access:  accessString,
		Refresh: refreshString,
	}, nil
}

func (s *UserService) Reg(ctx context.Context, email string, password string) (domain.User, AuthToken, error) {
	hashed, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return domain.User{}, AuthToken{}, fmt.Errorf("failed to secure password: %w", err)
	}

	user, err := s.repo.WriteNewUser(ctx, email, hashed)
	if err != nil {
		return domain.User{}, AuthToken{}, fmt.Errorf("registration failed: %w", err)
	}

	tokens, err := s.generateTokens(ctx, user.ID, user.Email)
	if err != nil {
		return domain.User{}, AuthToken{}, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return user, tokens, nil
}

func (s *UserService) GetTokens(ctx context.Context, email string, password string) (domain.User, AuthToken, error) {
	user, err := s.repo.GetUser(ctx, email)
	if err != nil {
		return domain.User{}, AuthToken{}, fmt.Errorf("auth failed: %w", err)
	}

	isPasswordMatch, err := argon2id.ComparePasswordAndHash(password, user.Password)
	if !isPasswordMatch {
		return domain.User{}, AuthToken{}, fmt.Errorf("auth failed: %w", domain.ErrWrongPassword)
	}

	tokens, err := s.generateTokens(ctx, user.ID, user.Email)
	if err != nil {
		return domain.User{}, AuthToken{}, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return user, tokens, nil
}

func (s *UserService) genNewAccessToken(userID uuid.UUID, email string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET environment variable is not set")
	}
	secretKey := []byte(secret)

	now := time.Now()

	accessClaims := jwt.MapClaims{
		"id":    userID.String(),
		"email": email,
		"exp":   now.Add(domain.ACCESS_TOKEN_LIFETIME).Unix(),
		"iat":   now.Unix(),
		"type":  "access",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return accessString, nil
}

func (s *UserService) RefreshToken(ctx context.Context, tokenIDStr string) (domain.User, AuthToken, error) {
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return domain.User{}, AuthToken{}, fmt.Errorf("failed to parse token id: %w", err)
	}

	user, refreshToken, err := s.repo.GetRefreshToken(ctx, tokenID)
	if err != nil {
		return domain.User{}, AuthToken{}, fmt.Errorf("failed to verify token: %w", err)
	}

	if refreshToken.ExpiresAt.Before(time.Now()) {
		return domain.User{}, AuthToken{}, fmt.Errorf("failed to continue user session: %w", domain.ErrExpiredRefreshToken)
	}

	if refreshToken.Revoked {
		return domain.User{}, AuthToken{}, fmt.Errorf("failed to continue user session: %w", domain.ErrRevokedRefreshToken)
	}

	newAccessToken, err := s.genNewAccessToken(user.ID, user.Email)
	if err != nil {
		return domain.User{}, AuthToken{}, fmt.Errorf("failed to generate new access token: %w", err)
	}

	return user, AuthToken{
		Access:  newAccessToken,
		Refresh: refreshToken.Token,
	}, nil
}

func (s *UserService) RevokeToken(ctx context.Context, tokenIDStr string) error {
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse token id: %w", err)
	}
	err = s.repo.RevokeRefreshToken(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}
