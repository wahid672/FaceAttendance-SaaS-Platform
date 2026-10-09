package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/faceattendance/go-api/internal/repository"
	"github.com/faceattendance/go-api/internal/utils"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserRequest struct {
	TenantID     uuid.UUID
	OfficeID     *uuid.UUID
	Name         string
	Email        string
	Password     string
	UserCode     string
	EmployeeCode string
}

type CreateEmployeeRequest = CreateUserRequest

type BulkItemError struct {
	Index int    `json:"index,omitempty"`
	Row   int    `json:"row,omitempty"`
	Email string `json:"email,omitempty"`
	Error string `json:"error"`
}

type BulkCreateResult struct {
	TotalRequested int             `json:"total_requested"`
	SuccessCount   int             `json:"success_count"`
	FailedCount    int             `json:"failed_count"`
	Errors         []BulkItemError `json:"errors"`
	Users          []*model.User   `json:"users"`
}

type BulkDeleteResult struct {
	TotalRequested int         `json:"total_requested"`
	DeletedCount   int64       `json:"deleted_count"`
	SkippedSelf    bool        `json:"skipped_self"`
	DeletedIDs     []uuid.UUID `json:"deleted_ids"`
}

type UserService interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*model.User, error)
	CreateEmployee(ctx context.Context, req CreateEmployeeRequest) (*model.Employee, error)
	BulkCreateUsers(ctx context.Context, tenantID uuid.UUID, requests []CreateUserRequest) (*BulkCreateResult, error)
	ImportUsersCSV(ctx context.Context, tenantID uuid.UUID, csvReader io.Reader) (*BulkCreateResult, error)
	DeleteUser(ctx context.Context, tenantID, callerUserID, targetUserID uuid.UUID) error
	DeleteEmployee(ctx context.Context, tenantID, callerEmployeeID, targetEmployeeID uuid.UUID) error
	BulkDeleteUsers(ctx context.Context, tenantID, callerUserID uuid.UUID, targetUserIDs []uuid.UUID) (*BulkDeleteResult, error)
	EnrollFace(ctx context.Context, tenantID, userID uuid.UUID, files []UploadedFile) ([]float32, error)
	GetUser(ctx context.Context, tenantID, userID uuid.UUID) (*model.User, error)
	GetEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (*model.Employee, error)
}

type EmployeeService = UserService

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

func NewUserService(employeeRepo repository.EmployeeRepository, aiClient AIEngineClient) UserService {
	return NewEmployeeService(employeeRepo, aiClient)
}

func (s *employeeService) CreateEmployee(ctx context.Context, req CreateEmployeeRequest) (*model.Employee, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	code := strings.TrimSpace(req.UserCode)
	if code == "" {
		code = strings.TrimSpace(req.EmployeeCode)
	}

	if name == "" {
		return nil, errors.New("name is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}
	if code == "" {
		return nil, errors.New("user_code / employee_code is required")
	}

	// Check if email already exists
	existingEmp, _, err := s.employeeRepo.GetByEmail(ctx, email)
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

	emp := &model.User{
		ID:           uuid.New(),
		TenantID:     req.TenantID,
		OfficeID:     req.OfficeID,
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
		UserCode:     code,
		EmployeeCode: code,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	if err := s.employeeRepo.Create(ctx, emp); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return emp, nil
}

func (s *employeeService) CreateUser(ctx context.Context, req CreateUserRequest) (*model.User, error) {
	return s.CreateEmployee(ctx, req)
}

func (s *employeeService) BulkCreateUsers(ctx context.Context, tenantID uuid.UUID, requests []CreateUserRequest) (*BulkCreateResult, error) {
	result := &BulkCreateResult{
		TotalRequested: len(requests),
		Errors:         []BulkItemError{},
		Users:          []*model.User{},
	}

	seenEmails := make(map[string]bool)
	var validUsers []*model.User

	for i, req := range requests {
		name := strings.TrimSpace(req.Name)
		email := strings.ToLower(strings.TrimSpace(req.Email))
		code := strings.TrimSpace(req.UserCode)
		if code == "" {
			code = strings.TrimSpace(req.EmployeeCode)
		}

		if name == "" {
			result.Errors = append(result.Errors, BulkItemError{Index: i, Email: email, Error: "name is required"})
			result.FailedCount++
			continue
		}
		if email == "" {
			result.Errors = append(result.Errors, BulkItemError{Index: i, Email: email, Error: "email is required"})
			result.FailedCount++
			continue
		}
		if len(req.Password) < 6 {
			result.Errors = append(result.Errors, BulkItemError{Index: i, Email: email, Error: "password must be at least 6 characters"})
			result.FailedCount++
			continue
		}
		if code == "" {
			result.Errors = append(result.Errors, BulkItemError{Index: i, Email: email, Error: "user_code / employee_code is required"})
			result.FailedCount++
			continue
		}

		if seenEmails[email] {
			result.Errors = append(result.Errors, BulkItemError{Index: i, Email: email, Error: "duplicate email in request batch"})
			result.FailedCount++
			continue
		}
		seenEmails[email] = true

		existing, _, err := s.employeeRepo.GetByEmail(ctx, email)
		if err != nil {
			result.Errors = append(result.Errors, BulkItemError{Index: i, Email: email, Error: fmt.Sprintf("db check failed: %v", err)})
			result.FailedCount++
			continue
		}
		if existing != nil {
			result.Errors = append(result.Errors, BulkItemError{Index: i, Email: email, Error: "email is already registered in database"})
			result.FailedCount++
			continue
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			result.Errors = append(result.Errors, BulkItemError{Index: i, Email: email, Error: fmt.Sprintf("password hash failed: %v", err)})
			result.FailedCount++
			continue
		}

		user := &model.User{
			ID:           uuid.New(),
			TenantID:     tenantID,
			OfficeID:     req.OfficeID,
			Name:         name,
			Email:        email,
			PasswordHash: string(hashedPassword),
			UserCode:     code,
			EmployeeCode: code,
			IsActive:     true,
			CreatedAt:    time.Now(),
		}
		validUsers = append(validUsers, user)
	}

	if len(validUsers) > 0 {
		created, err := s.employeeRepo.BulkCreate(ctx, validUsers)
		if err != nil {
			return nil, fmt.Errorf("failed to bulk insert users: %w", err)
		}
		result.Users = created
		result.SuccessCount = len(created)
	}

	return result, nil
}

func (s *employeeService) ImportUsersCSV(ctx context.Context, tenantID uuid.UUID, csvReader io.Reader) (*BulkCreateResult, error) {
	reader := csv.NewReader(csvReader)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, errors.New("CSV file is empty")
	}

	headerRow := records[0]
	colMap := make(map[string]int)
	hasHeader := false

	for i, col := range headerRow {
		clean := strings.ToLower(strings.TrimSpace(col))
		colMap[clean] = i
		if clean == "email" || clean == "name" || clean == "nama" {
			hasHeader = true
		}
	}

	startRow := 0
	if hasHeader {
		startRow = 1
	}

	var requests []CreateUserRequest
	var rowNumbers []int

	for r := startRow; r < len(records); r++ {
		row := records[r]
		if len(row) == 0 {
			continue
		}

		getVal := func(keys []string, fallbackIdx int) string {
			for _, k := range keys {
				if idx, ok := colMap[k]; ok && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
			}
			if fallbackIdx < len(row) {
				return strings.TrimSpace(row[fallbackIdx])
			}
			return ""
		}

		name := getVal([]string{"name", "nama"}, 0)
		email := getVal([]string{"email"}, 1)
		password := getVal([]string{"password", "pass"}, 2)
		code := getVal([]string{"user_code", "employee_code", "kode", "nis", "nip"}, 3)
		officeStr := getVal([]string{"office_id", "kantor_id"}, 4)

		if name == "" && email == "" && password == "" {
			continue // skip empty lines
		}

		var officeUUID *uuid.UUID
		if officeStr != "" {
			if parsed, err := uuid.Parse(officeStr); err == nil {
				officeUUID = &parsed
			}
		}

		requests = append(requests, CreateUserRequest{
			TenantID:     tenantID,
			OfficeID:     officeUUID,
			Name:         name,
			Email:        email,
			Password:     password,
			UserCode:     code,
			EmployeeCode: code,
		})
		rowNumbers = append(rowNumbers, r+1)
	}

	res, err := s.BulkCreateUsers(ctx, tenantID, requests)
	if err != nil {
		return nil, err
	}

	for i := range res.Errors {
		idx := res.Errors[i].Index
		if idx < len(rowNumbers) {
			res.Errors[i].Row = rowNumbers[idx]
		}
	}

	return res, nil
}

func (s *employeeService) EnrollFace(ctx context.Context, tenantID, employeeID uuid.UUID, files []UploadedFile) ([]float32, error) {
	if len(files) == 0 {
		return nil, errors.New("at least one face photo is required for enrollment")
	}

	emp, err := s.employeeRepo.GetByID(ctx, employeeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user: %w", err)
	}
	if emp == nil {
		return nil, errors.New("user not found or tenant mismatch")
	}

	var embedding []float32
	if len(files) == 1 {
		emb, err := s.aiClient.ExtractFace(ctx, files[0].Filename, files[0].Data)
		if err != nil {
			return nil, fmt.Errorf("face extraction failed: %w", err)
		}
		embedding = emb
	} else {
		emb, err := s.aiClient.EnrollMerge(ctx, files)
		if err != nil {
			return nil, fmt.Errorf("face enrollment merge failed: %w", err)
		}
		embedding = emb
	}

	if len(embedding) != 512 {
		return nil, fmt.Errorf("invalid embedding dimension: expected 512, got %d", len(embedding))
	}

	vecString := utils.VectorToString(embedding)

	if err := s.employeeRepo.UpdateFaceEmbedding(ctx, employeeID, tenantID, vecString); err != nil {
		return nil, fmt.Errorf("failed to save face embedding to database: %w", err)
	}

	return embedding, nil
}

func (s *employeeService) GetEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (*model.Employee, error) {
	return s.employeeRepo.GetByID(ctx, employeeID, tenantID)
}

func (s *employeeService) GetUser(ctx context.Context, tenantID, userID uuid.UUID) (*model.User, error) {
	return s.GetEmployee(ctx, userID, tenantID)
}

func (s *employeeService) DeleteEmployee(ctx context.Context, tenantID, callerEmployeeID, targetEmployeeID uuid.UUID) error {
	if callerEmployeeID == targetEmployeeID {
		return errors.New("cannot delete your own user account")
	}

	emp, err := s.employeeRepo.GetByID(ctx, targetEmployeeID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to verify user: %w", err)
	}
	if emp == nil {
		return errors.New("user not found or tenant mismatch")
	}

	if err := s.employeeRepo.Delete(ctx, targetEmployeeID, tenantID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (s *employeeService) DeleteUser(ctx context.Context, tenantID, callerUserID, targetUserID uuid.UUID) error {
	return s.DeleteEmployee(ctx, tenantID, callerUserID, targetUserID)
}

func (s *employeeService) BulkDeleteUsers(ctx context.Context, tenantID, callerUserID uuid.UUID, targetUserIDs []uuid.UUID) (*BulkDeleteResult, error) {
	result := &BulkDeleteResult{
		TotalRequested: len(targetUserIDs),
		DeletedIDs:     []uuid.UUID{},
	}

	var toDelete []uuid.UUID
	for _, id := range targetUserIDs {
		if id == callerUserID {
			result.SkippedSelf = true
			continue
		}
		toDelete = append(toDelete, id)
	}

	if len(toDelete) > 0 {
		count, err := s.employeeRepo.BulkDelete(ctx, toDelete, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to bulk delete users: %w", err)
		}
		result.DeletedCount = count
		result.DeletedIDs = toDelete
	}

	return result, nil
}
