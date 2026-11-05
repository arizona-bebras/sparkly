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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IssuesRepo struct {
	db *gorm.DB
}

func NewIssuesRepo(db *gorm.DB) *IssuesRepo {
	return &IssuesRepo{
		db: db,
	}
}

func (r *IssuesRepo) GetIssueByID(ctx context.Context, issueID, userID uuid.UUID) (domain.Issue, error) {
	issue := models.Issue{
		ID: issueID,
	}

	if err := r.db.WithContext(ctx).Scopes(scope.UserIssueScope(userID)).First(&issue).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Issue{}, domain.ErrIssueNotFound
		}
		return domain.Issue{}, fmt.Errorf("failed to get issue from db: %w", err)
	}

	return issue.ToDomain(), nil
}

const SIMILARITY_THRESHOLD = 0.3

func (r *IssuesRepo) GetIssues(ctx context.Context, filters domain.IssueFilters, parkID, userID uuid.UUID) ([]domain.Issue, error) {
	var severity *models.IssueSeverity
	if filters.Severity == nil {
		severity = nil
	} else {
		tmpS := models.IssueSeverity(*filters.Severity)
		severity = &tmpS
	}
	strictFilters := &struct {
		ParkID     *uuid.UUID
		Severity   *models.IssueSeverity
		IsResolved *bool
	}{
		ParkID:     &parkID,
		Severity:   severity,
		IsResolved: filters.IsResolved,
	}

	query := r.db.WithContext(ctx).Scopes(scope.UserIssueScope(userID)).Where(&strictFilters)

	if filters.FromTime != nil {
		query = query.Where("first_at >= ?", filters.FromTime)
	}

	if filters.ToTime != nil {
		query = query.Where("last_at <= ?", filters.ToTime)
	}

	if filters.Title != nil {
		query = query.Select("*, similarity(title, ?) as sim", filters.Title).
			Where("similarity(title, ?) >= ?", filters.Title, SIMILARITY_THRESHOLD).
			Order("sim DESC")
	}

	var issues []models.Issue
	if err := query.Find(&issues).Error; err != nil {
		return []domain.Issue{}, fmt.Errorf("failed to get issues from db: %w", err)
	}

	domainIssues := make([]domain.Issue, 0, len(issues))
	for _, issue := range issues {
		domainIssues = append(domainIssues, issue.ToDomain())
	}
	return domainIssues, nil
}

func (r *IssuesRepo) UpdateIssue(ctx context.Context, newData domain.UpdateIssueParams, issueID, userID uuid.UUID) (domain.Issue, error) {
	issue := models.Issue{
		ID: issueID,
	}

	if upd := r.db.WithContext(ctx).Scopes(scope.UserIssueScope(userID)).
		Model(&issue).
		Clauses(clause.Returning{}).
		Updates(&newData); upd.Error != nil {
		return domain.Issue{}, fmt.Errorf("failed to update issue in db: %w", upd.Error)
	} else if upd.RowsAffected == 0 {
		return domain.Issue{}, domain.ErrIssueNotFound
	}

	return issue.ToDomain(), nil
}

func (r *IssuesRepo) ResolveIssue(ctx context.Context, issueID, userID uuid.UUID) (domain.Issue, error) {
	issue := models.Issue{
		ID: issueID,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.db.Scopes(scope.UserIssueScope(userID)).First(&issue).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrIssueNotFound
			}
			return fmt.Errorf("failed to update issue in db: %w", err)
		}

		if issue.IsResolved {
			return domain.ErrIssueAlreadyResolved
		}

		if err := r.db.WithContext(ctx).Scopes(scope.UserIssueScope(userID)).
			Model(&issue).
			Clauses(clause.Returning{}).
			Updates(&models.Issue{IsResolved: true, ResolvedAt: time.Now()}).Error; err != nil {
			return fmt.Errorf("failed to update issue in db: %w", err)
		}

		return nil
	})

	if err != nil {
		return domain.Issue{}, err
	}

	return issue.ToDomain(), nil
}

func (r *IssuesRepo) DeleteIssue(ctx context.Context, issueID, userID uuid.UUID) error {
	issue := models.Issue{
		ID: issueID,
	}

	if del := r.db.WithContext(ctx).Scopes(scope.UserIssueScope(userID)).Delete(&issue); del.Error != nil {
		return fmt.Errorf("failed to delete issue from db: %w", del.Error)
	} else if del.RowsAffected == 0 {
		return domain.ErrIssueNotFound
	}

	return nil
}
