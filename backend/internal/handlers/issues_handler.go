package handlers

import (
	"context"
	"errors"
	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"
	"time"

	pb "sparkly/backend/proto/v1"
	"sparkly/backend/proto/v1/backendv1connect"

	"connectrpc.com/connect/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func pbSeverityToDomain(is *pb.IssueSeverity) *domain.IssueSeverity {
	if is == nil {
		return nil
	}

	var s domain.IssueSeverity
	switch *is {
	case pb.IssueSeverity_ISSUE_SEVERITY_UNSPECIFIED:
		return nil
	case pb.IssueSeverity_ISSUE_SEVERITY_INFO:
		s = domain.IssueSeverityInfo
	case pb.IssueSeverity_ISSUE_SEVERITY_LOW:
		s = domain.IssueSeverityLow
	case pb.IssueSeverity_ISSUE_SEVERITY_MEDIUM:
		s = domain.IssueSeverityMedium
	case pb.IssueSeverity_ISSUE_SEVERITY_HIGH:
		s = domain.IssueSeverityHigh
	case pb.IssueSeverity_ISSUE_SEVERITY_CRITICAL:
		s = domain.IssueSeverityCritical
	}

	return &s
}

func domainSeverityToPB(is domain.IssueSeverity) pb.IssueSeverity {

	switch is {
	case domain.IssueSeverityInfo:
		return pb.IssueSeverity_ISSUE_SEVERITY_INFO
	case domain.IssueSeverityLow:
		return pb.IssueSeverity_ISSUE_SEVERITY_LOW
	case domain.IssueSeverityMedium:
		return pb.IssueSeverity_ISSUE_SEVERITY_MEDIUM
	case domain.IssueSeverityHigh:
		return pb.IssueSeverity_ISSUE_SEVERITY_HIGH
	case domain.IssueSeverityCritical:
		return pb.IssueSeverity_ISSUE_SEVERITY_CRITICAL
	}

	return pb.IssueSeverity_ISSUE_SEVERITY_UNSPECIFIED
}

func fromDomainToResp(issue domain.Issue) *pb.Issue {
	return &pb.Issue{
		Id:          issue.ID.String(),
		ParkId:      issue.ParkID.String(),
		Title:       issue.Title,
		Description: issue.Description,
		Severity:    domainSeverityToPB(issue.Severity),
		IsResolved:  issue.IsResolved,
		ResolvedAt:  timestamppb.New(issue.ResolvedAt),
		FirstAt:     timestamppb.New(issue.FirstAt),
		LastAt:      timestamppb.New(issue.LastAt),
	}
}

type IssuesHandler struct {
	backendv1connect.UnimplementedIssueServiceHandler
	service *services.IssueService
}

func NewIssuesHandler(s *services.IssueService) *IssuesHandler {
	return &IssuesHandler{
		service: s,
	}
}

func (h *IssuesHandler) GetIssue(ctx context.Context, req *pb.GetIssueRequest) (*pb.GetIssueResponse, error) {
	issueID := req.GetId()
	issue, err := h.service.GetIssue(ctx, issueID)
	if err != nil {
		if errors.Is(err, domain.ErrIssueNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "issue not found")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.GetIssueResponse{
		Issue: fromDomainToResp(issue),
	}, nil
}

func (h *IssuesHandler) GetIssues(ctx context.Context, req *pb.GetIssuesRequest) (*pb.GetIssuesResponse, error) {
	parkID := req.GetParkId()

	var fromTime *time.Time
	if req.FromTime == nil {
		fromTime = nil
	} else {
		t := req.FromTime.AsTime()
		fromTime = &t
	}

	var toTime *time.Time
	if req.ToTime == nil {
		toTime = nil
	} else {
		t := req.ToTime.AsTime()
		toTime = &t
	}

	issues, err := h.service.GetIssues(ctx, domain.IssueFilters{
		Title:      req.Title,
		Severity:   pbSeverityToDomain(req.Severity),
		IsResolved: req.IsResolved,
		FromTime:   fromTime,
		ToTime:     toTime,
	}, parkID)

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	isseusResp := make([]*pb.Issue, 0, len(issues))
	for _, issue := range issues {
		isseusResp = append(isseusResp, fromDomainToResp(issue))
	}

	return &pb.GetIssuesResponse{
		Issues: isseusResp,
	}, nil
}

func (h *IssuesHandler) UpdateIssue(ctx context.Context, req *pb.UpdateIssueRequest) (*pb.UpdateIssueResponse, error) {
	id := req.GetId()

	issue, err := h.service.UpdateIssue(ctx, domain.UpdateIssueParams{
		Title:       req.Title,
		Description: req.Description,
		Severity:    pbSeverityToDomain(req.Severity),
	}, id)
	if err != nil {
		if errors.Is(err, domain.ErrIssueNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "issue not found")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.UpdateIssueResponse{
		Issue: fromDomainToResp(issue),
	}, nil
}

func (h *IssuesHandler) ResolveIssue(ctx context.Context, req *pb.ResolveIssueRequest) (*pb.ResolveIssueResponse, error) {
	id := req.GetId()

	issue, err := h.service.ResolveIssue(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrIssueNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "issue not found")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.ResolveIssueResponse{
		Issue: fromDomainToResp(issue),
	}, nil
}

func (h *IssuesHandler) DeleteIssue(ctx context.Context, req *pb.DeleteIssueRequest) (*pb.DeleteIssueResponse, error) {
	id := req.GetId()

	err := h.service.DeleteIssue(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrIssueNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "issue not found")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.DeleteIssueResponse{
		Success: true,
	}, nil
}
