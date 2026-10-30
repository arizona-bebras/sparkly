package models

import (
	"sparkly/backend/internal/domain"
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
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Email         string         `gorm:"uniqueIndex;not null"`
	Password      string         `gorm:"not null"`
	Parks         []Park         `gorm:"many2many:user_parks;constraint:OnDelete:CASCADE;"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) ToDomain() domain.User {
	return domain.User{
		ID:       u.ID,
		Email:    u.Email,
		Password: u.Password,
	}
}

func FromDomain(u *domain.User) User {
	return User{
		ID:       u.ID,
		Email:    u.Email,
		Password: u.Password,
	}
}

type Park struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"not null"`
	Description string
	TokenHash   string    `gorm:"uniqueIndex;not null"`
	Users       []User    `gorm:"many2many:user_parks"`
	Groups      []Group   `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Devices     []Device  `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Issues      []Issue   `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Builds      []Build   `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Releases    []Release `gorm:"foreignKey:ParkID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (Park) TableName() string {
	return "parks"
}

func (p Park) ToDomain() domain.Park {
	return domain.Park{
		ID:   p.ID,
		Name: p.Name,
		Desc: p.Description,
	}
}

func (_ Park) FromDomain(p domain.Park) Park {
	return Park{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Desc,
	}
}

type Group struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"not null"`
	Description string
	ParkID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Devices     []Device  `gorm:"many2many:group_devices;constraint:OnDelete:CASCADE;"`
}

func (Group) TableName() string {
	return "groups"
}

func (g Group) ToDomain() domain.Group {
	return domain.Group{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		ParkID:      g.ParkID,
	}
}

func (_ Group) FromDomain(g domain.Group) Group {
	return Group{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		ParkID:      g.ParkID,
	}
}

type Device struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"not null"`
	ParkID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"not null"`
	JoinedAt  time.Time
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

type KeyStatus string

const (
	KeyStatusActive  KeyStatus = "active"
	KeyStatusRotated KeyStatus = "rotated"
	KeyStatusRevoked KeyStatus = "revoked"
)

type SigningKey struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	KID          string    `gorm:"type:varchar(64);uniqueIndex;not null"`
	EncryptedKey []byte    `gorm:"type:bytea;not null"`
	Algorithm    string    `gorm:"type:varchar(16);not null"`
	Status       KeyStatus `gorm:"type:varchar(7);not null'default:active"`
	CreatedAt    time.Time `gorm:"not null"`
	ExpiresAt    time.Time `gorm:"not null"`
}

func (SigningKey) TableName() string {
	return "signing_keys"
}

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	User      User      `gorm:"foreignKey:UserID;references:ID"`
	Revoked   bool
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (t *RefreshToken) ToDomain() domain.RefreshToken {
	return domain.RefreshToken{
		ID:        t.ID,
		Revoked:   t.Revoked,
		ExpiresAt: t.ExpiresAt,
	}
}
