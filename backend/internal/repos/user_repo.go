package repos

import (
	"context"
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
		ID: id,
		Email: email,
		Password: hashedPassword,
	}

	checkQ := r.db.WithContext(ctx).Find(&models.User{Email: email})
	if checkQ.Error != nil {
		return domain.User{}, fmt.Errorf("failed to write new user: %w", checkQ.Error)
	}
	if checkQ.RowsAffected > 0 {
		return domain.User{}, domain.ErrDuplicatedUser
	}
	
	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return domain.User{}, fmt.Errorf("failed to write new user: %w", err)
	}

	return user.ToDomain(), nil
}