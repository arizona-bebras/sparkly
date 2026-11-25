package handlers

import (
	"context"
	
	"sparkly/analytics/internal/services"
	pb "sparkly/analytics/proto/v1"
	analyticsv1 "sparkly/analytics/proto/v1/analyticsv1connect"
	"connectrpc.com/connect/v2"
)

type AnalyticsHandler struct {
	analyticsv1.UnimplementedAnalyticsServiceHandler
	service *services.AnalyzeDeviceService
}

func NewAnalyzeDevicesHandler(service *services.AnalyzeDeviceService) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
	}
}

func (h *AnalyticsHandler) AnalyzeDevice(ctx context.Context, req *pb.AnalyzeDeviceRequest) (*pb.AnalyzeDeviceResponse, error) {
	deviceID := req.GetDeviceId()
	timerange := req.GetTimerange()

	var filters *services.GetDeviceDetailsFilters
	if timerange != nil {
		filters = &services.GetDeviceDetailsFilters{
			FromTime: timerange.FromTime.AsTime(),
			ToTime: timerange.ToTime.AsTime(),
		}
	}

	deviceAnalysis, err := h.service.GetDeviceDetails(ctx, deviceID, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}
}