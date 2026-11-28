package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"sparkly/analytics/internal/domain"
	pb "sparkly/analytics/proto/v1"
)

func TestDeviceDetailsDomainToPB(t *testing.T) {
	now := time.Date(2026, time.November, 27, 12, 30, 0, 0, time.UTC)
	deviceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	parkID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	buildID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	groupID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	bootID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	fileName := "src/main.c"
	reason := "hard_fault"

	details := domain.DeviceDetails{
		DeviceID: deviceID,
		Metrics: []domain.Metric{{
			MetricID:    "temperature",
			AvgValue:    21.5,
			MinValue:    20,
			MaxValue:    23,
			ChartPoints: []domain.MetricChartPoint{{Value: 21.5, Timestamp: now}},
		}},
		LogsCount: domain.LogsCount{
			Info:  1,
			Warn:  2,
			Error: 3,
			Fatal: 4,
			Trace: 5,
			Debug: 6,
			Total: 21,
		},
		Logs: []domain.Log{{
			ParkID:      parkID,
			DeviceID:    deviceID,
			BuildID:     &buildID,
			LogLevel:    domain.WARN,
			Message:     "battery low",
			Timestamp:   now,
			StackFrames: []domain.StackFrame{{File: &fileName, Line: 42, PC: 128}},
			Attributes:  map[string]string{"source": "agent"},
		}},
		Boots: []domain.Boot{{
			ID:              "boot-1",
			DeviceID:        deviceID,
			ReceivedAt:      now,
			BuildID:         &buildID,
			Diagnostics:     []string{"self-test ok"},
			ResetReason:     "power_on",
			FirmwareVersion: "1.4.2",
			AgentVersion:    "0.9.1",
		}},
		Crashes: []domain.Crash{{
			ParkID:      parkID,
			DeviceID:    deviceID,
			ReceivedAt:  now,
			BootID:      &bootID,
			BuildID:     &buildID,
			Diagnostics: []string{"fault"},
			StackFrames: []domain.StackFrame{{File: &fileName, Line: 7, PC: 99}},
			Reason:      &reason,
			Registers:   map[string]uint64{"pc": 4096, "lr": 8192},
		}},
		Timestamp: now,
		Incidents: []domain.Incident{{
			ID:       "incident-1",
			IssueID:  "issue-1",
			ParkID:   parkID,
			DeviceID: deviceID,
			BuildID:  buildID.String(),
			GroupIDs: []uuid.UUID{groupID},
			CauseLog: &domain.Log{
				ParkID:    parkID,
				DeviceID:  deviceID,
				LogLevel:  domain.ERROR,
				Message:   "panic",
				Timestamp: now,
			},
			CauseMetrics: []domain.Metric{{MetricID: "battery", AvgValue: 5, MinValue: 4, MaxValue: 6}},
			Timestamp:    now,
			StackFrames:  []domain.StackFrame{{File: &fileName, Line: 7, PC: 99}},
			Breadcrumbs: []domain.Log{{
				ParkID:    parkID,
				DeviceID:  deviceID,
				LogLevel:  domain.INFO,
				Message:   "before panic",
				Timestamp: now,
			}},
		}},
		MLInsights: &domain.MLInsights{},
	}

	buildIDStr := buildID.String()
	bootIDStr := bootID.String()
	want := &pb.AnalyzeDeviceResponse{
		DeviceId: deviceID.String(),
		Metrics: []*pb.Metric{{
			MetricId:    "temperature",
			AvgValue:    21.5,
			MinValue:    20,
			MaxValue:    23,
			ChartPoints: []*pb.Metric_MetricChartPoint{{Value: 21.5, Timestamp: timestamppb.New(now)}},
		}},
		LogsCount: &pb.LogsCount{
			Info:  1,
			Warn:  2,
			Error: 3,
			Fatal: 4,
			Trace: 5,
			Debug: 6,
			Total: 21,
		},
		Logs: []*pb.Log{{
			DeviceId:   deviceID.String(),
			ParkId:     parkID.String(),
			BuildId:    &buildIDStr,
			Level:      pb.LogLevel_LOG_LEVEL_WARN,
			Message:    "battery low",
			Timestamp:  timestamppb.New(now),
			Stack:      []*pb.StackFrame{{File: fileName, Line: 42, Pc: 128}},
			Attributes: map[string]string{"source": "agent"},
		}},
		Boots: []*pb.Boot{{
			Id:              "boot-1",
			DeviceId:        deviceID.String(),
			ReceivedAt:      timestamppb.New(now),
			BuildId:         &buildIDStr,
			Diagnostics:     []string{"self-test ok"},
			ResetReason:     "power_on",
			FirmwareVersion: "1.4.2",
			AgentVersion:    "0.9.1",
		}},
		Crashes: []*pb.Crash{{
			ParkId:      parkID.String(),
			DeviceId:    deviceID.String(),
			ReceivedAt:  timestamppb.New(now),
			BootId:      &bootIDStr,
			BuildId:     &buildIDStr,
			Diagnostics: []string{"fault"},
			Stack:       []*pb.StackFrame{{File: fileName, Line: 7, Pc: 99}},
			Reason:      &reason,
			Registers:   map[string]string{"pc": "4096", "lr": "8192"},
		}},
		Timestamp: timestamppb.New(now),
		Incidents: []*pb.Incident{{
			Id:       "incident-1",
			IssueId:  "issue-1",
			ParkId:   parkID.String(),
			DeviceId: deviceID.String(),
			BuildId:  buildID.String(),
			GroupIds: []string{groupID.String()},
			CauseLog: &pb.Log{
				DeviceId:  deviceID.String(),
				ParkId:    parkID.String(),
				Level:     pb.LogLevel_LOG_LEVEL_ERROR,
				Message:   "panic",
				Timestamp: timestamppb.New(now),
			},
			CauseMetrics: []*pb.Metric{{MetricId: "battery", AvgValue: 5, MinValue: 4, MaxValue: 6}},
			Timestamp:    timestamppb.New(now),
			Stack:        []*pb.StackFrame{{File: fileName, Line: 7, Pc: 99}},
			Breadcrumbs: []*pb.Log{{
				DeviceId:  deviceID.String(),
				ParkId:    parkID.String(),
				Level:     pb.LogLevel_LOG_LEVEL_INFO,
				Message:   "before panic",
				Timestamp: timestamppb.New(now),
			}},
		}},
		MlInsights: &pb.AnalyzeDeviceResponse_MLInsights{},
	}

	got := (&AnalyticsHandler{}).deviceDetailsDomainToPB(details)
	if !proto.Equal(got, want) {
		t.Fatalf("deviceDetailsDomainToPB() = %v, want %v", got, want)
	}
}

func TestDeviceDetailsDomainToPBOmitsAbsentOptionals(t *testing.T) {
	now := time.Date(2026, time.November, 27, 12, 30, 0, 0, time.UTC)
	deviceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	got := (&AnalyticsHandler{}).deviceDetailsDomainToPB(domain.DeviceDetails{
		DeviceID:  deviceID,
		Logs:      []domain.Log{{DeviceID: deviceID, LogLevel: domain.TRACE}},
		Boots:     []domain.Boot{{DeviceID: deviceID}},
		Timestamp: now,
	})

	if got.GetMlInsights() != nil {
		t.Fatalf("ml insights = %v, want nil", got.GetMlInsights())
	}
	if got.GetLogsCount() == nil {
		t.Fatal("logs count = nil, want zero value message")
	}
	if got.GetLogs()[0].BuildId != nil {
		t.Fatalf("log build id = %v, want nil", got.GetLogs()[0].BuildId)
	}
	if got.GetBoots()[0].BuildId != nil {
		t.Fatalf("boot build id = %v, want nil", got.GetBoots()[0].BuildId)
	}
	if !got.GetTimestamp().AsTime().Equal(now) {
		t.Fatalf("timestamp = %v, want %v", got.GetTimestamp().AsTime(), now)
	}
}

func TestLogLevelDomainToPB(t *testing.T) {
	tests := []struct {
		level domain.LogLevel
		want  pb.LogLevel
	}{
		{domain.UNSPECIFIED, pb.LogLevel_LOG_LEVEL_UNSPECIFIED},
		{domain.TRACE, pb.LogLevel_LOG_LEVEL_TRACE},
		{domain.DEBUG, pb.LogLevel_LOG_LEVEL_DEBUG},
		{domain.INFO, pb.LogLevel_LOG_LEVEL_INFO},
		{domain.WARN, pb.LogLevel_LOG_LEVEL_WARN},
		{domain.ERROR, pb.LogLevel_LOG_LEVEL_ERROR},
		{domain.FATAL, pb.LogLevel_LOG_LEVEL_FATAL},
		{domain.LogLevel(42), pb.LogLevel_LOG_LEVEL_UNSPECIFIED},
	}

	for _, tt := range tests {
		if got := logLevelDomainToPB(tt.level); got != tt.want {
			t.Errorf("logLevelDomainToPB(%d) = %v, want %v", tt.level, got, tt.want)
		}
	}
}
