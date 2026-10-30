package handlers

import (
	"context"
	"errors"
	"log"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"
	pb "sparkly/backend/proto/v1"
	"sparkly/backend/proto/v1/backendv1connect"

	"connectrpc.com/connect/v2"
)

type ParkHandler struct {
	backendv1connect.UnimplementedParkServiceHandler
	service      *services.ParkService
	groupService *services.GroupsService
}

func NewParkHandler(s *services.ParkService, gs *services.GroupsService) *ParkHandler {
	return &ParkHandler{
		service:      s,
		groupService: gs,
	}
}

func (h *ParkHandler) CreatePark(ctx context.Context, r *pb.CreateParkRequest) (*pb.CreateParkResponse, error) {
	name, desc := r.GetName(), r.GetDescription()
	park, parkToken, err := h.service.CreatePark(ctx, name, desc)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		log.Printf("ParkService/CreatePark error: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.CreateParkResponse{
		Park: &pb.Park{
			Id:          park.ID.String(),
			Name:        park.Name,
			Description: park.Desc,
		},
		ParkToken: parkToken,
	}, nil
}

func (h *ParkHandler) GetPark(ctx context.Context, r *pb.GetParkRequest) (*pb.GetParkResponse, error) {
	id := r.GetId()

	park, err := h.service.GetParkByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrParkNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "park with this id doesnt exists")
		}
		log.Printf("ParkService/GetPark error: %v\n", err)
		return nil, connect.NewError(connect.CodeNotFound, "internal error")
	}

	return &pb.GetParkResponse{
		Park: &pb.Park{
			Id:          park.ID.String(),
			Name:        park.Name,
			Description: park.Desc,
		},
	}, nil
}

func (h *ParkHandler) GetAllParks(ctx context.Context, r *pb.GetAllParksRequest) (*pb.GetAllParksResponse, error) {
	parks, err := h.service.GetAllParks(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		log.Printf("ParkService/GetAllParks error: %v\n", err)
		return nil, connect.NewError(connect.CodeNotFound, "internal error")
	}

	parksResponse := make([]*pb.Park, len(parks))
	for i, domainPark := range parks {
		parksResponse[i] = &pb.Park{
			Id:          domainPark.ID.String(),
			Name:        domainPark.Name,
			Description: domainPark.Desc,
		}
	}
	return &pb.GetAllParksResponse{
		Parks: parksResponse,
	}, nil
}

func (h *ParkHandler) UpdatePark(ctx context.Context, r *pb.UpdateParkRequest) (*pb.UpdateParkResponse, error) {
	id := r.GetId()
	newName := r.GetName()
	newDesc := r.GetDescription()

	park, err := h.service.UpdatePark(ctx, id, domain.UpdateParkParams{
		Name:        newName,
		Description: newDesc,
	})

	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrParkNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "park with this id doesnt exists")
		}
		log.Printf("ParkService/UpdatePark error: %v\n", err)
		return nil, connect.NewError(connect.CodeNotFound, "internal error")
	}

	return &pb.UpdateParkResponse{
		Park: &pb.Park{
			Id:          park.ID.String(),
			Name:        park.Name,
			Description: park.Desc,
		},
	}, nil
}

func (h *ParkHandler) DeletePark(ctx context.Context, r *pb.DeleteParkRequest) (*pb.DeleteParkResponse, error) {
	id := r.GetId()

	err := h.service.DeletePark(ctx, id)

	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrParkNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "park with this id doesnt exists")
		}
		log.Printf("ParkService/DeletePark error: %v\n", err)
		return nil, connect.NewError(connect.CodeNotFound, "internal error")
	}

	return &pb.DeleteParkResponse{
		Success: true,
	}, nil
}

func (h *ParkHandler) CreateGroup(ctx context.Context, r *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	parkID, name, desc := r.GetParkId(), r.GetName(), r.GetDescription()

	group, err := h.groupService.CreateGroup(ctx, parkID, name, desc)
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
		log.Printf("ParkService/CreateGroup error: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.CreateGroupResponse{
		Group: &pb.DeviceGroup{
			Id:          group.ID.String(),
			ParkId:      group.ParkID.String(),
			Name:        group.Name,
			Description: group.Description,
		},
	}, nil
}

func (h *ParkHandler) GetGroup(ctx context.Context, r *pb.GetGroupRequest) (*pb.GetGroupResponse, error) {
	id := r.GetId()

	group, err := h.groupService.GetGroup(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid group id")
		}
		if errors.Is(err, domain.ErrGroupNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "group with this id doesnt exists")
		}
		log.Printf("ParkService/GetGroup error: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.GetGroupResponse{
		Group: &pb.DeviceGroup{
			Id:          group.ID.String(),
			ParkId:      group.ParkID.String(),
			Name:        group.Name,
			Description: group.Description,
		},
	}, nil
}

func (h *ParkHandler) GetGroups(ctx context.Context, r *pb.GetGroupsRequest) (*pb.GetGroupsResponse, error) {
	parkID := r.GetParkId()

	groups, err := h.groupService.GetAllGroups(ctx, parkID)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid park id")
		}
		log.Printf("ParkService/GetGroups error: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	pbGroups := make([]*pb.DeviceGroup, len(groups))
	for i, g := range groups {
		pbGroups[i] = &pb.DeviceGroup{
			Id:          g.ID.String(),
			ParkId:      g.ParkID.String(),
			Name:        g.Name,
			Description: g.Description,
		}
	}

	return &pb.GetGroupsResponse{
		Groups: pbGroups,
	}, nil
}

func (h *ParkHandler) UpdateGroup(ctx context.Context, r *pb.UpdateGroupRequest) (*pb.UpdateGroupResponse, error) {
	id := r.GetId()
	newName := r.GetName()
	newDesc := r.GetDescription()

	group, err := h.groupService.UpdateGroup(ctx, id, newName, newDesc)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid group id")
		}
		if errors.Is(err, domain.ErrGroupNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "group with this id doesnt exists")
		}
		log.Printf("ParkService/UpdateGroup error: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.UpdateGroupResponse{
		Group: &pb.DeviceGroup{
			Id:          group.ID.String(),
			ParkId:      group.ParkID.String(),
			Name:        group.Name,
			Description: group.Description,
		},
	}, nil
}

func (h *ParkHandler) DeleteGroup(ctx context.Context, r *pb.DeleteGroupRequest) (*pb.DeleteGroupResponse, error) {
	id := r.GetId()

	err := h.groupService.DeleteGroup(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownUser) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated user")
		}
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid group id")
		}
		if errors.Is(err, domain.ErrGroupNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "group with this id doesnt exists")
		}
		log.Printf("ParkService/DeleteGroup error: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.DeleteGroupResponse{
		Success: true,
	}, nil
}
