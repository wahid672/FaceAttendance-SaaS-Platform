package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmployeeRepository interface {
	Create(ctx context.Context, employee *model.Employee) error
	BulkCreate(ctx context.Context, employees []*model.Employee) ([]*model.Employee, error)
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
	BulkDelete(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) (int64, error)
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*model.Employee, error)
	GetByEmail(ctx context.Context, email string) (*model.Employee, *model.Tenant, error)
	UpdateFaceEmbedding(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, embeddingStr string) error
	CalculateCosineSimilarity(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, embeddingStr string) (float64, error)
}

type UserRepository = EmployeeRepository

type employeeRepository struct {
	db *pgxpool.Pool
}

func NewEmployeeRepository(db *pgxpool.Pool) EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*model.Employee, error) {
	query := `
		SELECT id, tenant_id, office_id, name, email, password_hash, employee_code,
		       face_embedding::text, face_registered_at, is_active, created_at
		FROM employees
		WHERE id = $1 AND tenant_id = $2
	`
	var e model.Employee
	var embeddingStr *string
	var faceRegAt *time.Time

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&e.ID, &e.TenantID, &e.OfficeID, &e.Name, &e.Email, &e.PasswordHash, &e.EmployeeCode,
		&embeddingStr, &faceRegAt, &e.IsActive, &e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	e.FaceEmbedding = embeddingStr
	e.FaceRegisteredAt = faceRegAt
	return &e, nil
}

func (r *employeeRepository) GetByEmail(ctx context.Context, email string) (*model.Employee, *model.Tenant, error) {
	query := `
		SELECT e.id, e.tenant_id, e.office_id, e.name, e.email, e.password_hash, e.employee_code,
		       e.face_embedding::text, e.face_registered_at, e.is_active, e.created_at,
		       t.id, t.name, t.subdomain, t.is_active, t.created_at
		FROM employees e
		JOIN tenants t ON t.id = e.tenant_id
		WHERE e.email = $1
	`
	var e model.Employee
	var t model.Tenant
	var embeddingStr *string
	var faceRegAt *time.Time

	err := r.db.QueryRow(ctx, query, email).Scan(
		&e.ID, &e.TenantID, &e.OfficeID, &e.Name, &e.Email, &e.PasswordHash, &e.EmployeeCode,
		&embeddingStr, &faceRegAt, &e.IsActive, &e.CreatedAt,
		&t.ID, &t.Name, &t.Subdomain, &t.IsActive, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	e.FaceEmbedding = embeddingStr
	e.FaceRegisteredAt = faceRegAt
	return &e, &t, nil
}

func (r *employeeRepository) UpdateFaceEmbedding(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, embeddingStr string) error {
	query := `
		UPDATE employees
		SET face_embedding = $1::vector,
		    face_registered_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`
	cmdTag, err := r.db.Exec(ctx, query, embeddingStr, id, tenantID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("employee not found or tenant mismatch")
	}
	return nil
}

func (r *employeeRepository) CalculateCosineSimilarity(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, embeddingStr string) (float64, error) {
	// Cosine distance in pgvector is calculated using <=> operator
	// Cosine similarity = 1 - (face_embedding <=> query_vector)
	query := `
		SELECT (1.0 - (face_embedding <=> $1::vector)) AS similarity
		FROM employees
		WHERE id = $2 AND tenant_id = $3 AND face_embedding IS NOT NULL
	`
	var similarity float64
	err := r.db.QueryRow(ctx, query, embeddingStr, id, tenantID).Scan(&similarity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0.0, errors.New("employee face is not enrolled or employee not found")
		}
		return 0.0, err
	}
	return similarity, nil
}

func (r *employeeRepository) Create(ctx context.Context, employee *model.Employee) error {
	query := `
		INSERT INTO employees (
			id, tenant_id, office_id, name, email, password_hash, employee_code, is_active, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`
	if employee.ID == uuid.Nil {
		employee.ID = uuid.New()
	}
	if employee.CreatedAt.IsZero() {
		employee.CreatedAt = time.Now()
	}

	code := employee.UserCode
	if code == "" {
		code = employee.EmployeeCode
	}
	employee.EmployeeCode = code
	employee.UserCode = code

	return r.db.QueryRow(ctx, query,
		employee.ID,
		employee.TenantID,
		employee.OfficeID,
		employee.Name,
		employee.Email,
		employee.PasswordHash,
		code,
		employee.IsActive,
		employee.CreatedAt,
	).Scan(&employee.CreatedAt)
}

func (r *employeeRepository) BulkCreate(ctx context.Context, employees []*model.Employee) ([]*model.Employee, error) {
	if len(employees) == 0 {
		return []*model.Employee{}, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO employees (
			id, tenant_id, office_id, name, email, password_hash, employee_code, is_active, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`

	now := time.Now()
	for _, emp := range employees {
		if emp.ID == uuid.Nil {
			emp.ID = uuid.New()
		}
		if emp.CreatedAt.IsZero() {
			emp.CreatedAt = now
		}
		code := emp.UserCode
		if code == "" {
			code = emp.EmployeeCode
		}
		emp.EmployeeCode = code
		emp.UserCode = code

		err := tx.QueryRow(ctx, query,
			emp.ID,
			emp.TenantID,
			emp.OfficeID,
			emp.Name,
			emp.Email,
			emp.PasswordHash,
			code,
			emp.IsActive,
			emp.CreatedAt,
		).Scan(&emp.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to insert user (%s): %w", emp.Email, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit bulk insert: %w", err)
	}

	return employees, nil
}

func (r *employeeRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `DELETE FROM employees WHERE id = $1 AND tenant_id = $2`
	cmdTag, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("user not found or tenant mismatch")
	}
	return nil
}

func (r *employeeRepository) BulkDelete(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	query := `DELETE FROM employees WHERE id = ANY($1) AND tenant_id = $2`
	cmdTag, err := r.db.Exec(ctx, query, ids, tenantID)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return NewEmployeeRepository(db)
}


