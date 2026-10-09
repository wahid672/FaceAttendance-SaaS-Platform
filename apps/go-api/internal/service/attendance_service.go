package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/faceattendance/go-api/internal/config"
	"github.com/faceattendance/go-api/internal/model"
	"github.com/faceattendance/go-api/internal/repository"
	"github.com/faceattendance/go-api/internal/utils"
	"github.com/google/uuid"
)

type CheckInRequest struct {
	TenantID       uuid.UUID
	EmployeeID     uuid.UUID
	ImageFilename  string
	ImageData      []byte
	Latitude       float64
	Longitude      float64
	DeviceID       string
	AttendanceType string // 'IN' or 'OUT'
	PhotoURL       *string
}

type CheckInResult struct {
	AttendanceLog   *model.AttendanceLog `json:"attendance_log"`
	SimilarityScore float64              `json:"similarity_score"`
	DistanceMeters  float64              `json:"distance_meters"`
	AllowedRadius   int                  `json:"allowed_radius"`
	IsValid         bool                 `json:"is_valid"`
	ValidationNotes []string             `json:"validation_notes,omitempty"`
}

type AttendanceLogsResult struct {
	Total int                    `json:"total"`
	Page  int                    `json:"page"`
	Limit int                    `json:"limit"`
	Logs  []*model.AttendanceLog `json:"logs"`
}

type AttendanceService interface {
	CheckIn(ctx context.Context, req CheckInRequest) (*CheckInResult, error)
	GetAttendanceHistory(ctx context.Context, employeeID uuid.UUID, limit int) ([]*model.AttendanceLog, error)
	ListLogs(ctx context.Context, tenantID uuid.UUID, page, limit int, startDate, endDate *time.Time, userID *uuid.UUID, isValid *bool) (*AttendanceLogsResult, error)
	GetSummary(ctx context.Context, tenantID uuid.UUID, targetDate time.Time) (*repository.AttendanceSummary, error)
}

type attendanceService struct {
	cfg            *config.Config
	employeeRepo   repository.EmployeeRepository
	officeRepo     repository.OfficeRepository
	attendanceRepo repository.AttendanceRepository
	aiClient       AIEngineClient
}

func NewAttendanceService(
	cfg *config.Config,
	employeeRepo repository.EmployeeRepository,
	officeRepo repository.OfficeRepository,
	attendanceRepo repository.AttendanceRepository,
	aiClient AIEngineClient,
) AttendanceService {
	return &attendanceService{
		cfg:            cfg,
		employeeRepo:   employeeRepo,
		officeRepo:     officeRepo,
		attendanceRepo: attendanceRepo,
		aiClient:       aiClient,
	}
}

func (s *attendanceService) CheckIn(ctx context.Context, req CheckInRequest) (*CheckInResult, error) {
	if len(req.ImageData) == 0 {
		return nil, errors.New("selfie photo image is required")
	}

	if req.AttendanceType == "" {
		req.AttendanceType = "IN"
	}

	// 1. Fetch and validate employee
	emp, err := s.employeeRepo.GetByID(ctx, req.EmployeeID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch employee: %w", err)
	}
	if emp == nil {
		return nil, errors.New("employee not found or tenant mismatch")
	}
	if !emp.IsActive {
		return nil, errors.New("employee account is inactive")
	}
	if emp.FaceEmbedding == nil {
		return nil, errors.New("employee face is not enrolled yet. Please register your face first")
	}

	var validationNotes []string

	// 2. Geofencing check
	var distanceMeters float64 = 0.0
	var allowedRadius int = 50 // default fallback radius in meters
	geofenceValid := true

	if emp.OfficeID != nil {
		office, err := s.officeRepo.GetByID(ctx, *emp.OfficeID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch assigned office: %w", err)
		}
		if office != nil {
			allowedRadius = office.RadiusMeters
			distanceMeters = utils.HaversineDistance(req.Latitude, req.Longitude, office.Latitude, office.Longitude)
			if distanceMeters > float64(office.RadiusMeters) {
				geofenceValid = false
				validationNotes = append(validationNotes, fmt.Sprintf(
					"Out of geofence bounds: current distance %.1fm exceeds office limit %dm",
					distanceMeters, office.RadiusMeters,
				))
			}
		}
	}

	// 3. Extract 512-d face embedding from uploaded selfie via AI Engine
	embedding, err := s.aiClient.ExtractFace(ctx, req.ImageFilename, req.ImageData)
	if err != nil {
		return nil, fmt.Errorf("face recognition failed: %w", err)
	}

	// 4. Calculate Cosine Similarity in PostgreSQL using pgvector <=> operator
	vecString := utils.VectorToString(embedding)
	similarity, err := s.employeeRepo.CalculateCosineSimilarity(ctx, req.EmployeeID, req.TenantID, vecString)
	if err != nil {
		return nil, fmt.Errorf("failed to verify face embedding against master vector: %w", err)
	}

	faceMatchValid := similarity >= s.cfg.SimilarityThreshold
	if !faceMatchValid {
		validationNotes = append(validationNotes, fmt.Sprintf(
			"Face match failed: similarity score %.4f is below minimum threshold %.2f",
			similarity, s.cfg.SimilarityThreshold,
		))
	}

	// 5. Overall validation decision
	isValid := geofenceValid && faceMatchValid

	// 6. Record transaction in attendance_logs
	log := &model.AttendanceLog{
		TenantID:        req.TenantID,
		UserID:          req.EmployeeID,
		EmployeeID:      req.EmployeeID,
		ClockTime:       time.Now(),
		AttendanceType:  req.AttendanceType,
		SimilarityScore: similarity,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		DistanceMeters:  distanceMeters,
		DeviceID:        req.DeviceID,
		PhotoURL:        req.PhotoURL,
		IsValid:         isValid,
	}

	if err := s.attendanceRepo.Create(ctx, log); err != nil {
		return nil, fmt.Errorf("failed to record attendance transaction: %w", err)
	}

	return &CheckInResult{
		AttendanceLog:   log,
		SimilarityScore: similarity,
		DistanceMeters:  distanceMeters,
		AllowedRadius:   allowedRadius,
		IsValid:         isValid,
		ValidationNotes: validationNotes,
	}, nil
}

func (s *attendanceService) GetAttendanceHistory(ctx context.Context, employeeID uuid.UUID, limit int) ([]*model.AttendanceLog, error) {
	return s.attendanceRepo.GetByEmployeeID(ctx, employeeID, limit)
}

func (s *attendanceService) ListLogs(ctx context.Context, tenantID uuid.UUID, page, limit int, startDate, endDate *time.Time, userID *uuid.UUID, isValid *bool) (*AttendanceLogsResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	logs, total, err := s.attendanceRepo.ListLogs(ctx, tenantID, limit, offset, startDate, endDate, userID, isValid)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance logs: %w", err)
	}

	return &AttendanceLogsResult{
		Total: total,
		Page:  page,
		Limit: limit,
		Logs:  logs,
	}, nil
}

func (s *attendanceService) GetSummary(ctx context.Context, tenantID uuid.UUID, targetDate time.Time) (*repository.AttendanceSummary, error) {
	return s.attendanceRepo.GetSummary(ctx, tenantID, targetDate)
}

