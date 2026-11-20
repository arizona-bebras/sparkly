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
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BuildsHandler struct {
	backendv1connect.UnimplementedBuildServiceHandler
	service *services.BuildService
}

func NewBuildsHandler(s *services.BuildService) *BuildsHandler {
	return &BuildsHandler{
		service: s,
	}
}

func buildDomainToResp(b domain.Build) *pb.Build {
	var releaseID string
	if b.ReleaseID == uuid.Nil {
		releaseID = ""
	} else {
		releaseID = b.ReleaseID.String()
	}
	return &pb.Build{
		Id:             b.ID.String(),
		ParkId:         b.ParkID.String(),
		ReleaseId:      releaseID,
		Commit:         b.Commit,
		ElfObjectKey:   b.ElfObjectKey,
		DwarfObjectKey: b.DwarfObjectKey,
		CreatedAt:      timestamppb.New(b.CreatedAt),
	}
}

func (h *BuildsHandler) CreateBuild(ctx context.Context, req *pb.CreateBuildRequest) (*pb.CreateBuildResponse, error) {
	var commitP *string
	if req.Commit != nil {
		commitP = req.Commit
	}

	var releaseIDP *string
	if req.ReleaseId != nil {
		releaseIDP = req.ReleaseId
	}
	elfP := req.GetElfObject()

	build, err := h.service.CreateBuild(ctx, services.CreateBuildDTO{
		ParkID:    req.GetParkId(),
		ReleaseID: releaseIDP,
		Commit:    commitP,
		ElfObject: elfP,
	})
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
		if errors.Is(err, domain.ErrReleaseNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "release with this id doesnt exists")
		}
		if errors.Is(err, domain.ErrParkAccessDenied) {
			return nil, connect.NewError(connect.CodePermissionDenied, "access denied")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.CreateBuildResponse{
		Build: buildDomainToResp(build),
	}, nil
}

func (h *BuildsHandler) GetBuild(ctx context.Context, req *pb.GetBuildRequest) (*pb.GetBuildResponse, error) {
	build, err := h.service.GetBuild(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid build id")
		}
		if errors.Is(err, domain.ErrBuildNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "build with this id doesnt exists")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.GetBuildResponse{
		Build: buildDomainToResp(build),
	}, nil
}

func (h *BuildsHandler) GetBuilds(ctx context.Context, req *pb.GetBuildsRequest) (*pb.GetBuildsResponse, error) {
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

	builds, err := h.service.GetBuilds(ctx, services.BuildFiltersDTO{
		ParkID:    req.GetParkId(),
		ReleaseID: req.ReleaseId,
		Commit:    req.Commit,
		FromTime:  fromTime,
		ToTime:    toTime,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid park id")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	pbBuilds := make([]*pb.Build, len(builds))
	for i, b := range builds {
		pbBuilds[i] = buildDomainToResp(b)
	}

	return &pb.GetBuildsResponse{
		Builds: pbBuilds,
	}, nil
}

func (h *BuildsHandler) UpdateBuild(ctx context.Context, req *pb.UpdateBuildRequest) (*pb.UpdateBuildResponse, error) {
	build, err := h.service.UpdateBuild(ctx, services.UpdateBuildDTO{
		BuildID:   req.GetId(),
		ReleaseID: req.ReleaseId,
		Commit:    req.Commit,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid build id")
		}
		if errors.Is(err, domain.ErrBuildNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "build with this id doesnt exists")
		}
		if errors.Is(err, domain.ErrReleaseNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "release with this id doesnt exists")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.UpdateBuildResponse{
		Build: buildDomainToResp(build),
	}, nil
}

func (h *BuildsHandler) DeleteBuild(ctx context.Context, req *pb.DeleteBuildRequest) (*pb.DeleteBuildResponse, error) {
	err := h.service.DeleteBuild(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid build id")
		}
		if errors.Is(err, domain.ErrBuildNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "build with this id doesnt exists")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.DeleteBuildResponse{
		Success: true,
	}, nil
}
