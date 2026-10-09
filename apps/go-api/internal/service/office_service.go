package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/faceattendance/go-api/internal/repository"
	"github.com/google/uuid"
)

type CreateOfficeRequest struct {
	Name         string  `json:"name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	RadiusMeters int     `json:"radius_meters"`
}

type UpdateOfficeRequest struct {
	Name         string   `json:"name"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	RadiusMeters *int     `json:"radius_meters"`
}

type OfficeService interface {
	ListOffices(ctx context.Context, tenantID uuid.UUID) ([]*model.Office, error)
	GetOffice(ctx context.Context, id uuid.UUID) (*model.Office, error)
	CreateOffice(ctx context.Context, tenantID uuid.UUID, req CreateOfficeRequest) (*model.Office, error)
	UpdateOffice(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdateOfficeRequest) (*model.Office, error)
	DeleteOffice(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type officeService struct {
	officeRepo repository.OfficeRepository
}

func NewOfficeService(officeRepo repository.OfficeRepository) OfficeService {
	return &officeService{officeRepo: officeRepo}
}

func (s *officeService) ListOffices(ctx context.Context, tenantID uuid.UUID) ([]*model.Office, error) {
	return s.officeRepo.GetByTenantID(ctx, tenantID)
}

func (s *officeService) GetOffice(ctx context.Context, id uuid.UUID) (*model.Office, error) {
	office, err := s.officeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if office == nil {
		return nil, errors.New("office not found")
	}
	return office, nil
}

func (s *officeService) CreateOffice(ctx context.Context, tenantID uuid.UUID, req CreateOfficeRequest) (*model.Office, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("office name is required")
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		return nil, errors.New("invalid latitude: must be between -90 and 90")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return nil, errors.New("invalid longitude: must be between -180 and 180")
	}
	radius := req.RadiusMeters
	if radius <= 0 {
		radius = 50
	}

	office := &model.Office{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         name,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		RadiusMeters: radius,
		CreatedAt:    time.Now(),
	}

	if err := s.officeRepo.Create(ctx, office); err != nil {
		return nil, fmt.Errorf("failed to create office: %w", err)
	}

	return office, nil
}

func (s *officeService) UpdateOffice(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdateOfficeRequest) (*model.Office, error) {
	office, err := s.officeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if office == nil || office.TenantID != tenantID {
		return nil, errors.New("office not found or tenant mismatch")
	}

	name := strings.TrimSpace(req.Name)
	if name != "" {
		office.Name = name
	}
	if req.Latitude != nil {
		office.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		office.Longitude = *req.Longitude
	}
	if req.RadiusMeters != nil && *req.RadiusMeters > 0 {
		office.RadiusMeters = *req.RadiusMeters
	}

	if err := s.officeRepo.Update(ctx, office); err != nil {
		return nil, fmt.Errorf("failed to update office: %w", err)
	}

	return office, nil
}

func (s *officeService) DeleteOffice(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	return s.officeRepo.Delete(ctx, id, tenantID)
}
