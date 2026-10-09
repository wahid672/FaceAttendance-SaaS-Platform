package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error)
	Create(ctx context.Context, tenant *model.Tenant) error
	Update(ctx context.Context, tenant *model.Tenant) error
	List(ctx context.Context, limit, offset int, search string) ([]*model.Tenant, int, error)
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

func (r *tenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO tenants (id, name, subdomain, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`
	return r.db.QueryRow(ctx, query,
		tenant.ID, tenant.Name, tenant.Subdomain, tenant.IsActive, tenant.CreatedAt,
	).Scan(&tenant.CreatedAt)
}

func (r *tenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $1, subdomain = $2, is_active = $3
		WHERE id = $4
	`
	cmdTag, err := r.db.Exec(ctx, query, tenant.Name, tenant.Subdomain, tenant.IsActive, tenant.ID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("tenant not found")
	}
	return nil
}

func (r *tenantRepository) List(ctx context.Context, limit, offset int, search string) ([]*model.Tenant, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	search = strings.TrimSpace(search)
	var countQuery string
	var listQuery string
	var total int

	if search != "" {
		filter := "%" + search + "%"
		countQuery = `SELECT COUNT(*) FROM tenants WHERE name ILIKE $1 OR subdomain ILIKE $1`
		if err := r.db.QueryRow(ctx, countQuery, filter).Scan(&total); err != nil {
			return nil, 0, err
		}

		listQuery = `
			SELECT id, name, subdomain, is_active, created_at
			FROM tenants
			WHERE name ILIKE $1 OR subdomain ILIKE $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		rows, err := r.db.Query(ctx, listQuery, filter, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()

		var tenants []*model.Tenant
		for rows.Next() {
			var t model.Tenant
			if err := rows.Scan(&t.ID, &t.Name, &t.Subdomain, &t.IsActive, &t.CreatedAt); err != nil {
				return nil, 0, err
			}
			tenants = append(tenants, &t)
		}
		return tenants, total, nil
	}

	countQuery = `SELECT COUNT(*) FROM tenants`
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery = `
		SELECT id, name, subdomain, is_active, created_at
		FROM tenants
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []*model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Subdomain, &t.IsActive, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, &t)
	}
	return tenants, total, nil
}
