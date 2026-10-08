package repository

import (
	"context"
	"errors"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error)
}

type tenantRepository struct {
	db *pgxpool.Pool
}

func NewTenantRepository(db *pgxpool.Pool) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	query := `SELECT id, name, subdomain, is_active, created_at FROM tenants WHERE id = $1`
	var t model.Tenant
	err := r.db.QueryRow(ctx, query, id).Scan(&t.ID, &t.Name, &t.Subdomain, &t.IsActive, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *tenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error) {
	query := `SELECT id, name, subdomain, is_active, created_at FROM tenants WHERE subdomain = $1`
	var t model.Tenant
	err := r.db.QueryRow(ctx, query, subdomain).Scan(&t.ID, &t.Name, &t.Subdomain, &t.IsActive, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
