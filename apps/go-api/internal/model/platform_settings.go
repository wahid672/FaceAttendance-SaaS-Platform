package model

import (
	"time"
)

type PlatformSettings struct {
	ID           int       `json:"id"`
	AppName      string    `json:"app_name"`
	LogoURL      *string   `json:"logo_url,omitempty"`
	FaviconURL   *string   `json:"favicon_url,omitempty"`
	CompanyName  *string   `json:"company_name,omitempty"`
	SupportEmail *string   `json:"support_email,omitempty"`
	FooterText   *string   `json:"footer_text,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}
