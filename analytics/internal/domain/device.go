package domain

import (
	"time"

	"github.com/google/uuid"
)

type LogsCount struct {
	Info  int64
	Warn  int64
	Error int64
	Fatal int64
	Trace int64
	Debug int64
	Total int64
}

type Incident struct {
	ID           string
	IssueID      string
	ParkID       uuid.UUID
	DeviceID     uuid.UUID
	BuildID      string
	GroupIDs     []uuid.UUID
	CauseLog     *Log
	CauseMetrics []Metric
	Timestamp    time.Time
	StackFrames  []StackFrame
	Breadcrumbs  []Log
}

type MLInsights struct{}

type DeviceSummary struct {
	DeviceID   uuid.UUID
	LastSeenAt time.Time
}

type DevicesSummary struct {
	Devices   []DeviceSummary
	Timestamp time.Time
}

type DeviceDetails struct {
	DeviceID   uuid.UUID
	Metrics    []Metric
	LogsCount  LogsCount
	Logs       []Log
	Boots      []Boot
	Timestamp  time.Time
	Incidents  []Incident
	MLInsights *MLInsights
}

type GetDeviceDetailsFilters struct {
	FromTime *time.Time
	ToTime   *time.Time
}

type GetDeviceSummaryFilters struct {
	GroupIDs *[]uuid.UUID
}
