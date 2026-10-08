package repository

import (
	"context"
	"errors"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OfficeRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Office, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*model.Office, error)
}

type officeRepository struct {
	db *pgxpool.Pool
}

func NewOfficeRepository(db *pgxpool.Pool) OfficeRepository {
	return &officeRepository{db: db}
}

func (r *officeRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Office, error) {
	query := `
		SELECT id, tenant_id, name, latitude, longitude, radius_meters, created_at 
		FROM offices 
		WHERE id = $1
	`
	var o model.Office
	err := r.db.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.TenantID, &o.Name, &o.Latitude, &o.Longitude, &o.RadiusMeters, &o.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *officeRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*model.Office, error) {
	query := `
		SELECT id, tenant_id, name, latitude, longitude, radius_meters, created_at 
		FROM offices 
		WHERE tenant_id = $1
	`
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var offices []*model.Office
	for rows.Next() {
		var o model.Office
		if err := rows.Scan(&o.ID, &o.TenantID, &o.Name, &o.Latitude, &o.Longitude, &o.RadiusMeters, &o.CreatedAt); err != nil {
			return nil, err
		}
		offices = append(offices, &o)
	}
	return offices, nil
}
