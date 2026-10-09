package model

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	TenantID   uuid.UUID  `json:"tenant_id"`
	UserID     uuid.UUID  `json:"user_id"`
	EmployeeID uuid.UUID  `json:"employee_id"`
	OfficeID   *uuid.UUID `json:"office_id,omitempty"`
	Email      string     `json:"email"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	jwt.RegisteredClaims
}
