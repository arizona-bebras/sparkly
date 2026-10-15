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
		return domain.User{}, fmt.Errorf("failed to write new user: %w", err)
	}

	return user.ToDomain(), nil
}

func (r *UserRepo) CheckUser(ctx context.Context, email string) (bool, error) {
	exists := &models.User{}

	if err := r.db.WithContext(ctx).First(&exists, "email = ?", email).Error; err == nil {
		return true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, fmt.Errorf("failed to check user: %w", err)
	}

	return false, nil
}

func (r *UserRepo) GetUser(ctx context.Context, email string) (domain.User, error) {
	user := &models.User{}

	if err := r.db.First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("failed to find user: %w", err)
	}

	return user.ToDomain(), nil
}
