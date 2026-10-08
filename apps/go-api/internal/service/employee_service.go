package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/faceattendance/go-api/internal/repository"
	"github.com/faceattendance/go-api/internal/utils"
	"github.com/google/uuid"
)

type EmployeeService interface {
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
