package services

import (
	"context"
	"fmt"

	"sparkly/backend/internal/domain"

	"github.com/google/uuid"
)

type IssuesRepo interface {
	GetIssueByID(ctx context.Context, issueID, userID uuid.UUID) (domain.Issue, error)
	GetIssues(ctx context.Context, filters domain.IssueFilters, parkID, userID uuid.UUID) ([]domain.Issue, error)

	UpdateIssue(ctx context.Context, newData domain.UpdateIssueParams, issueID, userID uuid.UUID) (domain.Issue, error)
	ResolveIssue(ctx context.Context, issueID, userID uuid.UUID) (domain.Issue, error)

	DeleteIssue(ctx context.Context, issueID, userID uuid.UUID) error
}

type IssueService struct {
	repo IssuesRepo
}

func NewIssueService(repo IssuesRepo) *IssueService {
	return &IssueService{
		repo: repo,
	}
}

func (s *IssueService) GetIssue(ctx context.Context, issueIDStr string) (domain.Issue, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to parse user id from context: %w", err)
	}

	issueID, err := uuid.Parse(issueIDStr)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to parse issue id: %w", err)
	}

	issue, err := s.repo.GetIssueByID(ctx, issueID, userID)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to get issue: %w", err)
	}

	return issue, nil
}

func (s *IssueService) GetIssues(ctx context.Context, filters domain.IssueFilters, parkIDStr string) ([]domain.Issue, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return []domain.Issue{}, fmt.Errorf("failed to parse user id from context: %w", err)
	}

	parkID, err := uuid.Parse(parkIDStr)
	if err != nil {
		return []domain.Issue{}, fmt.Errorf("failed to parse issue id: %w", err)
	}

	issues, err := s.repo.GetIssues(ctx, filters, parkID, userID)
	if err != nil {
		return []domain.Issue{}, fmt.Errorf("failed to get issues: %w", err)
	}

	return issues, nil
}

func (s *IssueService) UpdateIssue(ctx context.Context, newData domain.UpdateIssueParams, issueIDStr string) (domain.Issue, error) {

	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to parse user id from context: %w", err)
	}

	issueID, err := uuid.Parse(issueIDStr)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to parse issue id: %w", err)
	}

	issue, err := s.repo.UpdateIssue(ctx, newData, issueID, userID)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to update issue: %w", err)
	}

	return issue, nil
}

func (s *IssueService) ResolveIssue(ctx context.Context, issueIDStr string) (domain.Issue, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to parse user id from context: %w", err)
	}

	issueID, err := uuid.Parse(issueIDStr)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to parse issue id: %w", err)
	}

	issue, err := s.repo.ResolveIssue(ctx, issueID, userID)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("failed to resolve issue: %w", err)
	}

	return issue, nil
}

func (s *IssueService) DeleteIssue(ctx context.Context, issueIDStr string) error {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse user id from context: %w", err)
	}

	issueID, err := uuid.Parse(issueIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse issue id: %w", err)
	}

	err = s.repo.DeleteIssue(ctx, issueID, userID)
	if err != nil {
		return fmt.Errorf("failed to resolve issue: %w", err)
	}

	return nil
}
