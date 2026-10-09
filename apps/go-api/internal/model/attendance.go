package model

import (
	"time"

	"github.com/google/uuid"
)

type AttendanceLog struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	UserID          uuid.UUID `json:"user_id"`
	EmployeeID      uuid.UUID `json:"employee_id,omitempty"` // Alias compatibility
	ClockTime       time.Time `json:"clock_time"`
	AttendanceType  string    `json:"attendance_type"` // 'IN' or 'OUT'
	SimilarityScore float64   `json:"similarity_score"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	DistanceMeters  float64   `json:"distance_meters"`
	DeviceID        string    `json:"device_id"`
	PhotoURL        *string   `json:"photo_url,omitempty"`
	IsValid         bool      `json:"is_valid"`
	CreatedAt       time.Time `json:"created_at"`
}
