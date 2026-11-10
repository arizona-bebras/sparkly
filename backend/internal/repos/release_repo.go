package repos

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/db/scope"
	"sparkly/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReleaseRepo struct {
	db *gorm.DB
}

func NewReleaseRepo(db *gorm.DB) *ReleaseRepo {
	return &ReleaseRepo{
		db: db,
	}
}

func (r *ReleaseRepo) CreateRelease(ctx context.Context, userID uuid.UUID, data domain.CreateReleaseParams) (domain.Release, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to generate new id: %w", err)
	}

	newRelease := models.Release{
		ID:        id,
		ParkID:    data.ParkID,
		Name:      data.Name,
		Version:   data.Version,
		CreatedAt: time.Now(),
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var park models.Park
		if err := tx.First(&park, "id = ?", data.ParkID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrParkNotFound
			}
			return fmt.Errorf("failed to find park in db: %w", err)
		}

		var users []models.User
		if err := tx.Model(&park).Association("Users").Find(&users, "id = ?", userID); err != nil {
			return fmt.Errorf("failed to verify user access: %w", err)
		}

		if len(users) == 0 {
			return domain.ErrParkAccessDenied
		}

		if err := tx.Create(&newRelease).Error; err != nil {
			var pgErr *pgconn.PgError
			if errors.Is(err, gorm.ErrForeignKeyViolated) || (errors.As(err, &pgErr) && pgErr.Code == "23503") {
				return domain.ErrParkNotFound
			}
			return fmt.Errorf("failed to write new release to db: %w", err)
		}

		return nil
	})

	if err != nil {
		return domain.Release{}, err
	}

	return newRelease.ToDomain(), nil
}

func (r *ReleaseRepo) GetRelease(ctx context.Context, releaseID, userID uuid.UUID) (domain.Release, error) {
	var release models.Release
	if err := r.db.WithContext(ctx).Scopes(scope.UserReleaseScope(userID)).First(&release, "id = ?", releaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Release{}, domain.ErrReleaseNotFound
		}
		return domain.Release{}, fmt.Errorf("failed to find release in db: %w", err)
	}

	return release.ToDomain(), nil
}

func (r *ReleaseRepo) GetReleases(ctx context.Context, filters domain.GetReleasesFilters, userID uuid.UUID) ([]domain.Release, error) {
	query := r.db.WithContext(ctx).Scopes(scope.UserReleaseScope(userID)).Where("releases.park_id = ?", filters.ParkID)

	if filters.Version != nil {
		query = query.Where("releases.version LIKE ?", "%"+*filters.Version+"%")
	}

	if filters.Name != nil {
		query = query.Select("releases.*, similarity(releases.name, ?) as sim", *filters.Name).
			Where("similarity(releases.name, ?) > ?", *filters.Name, SIMILARITY_THRESHOLD).
			Order("sim DESC")
	}

	if filters.FromTime != nil {
		query = query.Where("releases.created_at >= ?", *filters.FromTime)
	}

	if filters.ToTime != nil {
		query = query.Where("releases.created_at <= ?", *filters.ToTime)
	}

	var releases []models.Release
	if err := query.Find(&releases).Error; err != nil {
		return nil, fmt.Errorf("failed to get releases from db: %w", err)
	}

	domainReleases := make([]domain.Release, len(releases))
	for i, rel := range releases {
		domainReleases[i] = rel.ToDomain()
	}
	return domainReleases, nil
}

func (r *ReleaseRepo) UpdateRelease(ctx context.Context, newData domain.UpdateReleaseParams, releaseID, userID uuid.UUID) (domain.Release, error) {
	newRelease := models.Release{
		ID: releaseID,
	}

	updates := make(map[string]any)
	if newData.Name != nil {
		updates["name"] = *newData.Name
	}
	if newData.Version != nil {
		updates["version"] = *newData.Version
	}

	if len(updates) == 0 {
		return r.GetRelease(ctx, releaseID, userID)
	}

	upd := r.db.WithContext(ctx).Scopes(scope.UserReleaseScope(userID)).Model(&newRelease).
		Clauses(clause.Returning{}).
		Updates(updates)

	if upd.Error != nil {
		return domain.Release{}, fmt.Errorf("update release db error: %w", upd.Error)
	} else if upd.RowsAffected == 0 {
		return domain.Release{}, domain.ErrReleaseNotFound
	}

	return newRelease.ToDomain(), nil
}

func (r *ReleaseRepo) DeleteRelease(ctx context.Context, releaseID, userID uuid.UUID) error {
	del := r.db.WithContext(ctx).Scopes(scope.UserReleaseScope(userID)).Delete(&models.Release{ID: releaseID})
	if del.Error != nil {
		return fmt.Errorf("delete release db error: %w", del.Error)
	} else if del.RowsAffected == 0 {
		return domain.ErrReleaseNotFound
	}

	return nil
}
