package model

import (
	"time"

	"github.com/google/uuid"
)

type Office struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	RadiusMeters int       `json:"radius_meters"`
	CreatedAt    time.Time `json:"created_at"`
}
