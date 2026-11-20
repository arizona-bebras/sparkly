package repos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/db/scope"
	"sparkly/backend/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BuildsRepo struct {
	fs *s3.Client
	db *gorm.DB
}

func NewBuildsRepo(db *gorm.DB, fs *s3.Client) *BuildsRepo {
	return &BuildsRepo{
		fs: fs,
		db: db,
	}
}

func (r *BuildsRepo) CreateBuild(ctx context.Context, userID uuid.UUID, data domain.CreateBuildParams) (domain.Build, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to generate new id: %w", err)
	}

	newBuild := models.Build{
		ID:        id,
		ParkID:    data.ParkID,
		ReleaseID: data.ReleaseID,
		CreatedAt: time.Now(),
	}
	newBuild.ElfObjectKey = "elf-" + id.String()
	if data.Commit != nil {
		newBuild.Commit = *data.Commit
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

		if data.ReleaseID != nil {
			var releases []models.Release
			if err := tx.Model(&park).Association("Releases").Find(&releases, "id = ?", *data.ReleaseID); err != nil {
				return fmt.Errorf("failed to get associated releases from db: %w", err)
			}

			if len(releases) == 0 {
				return domain.ErrReleaseNotFound
			}
		}

		if len(users) == 0 {
			return domain.ErrParkAccessDenied
		}

		if err := tx.Create(&newBuild).Error; err != nil {
			return fmt.Errorf("failed to write new build to db: %w", err)
		}

		_, err = r.fs.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(domain.ELF_OBJECTS_BUCKET_NAME),
			Key:    aws.String(newBuild.ElfObjectKey),
			Body:   bytes.NewReader(data.ElfObject),
		})

		if err != nil {
			return fmt.Errorf("failed to write build elf-object to rustfs storage: %w", err)
		}

		return nil
	})

	if err != nil {
		return domain.Build{}, err
	}

	return newBuild.ToDomain(), nil
}

func (r *BuildsRepo) GetBuild(ctx context.Context, buildID, userID uuid.UUID) (domain.Build, error) {
	var build models.Build
	if err := r.db.WithContext(ctx).Scopes(scope.UserBuildScope(userID)).First(&build, "id = ?", buildID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Build{}, domain.ErrBuildNotFound
		}
		return domain.Build{}, fmt.Errorf("failed to find build in db: %w", err)
	}

	return build.ToDomain(), nil
}

func (r *BuildsRepo) GetBuilds(ctx context.Context, filters domain.GetBuildsFilters, userID uuid.UUID) ([]domain.Build, error) {
	query := r.db.WithContext(ctx).Scopes(scope.UserBuildScope(userID)).Where("builds.park_id = ?", filters.ParkID)

	if filters.ReleaseID != nil {
		id := *filters.ReleaseID
		query = query.Where("builds.version LIKE ?", "%"+id.String()+"%")
	}

	if filters.Commit != nil {
		query = query.Where("builds.commit LIKE ?", "%"+*filters.Commit+"%")
	}

	if filters.FromTime != nil {
		query = query.Where("builds.created_at >= ?", *filters.FromTime)
	}

	if filters.ToTime != nil {
		query = query.Where("builds.created_at <= ?", *filters.ToTime)
	}

	var builds []models.Build
	if err := query.Find(&builds).Error; err != nil {
		return nil, fmt.Errorf("failed to get builds from db: %w", err)
	}

	domainBuilds := make([]domain.Build, len(builds))
	for i, b := range builds {
		domainBuilds[i] = b.ToDomain()
	}
	return domainBuilds, nil
}

func (r *BuildsRepo) UpdateBuild(ctx context.Context, newData domain.UpdateBuildParams, buildID, userID uuid.UUID) (domain.Build, error) {
	newBuild := models.Build{
		ID: buildID,
	}

	updates := make(map[string]any)
	if newData.ReleaseID != nil {
		id := *newData.ReleaseID
		updates["release_id"] = id.String()
	}
	if newData.Commit != nil {
		updates["commit"] = *newData.Commit
	}

	if len(updates) == 0 {
		return r.GetBuild(ctx, buildID, userID)
	}

	upd := r.db.WithContext(ctx).Scopes(scope.UserBuildScope(userID)).Model(&newBuild).
		Clauses(clause.Returning{}).
		Updates(updates)

	if upd.Error != nil {
		return domain.Build{}, fmt.Errorf("update build db error: %w", upd.Error)
	} else if upd.RowsAffected == 0 {
		return domain.Build{}, domain.ErrBuildNotFound
	}

	return newBuild.ToDomain(), nil
}

func (r *BuildsRepo) DeleteBuild(ctx context.Context, buildID, userID uuid.UUID) error {
	del := r.db.WithContext(ctx).Scopes(scope.UserBuildScope(userID)).Delete(&models.Build{ID: buildID})
	if del.Error != nil {
		return fmt.Errorf("delete build db error: %w", del.Error)
	} else if del.RowsAffected == 0 {
		return domain.ErrBuildNotFound
	}

	return nil
}
