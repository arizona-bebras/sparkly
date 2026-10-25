package repos

import (
	"context"
	"errors"
	"fmt"
	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

func (r *UserRepo) WriteNewUser(ctx context.Context, email, hashedPassword string) (domain.User, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to generate new id: %w", err)
	}

	user := models.User{
		ID:       id,
		Email:    email,
		Password: hashedPassword,
	}

	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
			return domain.User{}, domain.ErrDuplicatedUser
		}
		return domain.User{}, fmt.Errorf("failed to write new user: %w", err)
	}

	return user.ToDomain(), nil
}

func (r *UserRepo) GetUser(ctx context.Context, email string) (domain.User, error) {
	user := &models.User{}

	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("failed to find user: %w", err)
	}

	return user.ToDomain(), nil
}

func (r *UserRepo) WriteRefreshToken(ctx context.Context, refreshString string, tokenID, userID uuid.UUID) error {
	token := models.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		Token:     refreshString,
		Revoked:   false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(domain.REFRESH_TOKEN_LIFETIME),
	}

	if err := r.db.WithContext(ctx).Create(&token).Error; err != nil {
		return fmt.Errorf("failed to write token: %w", err)
	}

	return nil
}

func (r *UserRepo) GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (domain.User, domain.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.WithContext(ctx).Preload("User").First(&token, "id = ?", tokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.RefreshToken{}, domain.ErrTokenNotFound
		}
		return domain.User{}, domain.RefreshToken{}, fmt.Errorf("failed to get token from db: %w", err)
	}
	return token.User.ToDomain(), token.ToDomain(), nil
}

func (r *UserRepo) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	token := models.RefreshToken{
		ID: tokenID,
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).First(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrTokenNotFound
			}
			return fmt.Errorf("failed to get token from db: %w", err)
		}

		if token.Revoked {
			return domain.ErrRevokedRefreshToken
		}

		if err := tx.WithContext(ctx).Model(&token).Update("revoked", true).Error; err != nil {
			return fmt.Errorf("failed to update token in db: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
