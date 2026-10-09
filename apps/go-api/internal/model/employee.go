package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents any account in the platform:
// - superadmin: Platform owner (tenant_id is nil, email required)
// - tenant_admin: School / Institution admin (tenant_id set, email required)
// - user: Student, santri, teacher, employee (tenant_id set, email optional, identified by user_code)
type User struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         *uuid.UUID `json:"tenant_id,omitempty"`
	OfficeID         *uuid.UUID `json:"office_id,omitempty"`
	Role             string     `json:"role"` // 'superadmin', 'tenant_admin', 'user'
	Name             string     `json:"name"`
	Email            *string    `json:"email,omitempty"`
	PasswordHash     string     `json:"-"`
	UserCode         string     `json:"user_code"` // NIS, NISN, NIK, ID Unik
	EmployeeCode     string     `json:"employee_code,omitempty"`
	FaceEmbedding    *string    `json:"face_embedding,omitempty"` // String representation of vector(512)
	FaceRegisteredAt *time.Time `json:"face_registered_at,omitempty"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
}

// Type alias Employee to User for backward compatibility
type Employee = User
