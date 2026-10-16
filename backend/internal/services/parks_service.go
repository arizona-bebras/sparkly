package services

import (
	"sparkly/backend/internal/domain"

	"github.com/google/uuid"
)

type ParkRepo interface {
	CreatePark(name string, desc string) (domain.Park, error)
	GetParkByID(id uuid.UUID) (domain.Park, error)
	GetAllParks(id uuid.UUID) (domain.Park, error)
	UpdatePark(id uuid.UUID, name string, desc string) (domain.Park, error)
	DeletePark(id uuid.UUID) error
}

type ParkService struct {
	repo ParkRepo
}