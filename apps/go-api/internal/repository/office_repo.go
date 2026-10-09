package repository

import (
	"context"
	"errors"
	"time"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OfficeRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Office, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*model.Office, error)
	Create(ctx context.Context, office *model.Office) error
	Update(ctx context.Context, office *model.Office) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
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
		ORDER BY created_at ASC
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

func (r *officeRepository) Create(ctx context.Context, office *model.Office) error {
	if office.ID == uuid.Nil {
		office.ID = uuid.New()
	}
	if office.CreatedAt.IsZero() {
		office.CreatedAt = time.Now()
	}
	if office.RadiusMeters <= 0 {
		office.RadiusMeters = 50
	}

	query := `
		INSERT INTO offices (id, tenant_id, name, latitude, longitude, radius_meters, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`
	return r.db.QueryRow(ctx, query,
		office.ID, office.TenantID, office.Name, office.Latitude, office.Longitude, office.RadiusMeters, office.CreatedAt,
	).Scan(&office.CreatedAt)
}

func (r *officeRepository) Update(ctx context.Context, office *model.Office) error {
	query := `
		UPDATE offices 
		SET name = $1, latitude = $2, longitude = $3, radius_meters = $4
		WHERE id = $5 AND tenant_id = $6
	`
	cmdTag, err := r.db.Exec(ctx, query,
		office.Name, office.Latitude, office.Longitude, office.RadiusMeters, office.ID, office.TenantID,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("office not found or tenant mismatch")
	}
	return nil
}

func (r *officeRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `DELETE FROM offices WHERE id = $1 AND tenant_id = $2`
	cmdTag, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("office not found or tenant mismatch")
	}
	return nil
}
