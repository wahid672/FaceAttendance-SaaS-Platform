package model

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Subdomain string    `json:"subdomain"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
