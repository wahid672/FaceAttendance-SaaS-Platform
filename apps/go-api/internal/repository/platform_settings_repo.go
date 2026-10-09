package repository

import (
	"context"
	"errors"
	"time"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlatformSettingsRepository interface {
	Get(ctx context.Context) (*model.PlatformSettings, error)
	Update(ctx context.Context, settings *model.PlatformSettings) error
}

type platformSettingsRepository struct {
	db *pgxpool.Pool
}

func NewPlatformSettingsRepository(db *pgxpool.Pool) PlatformSettingsRepository {
	return &platformSettingsRepository{db: db}
}

func (r *platformSettingsRepository) Get(ctx context.Context) (*model.PlatformSettings, error) {
	query := `
		SELECT id, app_name, logo_url, favicon_url, company_name, support_email, footer_text, updated_at
		FROM platform_settings
		WHERE id = 1
	`
	var s model.PlatformSettings
	err := r.db.QueryRow(ctx, query).Scan(
		&s.ID, &s.AppName, &s.LogoURL, &s.FaviconURL, &s.CompanyName, &s.SupportEmail, &s.FooterText, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// fallback default if not seeded
			return &model.PlatformSettings{
				ID:        1,
				AppName:   "FaceAttendance SaaS Platform",
				UpdatedAt: time.Now(),
			}, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *platformSettingsRepository) Update(ctx context.Context, settings *model.PlatformSettings) error {
	query := `
		INSERT INTO platform_settings (id, app_name, logo_url, favicon_url, company_name, support_email, footer_text, updated_at)
		VALUES (1, $1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id) DO UPDATE SET
			app_name = EXCLUDED.app_name,
			logo_url = EXCLUDED.logo_url,
			favicon_url = EXCLUDED.favicon_url,
			company_name = EXCLUDED.company_name,
			support_email = EXCLUDED.support_email,
			footer_text = EXCLUDED.footer_text,
			updated_at = NOW()
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query,
		settings.AppName,
		settings.LogoURL,
		settings.FaviconURL,
		settings.CompanyName,
		settings.SupportEmail,
		settings.FooterText,
	).Scan(&settings.UpdatedAt)
}
