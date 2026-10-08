package model

import (
	"time"

	"github.com/google/uuid"
)

type Employee struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	OfficeID         *uuid.UUID `json:"office_id,omitempty"`
	Name             string     `json:"name"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"`
	EmployeeCode     string     `json:"employee_code"`
	FaceEmbedding    *string    `json:"face_embedding,omitempty"` // String representation of vector(512)
	FaceRegisteredAt *time.Time `json:"face_registered_at,omitempty"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
}
