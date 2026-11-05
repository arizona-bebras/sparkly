package domain

import (
	"time"

	"github.com/google/uuid"
)

type IssueSeverity string

const (
	IssueSeverityInfo     IssueSeverity = "info"
	IssueSeverityLow      IssueSeverity = "low"
	IssueSeverityMedium   IssueSeverity = "medium"
	IssueSeverityHigh     IssueSeverity = "high"
	IssueSeverityCritical IssueSeverity = "critical"
)

type Issue struct {
	ID          uuid.UUID
	ParkID      uuid.UUID
	Title       string
	Description string
	Severity    IssueSeverity
	IsResolved  bool
	ResolvedAt  time.Time
	FirstAt     time.Time
	LastAt      time.Time
}

type UpdateIssueParams struct {
	Title       *string
	Description *string
	Severity    *IssueSeverity
}

type IssueFilters struct {
	Title      *string
	Severity   *IssueSeverity
	IsResolved *bool
	FromTime   *time.Time
	ToTime     *time.Time
}
