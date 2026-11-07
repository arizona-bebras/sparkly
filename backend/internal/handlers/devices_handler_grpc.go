package handlers

import (
	"context"
	"errors"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"

	pb "sparkly/backend/proto/v1"
	"sparkly/backend/proto/v1/backendv1connect"

	"connectrpc.com/connect/v2"
)

type DevicesHandler struct {
	backendv1connect.UnimplementedDeviceServiceHandler
	service *services.DeviceService
}

func NewDevicesHandler(s *services.DeviceService) *DevicesHandler {
	return &DevicesHandler{
		service: s,
	}
}

func deviceDomainToResp(device domain.Device) *pb.Device {
	pbGroups := make([]*pb.DeviceGroup, 0, len(device.Groups))
	for _, g := range device.Groups {
		pbGroups = append(pbGroups, &pb.DeviceGroup{
			Id:          g.ID.String(),
			ParkId:      g.ParkID.String(),
			Name:        g.Name,
			Description: g.Description,
		})
	}

	return &pb.Device{
		Id:     device.ID.String(),
		Name:   device.Name,
		ParkId: device.ParkID.String(),
		Groups: pbGroups,
	}
}

func (h *DevicesHandler) GetDevice(ctx context.Context, req *pb.GetDeviceRequest) (*pb.GetDeviceResponse, error) {
	id := req.GetId()

	device, err := h.service.GetDevice(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "device not found")
		}
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.GetDeviceResponse{
		Device: deviceDomainToResp(device),
	}, nil
}

func (h *DevicesHandler) GetDevices(ctx context.Context, req *pb.GetDevicesRequest) (*pb.GetDevicesResponse, error) {
	parkID := req.GetParkId()
	filters := services.DeviceFiltersDTO{
		ParkID:     parkID,
		DevicesIDs: req.DeviceIds,
		Name:       req.Name,
		GroupIDs:   req.GroupIds,
		BuildIDs:   req.BuildIds,
	}

	devices, err := h.service.GetDevices(ctx, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	respDevices := make([]*pb.Device, 0, len(devices))
	for _, device := range devices {
		respDevices = append(respDevices, deviceDomainToResp(device))
	}

	return &pb.GetDevicesResponse{
		Devices: respDevices,
	}, nil
}
