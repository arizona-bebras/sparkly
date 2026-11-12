package handlers

import (
	"context"
	"errors"
	"time"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"

	pb "sparkly/backend/proto/v1"
	"sparkly/backend/proto/v1/backendv1connect"

	"connectrpc.com/connect/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReleaseHandler struct {
	backendv1connect.UnimplementedReleaseServiceHandler
	service *services.ReleaseService
}

func NewReleaseHandler(s *services.ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{
		service: s,
	}
}

func releaseDomainToResp(r domain.Release) *pb.Release {
	return &pb.Release{
		Id:        r.ID.String(),
		ParkId:    r.ParkID.String(),
		Name:      r.Name,
		Version:   r.Version,
		CreatedAt: timestamppb.New(r.CreatedAt),
	}
}

func (h *ReleaseHandler) CreateRelease(ctx context.Context, req *pb.CreateReleaseRequest) (*pb.CreateReleaseResponse, error) {
	params := domain.CreateReleaseParams{
		Name:    req.GetName(),
		Version: req.GetVersion(),
	}

	release, err := h.service.CreateRelease(ctx, req.GetParkId(), params)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid park id")
		}
		if errors.Is(err, domain.ErrParkNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "park with this id doesnt exists")
		}
		if errors.Is(err, domain.ErrParkAccessDenied) {
			return nil, connect.NewError(connect.CodePermissionDenied, "access denied")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.CreateReleaseResponse{
		Release: releaseDomainToResp(release),
	}, nil
}

func (h *ReleaseHandler) GetRelease(ctx context.Context, req *pb.GetReleaseRequest) (*pb.GetReleaseResponse, error) {
	release, err := h.service.GetRelease(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid release id")
		}
		if errors.Is(err, domain.ErrReleaseNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "release with this id doesnt exists")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.GetReleaseResponse{
		Release: releaseDomainToResp(release),
	}, nil
}

func (h *ReleaseHandler) GetReleases(ctx context.Context, req *pb.GetReleasesRequest) (*pb.GetReleasesResponse, error) {
	var fromTime *time.Time
	if req.FromTime != nil {
		t := req.FromTime.AsTime()
		fromTime = &t
	}
	var toTime *time.Time
	if req.ToTime != nil {
		t := req.ToTime.AsTime()
		toTime = &t
	}

	filters := domain.GetReleasesFilters{
		Name:     req.Name,
		Version:  req.Version,
		FromTime: fromTime,
		ToTime:   toTime,
	}

	releases, err := h.service.GetReleases(ctx, req.GetParkId(), filters)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid park id")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	pbReleases := make([]*pb.Release, len(releases))
	for i, r := range releases {
		pbReleases[i] = releaseDomainToResp(r)
	}

	return &pb.GetReleasesResponse{
		Releases: pbReleases,
	}, nil
}

func (h *ReleaseHandler) UpdateRelease(ctx context.Context, req *pb.UpdateReleaseRequest) (*pb.UpdateReleaseResponse, error) {
	params := domain.UpdateReleaseParams{
		Name:    req.Name,
		Version: req.Version,
	}

	release, err := h.service.UpdateRelease(ctx, req.GetId(), params)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid release id")
		}
		if errors.Is(err, domain.ErrReleaseNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "release with this id doesnt exists")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.UpdateReleaseResponse{
		Release: releaseDomainToResp(release),
	}, nil
}

func (h *ReleaseHandler) DeleteRelease(ctx context.Context, req *pb.DeleteReleaseRequest) (*pb.DeleteReleaseResponse, error) {
	err := h.service.DeleteRelease(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid release id")
		}
		if errors.Is(err, domain.ErrReleaseNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "release with this id doesnt exists")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.DeleteReleaseResponse{
		Success: true,
	}, nil
}
