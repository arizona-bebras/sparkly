package repos

import (
	"context"
	"errors"
	"fmt"

	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/db/scope"
	"sparkly/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupsRepo struct {
	db *gorm.DB
}

func NewGroupsRepo(db *gorm.DB) *GroupsRepo {
	return &GroupsRepo{
		db: db,
	}
}

func (r *GroupsRepo) CreateGroup(ctx context.Context, userID uuid.UUID, data domain.CreateGroupParams) (domain.Group, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to generate new id: %w", err)
	}

	newGroup := models.Group{
		ID:          id,
		Name:        data.Name,
		Description: data.Description,
		ParkID:      data.ParkID,
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

		if err := tx.Create(&newGroup).Error; err != nil {
			var pgErr *pgconn.PgError
			if errors.Is(err, gorm.ErrForeignKeyViolated) || (errors.As(err, &pgErr) && pgErr.Code == "23503") {
				return domain.ErrParkNotFound
			}
			return fmt.Errorf("failed to write new group to db: %w", err)
		}

		return nil
	})

	if err != nil {
		return domain.Group{}, err
	}

	return newGroup.ToDomain(), nil
}

func (r *GroupsRepo) GetGroup(ctx context.Context, groupID, userID uuid.UUID) (domain.Group, error) {
	var group models.Group
	if err := r.db.WithContext(ctx).Scopes(scope.UserGroupScope(userID)).First(&group, "id = ?", groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Group{}, domain.ErrGroupNotFound
		}
		return domain.Group{}, fmt.Errorf("failed to find group in db: %w", err)
	}

	return group.ToDomain(), nil
}

func (r *GroupsRepo) GetAllGroups(ctx context.Context, parkID, userID uuid.UUID) ([]domain.Group, error) {
	var groups []models.Group
	if err := r.db.WithContext(ctx).Scopes(scope.UserGroupScope(userID)).Find(&groups, "park_id = ?", parkID).Error; err != nil {
		return nil, fmt.Errorf("failed to get groups from db: %w", err)
	}

	domainGroups := make([]domain.Group, len(groups))
	for i, group := range groups {
		domainGroups[i] = group.ToDomain()
	}
	return domainGroups, nil
}

func (r *GroupsRepo) UpdateGroup(ctx context.Context, newData domain.UpdateGroupParams, groupID, userID uuid.UUID) (domain.Group, error) {
	newGroup := models.Group{
		ID: groupID,
	}

	updates := make(map[string]any)
	if newData.Name != "" {
		updates["name"] = newData.Name
	}
	if newData.Description != "" {
		updates["description"] = newData.Description
	}

	if len(updates) == 0 {
		return r.GetGroup(ctx, groupID, userID)
	}

	upd := r.db.WithContext(ctx).Scopes(scope.UserGroupScope(userID)).Model(&newGroup).
		Clauses(clause.Returning{}).
		Updates(updates)

	if upd.Error != nil {
		return domain.Group{}, fmt.Errorf("update group db error: %w", upd.Error)
	} else if upd.RowsAffected == 0 {
		return domain.Group{}, domain.ErrGroupNotFound
	}

	return newGroup.ToDomain(), nil
}

func (r *GroupsRepo) DeleteGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	del := r.db.WithContext(ctx).Scopes(scope.UserGroupScope(userID)).Delete(&models.Group{ID: groupID})
	if del.Error != nil {
		return fmt.Errorf("delete group db error: %w", del.Error)
	} else if del.RowsAffected == 0 {
		return domain.ErrGroupNotFound
	}

	return nil
}
