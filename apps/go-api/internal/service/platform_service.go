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
	"golang.org/x/crypto/bcrypt"
)

type CreateTenantRequest struct {
	Name            string `json:"name"`
	Subdomain       string `json:"subdomain"`
	AdminName       string `json:"admin_name"`
	AdminEmail      string `json:"admin_email"`
	AdminPassword   string `json:"admin_password"`
	AdminUserCode   string `json:"admin_user_code"`
}

type TenantListResult struct {
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
	Tenants []*model.Tenant `json:"tenants"`
}

type PlatformService interface {
	GetSettings(ctx context.Context) (*model.PlatformSettings, error)
	UpdateSettings(ctx context.Context, settings *model.PlatformSettings) (*model.PlatformSettings, error)
	ListTenants(ctx context.Context, page, limit int, search string) (*TenantListResult, error)
	GetTenant(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	CreateTenant(ctx context.Context, req CreateTenantRequest) (*model.Tenant, *model.User, error)
	UpdateTenant(ctx context.Context, id uuid.UUID, name, subdomain string, isActive bool) (*model.Tenant, error)
}

type platformService struct {
	settingsRepo repository.PlatformSettingsRepository
	tenantRepo   repository.TenantRepository
	userRepo     repository.UserRepository
}

func NewPlatformService(
	settingsRepo repository.PlatformSettingsRepository,
	tenantRepo repository.TenantRepository,
	userRepo repository.UserRepository,
) PlatformService {
	return &platformService{
		settingsRepo: settingsRepo,
		tenantRepo:   tenantRepo,
		userRepo:     userRepo,
	}
}

func (s *platformService) GetSettings(ctx context.Context) (*model.PlatformSettings, error) {
	return s.settingsRepo.Get(ctx)
}

func (s *platformService) UpdateSettings(ctx context.Context, settings *model.PlatformSettings) (*model.PlatformSettings, error) {
	if strings.TrimSpace(settings.AppName) == "" {
		return nil, errors.New("app_name cannot be empty")
	}
	if err := s.settingsRepo.Update(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to update platform settings: %w", err)
	}
	return s.settingsRepo.Get(ctx)
}

func (s *platformService) ListTenants(ctx context.Context, page, limit int, search string) (*TenantListResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	tenants, total, err := s.tenantRepo.List(ctx, limit, offset, search)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	return &TenantListResult{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Tenants: tenants,
	}, nil
}

func (s *platformService) GetTenant(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errors.New("tenant not found")
	}
	return tenant, nil
}

func (s *platformService) CreateTenant(ctx context.Context, req CreateTenantRequest) (*model.Tenant, *model.User, error) {
	name := strings.TrimSpace(req.Name)
	subdomain := strings.ToLower(strings.TrimSpace(req.Subdomain))
	adminEmail := strings.ToLower(strings.TrimSpace(req.AdminEmail))
	adminName := strings.TrimSpace(req.AdminName)
	adminCode := strings.TrimSpace(req.AdminUserCode)

	if name == "" {
		return nil, nil, errors.New("tenant name is required")
	}
	if subdomain == "" {
		return nil, nil, errors.New("subdomain is required")
	}
	if adminEmail == "" {
		return nil, nil, errors.New("admin_email is required")
	}
	if len(req.AdminPassword) < 6 {
		return nil, nil, errors.New("admin_password must be at least 6 characters")
	}
	if adminName == "" {
		adminName = "Administrator " + name
	}
	if adminCode == "" {
		adminCode = "ADM-001"
	}

	// Check if subdomain already exists
	existingTenant, err := s.tenantRepo.GetBySubdomain(ctx, subdomain)
	if err != nil {
		return nil, nil, err
	}
	if existingTenant != nil {
		return nil, nil, errors.New("subdomain is already taken")
	}

	// Check if admin email already exists
	existingUser, _, err := s.userRepo.GetByEmail(ctx, adminEmail)
	if err != nil {
		return nil, nil, err
	}
	if existingUser != nil {
		return nil, nil, errors.New("admin email is already registered")
	}

	tenant := &model.Tenant{
		ID:        uuid.New(),
		Name:      name,
		Subdomain: subdomain,
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	adminUser := &model.User{
		ID:           uuid.New(),
		TenantID:     &tenant.ID,
		Role:         "tenant_admin",
		Name:         adminName,
		Email:        &adminEmail,
		PasswordHash: string(hash),
		UserCode:     adminCode,
		EmployeeCode: adminCode,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, adminUser); err != nil {
		return nil, nil, fmt.Errorf("failed to create tenant admin: %w", err)
	}

	return tenant, adminUser, nil
}

func (s *platformService) UpdateTenant(ctx context.Context, id uuid.UUID, name, subdomain string, isActive bool) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errors.New("tenant not found")
	}

	name = strings.TrimSpace(name)
	subdomain = strings.ToLower(strings.TrimSpace(subdomain))

	if name != "" {
		tenant.Name = name
	}
	if subdomain != "" && subdomain != tenant.Subdomain {
		existing, err := s.tenantRepo.GetBySubdomain(ctx, subdomain)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, errors.New("subdomain is already taken by another tenant")
		}
		tenant.Subdomain = subdomain
	}
	tenant.IsActive = isActive

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	return tenant, nil
}
