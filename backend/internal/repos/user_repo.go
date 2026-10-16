package repos

import (
	"context"
	"errors"
	"fmt"
	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/domain"

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

	if err := r.db.First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("failed to find user: %w", err)
	}

	return user.ToDomain(), nil
}
