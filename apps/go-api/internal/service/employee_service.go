package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/faceattendance/go-api/internal/repository"
	"github.com/faceattendance/go-api/internal/utils"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CreateEmployeeRequest struct {
	TenantID     uuid.UUID
	OfficeID     *uuid.UUID
	Name         string
	Email        string
	Password     string
	EmployeeCode string
}

type EmployeeService interface {
	CreateEmployee(ctx context.Context, req CreateEmployeeRequest) (*model.Employee, error)
	DeleteEmployee(ctx context.Context, tenantID, callerEmployeeID, targetEmployeeID uuid.UUID) error
	EnrollFace(ctx context.Context, tenantID, employeeID uuid.UUID, files []UploadedFile) ([]float32, error)
	GetEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (*model.Employee, error)
}

type employeeService struct {
	employeeRepo repository.EmployeeRepository
	aiClient     AIEngineClient
}

func NewEmployeeService(employeeRepo repository.EmployeeRepository, aiClient AIEngineClient) EmployeeService {
	return &employeeService{
		employeeRepo: employeeRepo,
		aiClient:     aiClient,
	}
}

func (s *employeeService) CreateEmployee(ctx context.Context, req CreateEmployeeRequest) (*model.Employee, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("employee name is required")
	}
	if strings.TrimSpace(req.Email) == "" {
		return nil, errors.New("employee email is required")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}
	if strings.TrimSpace(req.EmployeeCode) == "" {
		return nil, errors.New("employee code is required")
	}

	// Check if email already exists
	existingEmp, _, err := s.employeeRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}
	if existingEmp != nil {
		return nil, errors.New("email is already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	emp := &model.Employee{
		ID:           uuid.New(),
		TenantID:     req.TenantID,
		OfficeID:     req.OfficeID,
		Name:         strings.TrimSpace(req.Name),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hashedPassword),
		EmployeeCode: strings.TrimSpace(req.EmployeeCode),
		IsActive:     true,
	}

	if err := s.employeeRepo.Create(ctx, emp); err != nil {
		return nil, fmt.Errorf("failed to create employee: %w", err)
	}

	return emp, nil
}

func (s *employeeService) EnrollFace(ctx context.Context, tenantID, employeeID uuid.UUID, files []UploadedFile) ([]float32, error) {
	if len(files) == 0 {
		return nil, errors.New("at least one face photo is required for enrollment")
	}

	// Verify employee exists
	emp, err := s.employeeRepo.GetByID(ctx, employeeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check employee: %w", err)
	}
	if emp == nil {
		return nil, errors.New("employee not found or tenant mismatch")
	}

	var embedding []float32
	if len(files) == 1 {
		// Single image extraction
		emb, err := s.aiClient.ExtractFace(ctx, files[0].Filename, files[0].Data)
		if err != nil {
			return nil, fmt.Errorf("face extraction failed: %w", err)
		}
		embedding = emb
	} else {
		// Multi-pose merge extraction (e.g. 3-5 images as recommended in SPEC.md)
		emb, err := s.aiClient.EnrollMerge(ctx, files)
		if err != nil {
			return nil, fmt.Errorf("face enrollment merge failed: %w", err)
		}
		embedding = emb
	}

	if len(embedding) != 512 {
		return nil, fmt.Errorf("invalid embedding dimension: expected 512, got %d", len(embedding))
	}

	// Convert float32 slice to pgvector string format "[0.012,-0.045,...]"
	vecString := utils.VectorToString(embedding)

	// Persist to PostgreSQL employees table
	if err := s.employeeRepo.UpdateFaceEmbedding(ctx, employeeID, tenantID, vecString); err != nil {
		return nil, fmt.Errorf("failed to save face embedding to database: %w", err)
	}

	return embedding, nil
}

func (s *employeeService) GetEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (*model.Employee, error) {
	return s.employeeRepo.GetByID(ctx, employeeID, tenantID)
}

func (s *employeeService) DeleteEmployee(ctx context.Context, tenantID, callerEmployeeID, targetEmployeeID uuid.UUID) error {
	if callerEmployeeID == targetEmployeeID {
		return errors.New("cannot delete your own employee account")
	}

	emp, err := s.employeeRepo.GetByID(ctx, targetEmployeeID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to verify employee: %w", err)
	}
	if emp == nil {
		return errors.New("employee not found or tenant mismatch")
	}

	if err := s.employeeRepo.Delete(ctx, targetEmployeeID, tenantID); err != nil {
		return fmt.Errorf("failed to delete employee: %w", err)
	}

	return nil
}

