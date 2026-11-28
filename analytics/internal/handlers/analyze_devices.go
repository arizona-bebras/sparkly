package handlers

import (
	"context"
	"log"
	"strconv"

	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/services"
	pb "sparkly/analytics/proto/v1"
	analyticsv1 "sparkly/analytics/proto/v1/analyticsv1connect"

	"connectrpc.com/connect/v2"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (h *AnalyticsHandler) deviceDetailsDomainToPB(details domain.DeviceDetails) *pb.AnalyzeDeviceResponse {
	metrics := make([]*pb.Metric, 0, len(details.Metrics))
	for _, metric := range details.Metrics {
		metrics = append(metrics, metricDomainToPB(metric))
	}

	logs := make([]*pb.Log, 0, len(details.Logs))
	for _, entry := range details.Logs {
		logs = append(logs, logDomainToPB(entry))
	}

	boots := make([]*pb.Boot, 0, len(details.Boots))
	for _, boot := range details.Boots {
		boots = append(boots, bootDomainToPB(boot))
	}

	crashes := make([]*pb.Crash, 0, len(details.Crashes))
	for _, crash := range details.Crashes {
		crashes = append(crashes, crashDomainToPB(crash))
	}

	incidents := make([]*pb.Incident, 0, len(details.Incidents))
	for _, incident := range details.Incidents {
		incidents = append(incidents, incidentDomainToPB(incident))
	}

	return &pb.AnalyzeDeviceResponse{
		DeviceId:   details.DeviceID.String(),
		Metrics:    metrics,
		LogsCount:  logsCountDomainToPB(details.LogsCount),
		Logs:       logs,
		Boots:      boots,
		Crashes:    crashes,
		Timestamp:  timestamppb.New(details.Timestamp),
		Incidents:  incidents,
		MlInsights: mlInsightsDomainToPB(details.MLInsights),
	}
}

func metricDomainToPB(metric domain.Metric) *pb.Metric {
	chartPoints := make([]*pb.Metric_MetricChartPoint, 0, len(metric.ChartPoints))
	for _, point := range metric.ChartPoints {
		chartPoints = append(chartPoints, &pb.Metric_MetricChartPoint{
			Value:     point.Value,
			Timestamp: timestamppb.New(point.Timestamp),
		})
	}

	return &pb.Metric{
		MetricId:    metric.MetricID,
		AvgValue:    metric.AvgValue,
		MinValue:    metric.MinValue,
		MaxValue:    metric.MaxValue,
		ChartPoints: chartPoints,
	}
}

func logsCountDomainToPB(count domain.LogsCount) *pb.LogsCount {
	return &pb.LogsCount{
		Info:  count.Info,
		Warn:  count.Warn,
		Error: count.Error,
		Fatal: count.Fatal,
		Trace: count.Trace,
		Debug: count.Debug,
		Total: count.Total,
	}
}

func logDomainToPB(entry domain.Log) *pb.Log {
	stack := make([]*pb.StackFrame, 0, len(entry.StackFrames))
	for _, frame := range entry.StackFrames {
		stack = append(stack, stackFrameDomainToPB(frame))
	}

	return &pb.Log{
		DeviceId:   entry.DeviceID.String(),
		ParkId:     entry.ParkID.String(),
		BuildId:    optionalUUIDToString(entry.BuildID),
		Level:      logLevelDomainToPB(entry.LogLevel),
		Message:    entry.Message,
		Timestamp:  timestamppb.New(entry.Timestamp),
		Stack:      stack,
		Attributes: entry.Attributes,
	}
}

func stackFrameDomainToPB(frame domain.StackFrame) *pb.StackFrame {
	var file string
	if frame.File != nil {
		file = *frame.File
	}

	return &pb.StackFrame{
		File: file,
		Line: int32(frame.Line),
		Pc:   int64(frame.PC),
	}
}

func logLevelDomainToPB(level domain.LogLevel) pb.LogLevel {
	switch level {
	case domain.TRACE:
		return pb.LogLevel_LOG_LEVEL_TRACE
	case domain.DEBUG:
		return pb.LogLevel_LOG_LEVEL_DEBUG
	case domain.INFO:
		return pb.LogLevel_LOG_LEVEL_INFO
	case domain.WARN:
		return pb.LogLevel_LOG_LEVEL_WARN
	case domain.ERROR:
		return pb.LogLevel_LOG_LEVEL_ERROR
	case domain.FATAL:
		return pb.LogLevel_LOG_LEVEL_FATAL
	}

	return pb.LogLevel_LOG_LEVEL_UNSPECIFIED
}

func bootDomainToPB(boot domain.Boot) *pb.Boot {
	return &pb.Boot{
		Id:              boot.ID,
		DeviceId:        boot.DeviceID.String(),
		ReceivedAt:      timestamppb.New(boot.ReceivedAt),
		BuildId:         optionalUUIDToString(boot.BuildID),
		Diagnostics:     boot.Diagnostics,
		ResetReason:     boot.ResetReason,
		FirmwareVersion: boot.FirmwareVersion,
		AgentVersion:    boot.AgentVersion,
	}
}

func crashDomainToPB(crash domain.Crash) *pb.Crash {
	stack := make([]*pb.StackFrame, 0, len(crash.StackFrames))
	for _, frame := range crash.StackFrames {
		stack = append(stack, stackFrameDomainToPB(frame))
	}

	registers := make(map[string]string, len(crash.Registers))
	for name, value := range crash.Registers {
		registers[name] = strconv.FormatUint(value, 10)
	}

	return &pb.Crash{
		ParkId:      crash.ParkID.String(),
		DeviceId:    crash.DeviceID.String(),
		ReceivedAt:  timestamppb.New(crash.ReceivedAt),
		BootId:      optionalUUIDToString(crash.BootID),
		BuildId:     optionalUUIDToString(crash.BuildID),
		Diagnostics: crash.Diagnostics,
		Stack:       stack,
		Reason:      crash.Reason,
		Registers:   registers,
	}
}

func incidentDomainToPB(incident domain.Incident) *pb.Incident {
	var causeLog *pb.Log
	if incident.CauseLog != nil {
		causeLog = logDomainToPB(*incident.CauseLog)
	}

	causeMetrics := make([]*pb.Metric, 0, len(incident.CauseMetrics))
	for _, metric := range incident.CauseMetrics {
		causeMetrics = append(causeMetrics, metricDomainToPB(metric))
	}

	stack := make([]*pb.StackFrame, 0, len(incident.StackFrames))
	for _, frame := range incident.StackFrames {
		stack = append(stack, stackFrameDomainToPB(frame))
	}

	breadcrumbs := make([]*pb.Log, 0, len(incident.Breadcrumbs))
	for _, entry := range incident.Breadcrumbs {
		breadcrumbs = append(breadcrumbs, logDomainToPB(entry))
	}

	groupIDs := make([]string, 0, len(incident.GroupIDs))
	for _, groupID := range incident.GroupIDs {
		groupIDs = append(groupIDs, groupID.String())
	}

	return &pb.Incident{
		Id:           incident.ID,
		IssueId:      incident.IssueID,
		ParkId:       incident.ParkID.String(),
		DeviceId:     incident.DeviceID.String(),
		BuildId:      incident.BuildID,
		GroupIds:     groupIDs,
		CauseLog:     causeLog,
		CauseMetrics: causeMetrics,
		Timestamp:    timestamppb.New(incident.Timestamp),
		Stack:        stack,
		Breadcrumbs:  breadcrumbs,
	}
}

func mlInsightsDomainToPB(insights *domain.MLInsights) *pb.AnalyzeDeviceResponse_MLInsights {
	if insights == nil {
		return nil
	}

	return &pb.AnalyzeDeviceResponse_MLInsights{}
}

func optionalUUIDToString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}

	value := id.String()
	return &value
}

func (h *AnalyticsHandler) AnalyzeDevice(ctx context.Context, req *pb.AnalyzeDeviceRequest) (*pb.AnalyzeDeviceResponse, error) {
	deviceID := req.GetDeviceId()
	timerange := req.GetTimerange()

	var filters *services.GetDeviceDetailsFilters
	if timerange != nil {
		filters = &services.GetDeviceDetailsFilters{
			FromTime: timerange.FromTime.AsTime(),
			ToTime:   timerange.ToTime.AsTime(),
		}
	}

	deviceAnalysis, err := h.service.GetDeviceDetails(ctx, deviceID, filters)
	if err != nil {
		log.Printf("AnalyticsService/AnalyzeDevice error: %v", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return h.deviceDetailsDomainToPB(deviceAnalysis), nil
}

func deviceSummaryDomainToPB(sum domain.DeviceSummary) *pb.GetDevicesSummaryResponse_DeviceSummary {
	return &pb.GetDevicesSummaryResponse_DeviceSummary{
		DeviceId:   sum.DeviceID.String(),
		LastSeenAt: timestamppb.New(sum.LastSeenAt),
	}
}

func devicesSummaryDomainToPB(sum domain.DevicesSummary) *pb.GetDevicesSummaryResponse {
	pbSummary := make([]*pb.GetDevicesSummaryResponse_DeviceSummary, 0, len(sum.Devices))
	for _, singleSum := range sum.Devices {
		pbSummary = append(pbSummary, deviceSummaryDomainToPB(singleSum))
	}

	return &pb.GetDevicesSummaryResponse{
		Devices:   pbSummary,
		Timestamp: timestamppb.New(sum.Timestamp),
	}
}

func (h *AnalyticsHandler) GetDevicesSummary(ctx context.Context, req *pb.GetDevicesSummaryRequest) (*pb.GetDevicesSummaryResponse, error) {
	parkID := req.GetParkId()

	filters := services.GetDeviceSummaryFilters{
		GroupIDsStr: &req.GroupIds,
	}

	summary, err := h.service.GetDeviceSummary(ctx, parkID, filters)
	if err != nil {
		log.Printf("AnalyticsService/GetDevicesSummary error: %v", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return devicesSummaryDomainToPB(summary), nil
}
