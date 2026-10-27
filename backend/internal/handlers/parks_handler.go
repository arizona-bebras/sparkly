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
	service *services.ParkService
}

func NewParkHandler(s *services.ParkService) *ParkHandler {
	return &ParkHandler{
		service: s,
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
