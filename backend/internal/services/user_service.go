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
	"sparkly/backend/internal/metadata"
)

type UserRepo interface {
	WriteNewUser(ctx context.Context, email, hashedPassword string) (domain.User, error)
	GetUser(ctx context.Context, email string) (domain.User, error)
	WriteRefreshToken(ctx context.Context, tokenID, userID uuid.UUID) error
	GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (domain.User, domain.RefreshToken, error)
}

type RefreshTokenRepo interface {
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error
}

type KeyManager interface {
	GetCurrSignKey(ctx context.Context) (*rsa.PrivateKey, string, error)
}

type UserService struct {
	userRepo         UserRepo
	refreshTokenRepo RefreshTokenRepo
	keyManager       KeyManager
}

func NewUserService(userRepo UserRepo, refreshTokenRepo RefreshTokenRepo, KeyManager KeyManager) *UserService {
	return &UserService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		keyManager:       KeyManager,
	}
}

type AuthToken struct {
	Access  string
	Refresh string
}

func (s *UserService) generateTokens(ctx context.Context, userID uuid.UUID, email string) (AuthToken, error) {
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

	err = s.userRepo.WriteRefreshToken(ctx, tokenID, userID)
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

	user, err := s.userRepo.WriteNewUser(ctx, email, hashed)
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
	user, err := s.userRepo.GetUser(ctx, email)
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

func (s *UserService) RevokeSession(ctx context.Context) error {
	tokenIDStr, err := metadata.TokenIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get refresh token id: %w", err)
	}

	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse token id: %w", err)
	}
	err = s.refreshTokenRepo.RevokeRefreshToken(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}
