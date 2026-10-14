package models

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

type User struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email    string    `gorm:"not null"`
	Password string    `gorm:"not null"`
	Parks    []Park    `gorm:"many2many:user_parks"`
}

func (User) TableName() string {
	return "users"
}

type Park struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"not null"`
	Description string
	Groups      []Group   `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Devices     []Device  `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Issues      []Issue   `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Builds      []Build   `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Releases    []Release `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (Park) TableName() string {
	return "parks"
}

type Group struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name    string    `gorm:"not null"`
	ParkID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Devices []Device  `gorm:"many2many:group_devices"`
}

func (Group) TableName() string {
	return "groups"
}

type Device struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name     string    `gorm:"not null"`
	ParkID   uuid.UUID `gorm:"type:uuid;not null;index"`
	JoinedAt time.Time
}

func (Device) TableName() string {
	return "devices"
}

type Issue struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	ParkID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Title      string    `gorm:"not null"`
	Desc       string
	Severity   IssueSeverity `gorm:"type:issue_severity;not null;default:'info'"`
	ResolvedAt time.Time
	FirstAt    time.Time
	LastAt     time.Time
}

func (Issue) TableName() string {
	return "issues"
}

type Build struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	ParkID         uuid.UUID `gorm:"type:uuid;not null;index"`
	ReleaseID      uuid.UUID
	Commit         string
	ElfObjectKey   string
	DwarfObjectKey string
}

func (Build) TableName() string {
	return "builds"
}

type Release struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	ParkID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Version string
}

func (Release) TableName() string {
	return "releases"
}
