package repos

import (
	"context"
	"errors"
	"fmt"
	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{
		db: db,
	}
}

func (r *RefreshTokenRepo) GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (domain.User, domain.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.WithContext(ctx).Preload("User").First(&token, "id = ?", tokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.RefreshToken{}, domain.ErrTokenNotFound
		}
		return domain.User{}, domain.RefreshToken{}, fmt.Errorf("failed to get token from db: %w", err)
	}
	return token.User.ToDomain(), token.ToDomain(), nil
}

func (r *RefreshTokenRepo) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	var token models.RefreshToken

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).First(&token, "id = ?", tokenID).Error; err != nil {
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
