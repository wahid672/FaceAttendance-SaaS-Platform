package service

import (
	"context"
	"errors"
	"time"

	"github.com/faceattendance/go-api/internal/config"
	"github.com/faceattendance/go-api/internal/model"
	"github.com/faceattendance/go-api/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, *model.Employee, *model.Tenant, error)
	ValidateToken(tokenString string) (*model.JWTClaims, error)
}

type authService struct {
	cfg          *config.Config
	employeeRepo repository.EmployeeRepository
}

func NewAuthService(cfg *config.Config, employeeRepo repository.EmployeeRepository) AuthService {
	return &authService{
		cfg:          cfg,
		employeeRepo: employeeRepo,
	}
}

func (s *authService) Login(ctx context.Context, email, password string) (string, *model.Employee, *model.Tenant, error) {
	employee, tenant, err := s.employeeRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, nil, err
	}
	if employee == nil || tenant == nil {
		return "", nil, nil, errors.New("invalid email or password")
	}

	if !tenant.IsActive {
		return "", nil, nil, errors.New("tenant account is inactive")
	}

	if !employee.IsActive {
		return "", nil, nil, errors.New("employee account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(password)); err != nil {
		return "", nil, nil, errors.New("invalid email or password")
	}

	// Generate JWT Token (valid for 24 hours)
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &model.JWTClaims{
		TenantID:   tenant.ID,
		EmployeeID: employee.ID,
		OfficeID:   employee.OfficeID,
		Email:      employee.Email,
		Name:       employee.Name,
		Role:       "employee",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "faceattendance-go-api",
			Subject:   employee.ID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", nil, nil, err
	}

	return tokenString, employee, tenant, nil
}

func (s *authService) ValidateToken(tokenString string) (*model.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*model.JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
