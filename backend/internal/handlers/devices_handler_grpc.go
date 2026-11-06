package handlers

import (
	"context"
	
	"sparkly/backend/internal/services"

	pb "sparkly/backend/proto/v1"
	"sparkly/backend/proto/v1/backendv1connect"
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

func (h *DevicesHandler) GetDevice(context.Context, *pb.GetDeviceRequest) (*pb.GetDeviceResponse, error)