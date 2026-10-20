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
}

type KeyManager interface {
	GetCurrSignKey(ctx context.Context) (*rsa.PrivateKey, error)
}

type UserService struct {
	repo       UserRepo
	keyManager KeyManager
}

func NewUserService(userRepo UserRepo, KeyManager KeyManager) *UserService {
	return &UserService{
		repo: userRepo,
		keyManager: KeyManager,
	}
}

type AuthToken struct {
	Access  string
	Refresh string
}

func (s *UserService) generateTokens(ctx context.Context, id uuid.UUID, email string) (AuthToken, error) {
	signKey, err := s.keyManager.GetCurrSignKey(ctx)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to get private key: %w", err)
	}

	now := time.Now()

	accessClaims := jwt.MapClaims{
		"id":    id.String(),
		"email": email,
		"exp":   now.Add(24 * time.Hour).Unix(),
		"iat":   now.Unix(),
		"type":  "access",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessString, err := accessToken.SignedString(signKey)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshClaims := jwt.MapClaims{
		"id":    id.String(),
		"email": email,
		"exp":   now.Add(7 * 24 * time.Hour).Unix(),
		"iat":   now.Unix(),
		"type":  "refresh",
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshString, err := refreshToken.SignedString(signKey)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to sign refresh token: %w", err)
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
