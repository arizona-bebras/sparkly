package repos

import (
	"context"
	"errors"
	"fmt"

	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/db/scope"
	"sparkly/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ParkRepo struct {
	db *gorm.DB
}

func NewParkRepo(db *gorm.DB) *ParkRepo {
	return &ParkRepo{
		db: db,
	}
}

func (r *ParkRepo) CreatePark(ctx context.Context, userID uuid.UUID, data domain.CreateParkParams) (domain.Park, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to gen park id: %w", err)
	}

	park := models.Park{
		ID:          id,
		Name:        data.Name,
		Description: data.Description,
		TokenHash:   data.TokenHash,
	}

	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&park).Error; err != nil {
			return fmt.Errorf("create park db error: %w", err)
		}
		if err := tx.Model(&models.User{ID: userID}).Association("Parks").Append(&park); err != nil {
			return fmt.Errorf("failed to assign user to park: %w", err)
		}

		return nil
	})

	if err != nil {
		return domain.Park{}, err
	}

	return park.ToDomain(), nil
}

func (r *ParkRepo) GetParkByID(ctx context.Context, userID, parkID uuid.UUID) (domain.Park, error) {
	var park models.Park
	if err := r.db.Scopes(scope.UserParkScope(userID)).First(&park, "id = ?", parkID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Park{}, domain.ErrParkNotFound
		}
		return domain.Park{}, fmt.Errorf("get park db error: %w", err)
	}

	return park.ToDomain(), nil
}

func (r *ParkRepo) GetAllParks(ctx context.Context, userID uuid.UUID) ([]domain.Park, error) {
	var parks []models.Park
	if err := r.db.Scopes(scope.UserParkScope(userID)).Find(&parks).Error; err != nil {
		return []domain.Park{}, fmt.Errorf("get parks db error: %w", err)
	}

	domainParks := make([]domain.Park, len(parks))
	for i, parkModel := range parks {
		domainParks[i] = parkModel.ToDomain()
	}
	return domainParks, nil
}

func (r *ParkRepo) UpdatePark(ctx context.Context, userID uuid.UUID, parkID uuid.UUID, data domain.UpdateParkParams) (domain.Park, error) {
	park := models.Park{
		ID: parkID,
	}

	if upd := r.db.Scopes(scope.UserParkScope(userID)).Model(&park).Clauses(clause.Returning{}).Updates(models.Park{
		Name:        data.Name,
		Description: data.Description,
	}); upd.Error != nil {
		return domain.Park{}, fmt.Errorf("update park db error: %w", upd.Error)
	} else if upd.RowsAffected == 0 {
		return domain.Park{}, domain.ErrParkNotFound
	}

	return park.ToDomain(), nil
}

func (r *ParkRepo) DeletePark(ctx context.Context, userID, parkID uuid.UUID) error {
	if del := r.db.Scopes(scope.UserParkScope(userID)).Delete(&models.Park{ID: parkID}); del.Error != nil {
		return fmt.Errorf("delete parks db error: %w", del.Error)
	} else if del.RowsAffected == 0 {
		return domain.ErrParkNotFound
	}

	return nil
}
