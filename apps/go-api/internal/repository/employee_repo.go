package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmployeeRepository interface {
	Create(ctx context.Context, employee *model.Employee) error
	BulkCreate(ctx context.Context, employees []*model.Employee) ([]*model.Employee, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
	BulkDelete(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) (int64, error)
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*model.Employee, error)
	GetByEmail(ctx context.Context, email string) (*model.Employee, *model.Tenant, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int, search string, officeID *uuid.UUID, role string, isActive *bool) ([]*model.User, int, error)
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
		SELECT id, tenant_id, office_id, role, name, email, password_hash, user_code,
		       face_embedding::text, face_registered_at, is_active, created_at
		FROM users
		WHERE id = $1 AND ($2 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $2)
	`
	var e model.Employee
	var embeddingStr *string
	var faceRegAt *time.Time

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&e.ID, &e.TenantID, &e.OfficeID, &e.Role, &e.Name, &e.Email, &e.PasswordHash, &e.UserCode,
		&embeddingStr, &faceRegAt, &e.IsActive, &e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	e.EmployeeCode = e.UserCode
	e.FaceEmbedding = embeddingStr
	e.FaceRegisteredAt = faceRegAt
	return &e, nil
}

func (r *employeeRepository) GetByEmail(ctx context.Context, email string) (*model.Employee, *model.Tenant, error) {
	query := `
		SELECT u.id, u.tenant_id, u.office_id, u.role, u.name, u.email, u.password_hash, u.user_code,
		       u.face_embedding::text, u.face_registered_at, u.is_active, u.created_at,
		       t.id, t.name, t.subdomain, t.is_active, t.created_at
		FROM users u
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE u.email = $1
	`
	var e model.Employee
	var embeddingStr *string
	var faceRegAt *time.Time

	var tenantID *uuid.UUID
	var tenantName sql.NullString
	var tenantSubdomain sql.NullString
	var tenantIsActive sql.NullBool
	var tenantCreatedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, email).Scan(
		&e.ID, &e.TenantID, &e.OfficeID, &e.Role, &e.Name, &e.Email, &e.PasswordHash, &e.UserCode,
		&embeddingStr, &faceRegAt, &e.IsActive, &e.CreatedAt,
		&tenantID, &tenantName, &tenantSubdomain, &tenantIsActive, &tenantCreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	e.EmployeeCode = e.UserCode
	e.FaceEmbedding = embeddingStr
	e.FaceRegisteredAt = faceRegAt

	var tenant *model.Tenant
	if tenantID != nil && tenantName.Valid {
		tenant = &model.Tenant{
			ID:        *tenantID,
			Name:      tenantName.String,
			Subdomain: tenantSubdomain.String,
			IsActive:  tenantIsActive.Bool,
			CreatedAt: tenantCreatedAt.Time,
		}
	}

	return &e, tenant, nil
}

func (r *employeeRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int, search string, officeID *uuid.UUID, role string, isActive *bool) ([]*model.User, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	whereClauses := []string{"($1 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $1)"}
	args := []interface{}{tenantID}
	argIdx := 2

	if search = strings.TrimSpace(search); search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR user_code ILIKE $%d OR COALESCE(email, '') ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if officeID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("office_id = $%d", argIdx))
		args = append(args, *officeID)
		argIdx++
	}

	if role = strings.TrimSpace(role); role != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, role)
		argIdx++
	}

	if isActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *isActive)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", whereSQL)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, office_id, role, name, email, password_hash, user_code,
		       face_embedding::text, face_registered_at, is_active, created_at
		FROM users
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		var embeddingStr *string
		var faceRegAt *time.Time
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.OfficeID, &u.Role, &u.Name, &u.Email, &u.PasswordHash, &u.UserCode,
			&embeddingStr, &faceRegAt, &u.IsActive, &u.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		u.EmployeeCode = u.UserCode
		u.FaceEmbedding = embeddingStr
		u.FaceRegisteredAt = faceRegAt
		users = append(users, &u)
	}

	return users, total, nil
}

func (r *employeeRepository) Update(ctx context.Context, user *model.User) error {
	var query string
	var err error

	code := user.UserCode
	if code == "" {
		code = user.EmployeeCode
	}

	if user.PasswordHash != "" {
		query = `
			UPDATE users
			SET name = $1, email = $2, user_code = $3, office_id = $4, is_active = $5, role = $6, password_hash = $7
			WHERE id = $8 AND ($9 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $9)
		`
		var tenantID uuid.UUID
		if user.TenantID != nil {
			tenantID = *user.TenantID
		}
		cmdTag, execErr := r.db.Exec(ctx, query,
			user.Name, user.Email, code, user.OfficeID, user.IsActive, user.Role, user.PasswordHash, user.ID, tenantID,
		)
		if execErr != nil {
			return execErr
		}
		if cmdTag.RowsAffected() == 0 {
			return errors.New("user not found or tenant mismatch")
		}
		return nil
	}

	query = `
		UPDATE users
		SET name = $1, email = $2, user_code = $3, office_id = $4, is_active = $5, role = $6
		WHERE id = $7 AND ($8 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $8)
	`
	var tenantID uuid.UUID
	if user.TenantID != nil {
		tenantID = *user.TenantID
	}
	cmdTag, execErr := r.db.Exec(ctx, query,
		user.Name, user.Email, code, user.OfficeID, user.IsActive, user.Role, user.ID, tenantID,
	)
	if execErr != nil {
		return execErr
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("user not found or tenant mismatch")
	}
	return err
}

func (r *employeeRepository) UpdateFaceEmbedding(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, embeddingStr string) error {
	query := `
		UPDATE users
		SET face_embedding = $1::vector,
		    face_registered_at = NOW()
		WHERE id = $2 AND ($3 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $3)
	`
	cmdTag, err := r.db.Exec(ctx, query, embeddingStr, id, tenantID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("user not found or tenant mismatch")
	}
	return nil
}

func (r *employeeRepository) CalculateCosineSimilarity(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, embeddingStr string) (float64, error) {
	query := `
		SELECT (1.0 - (face_embedding <=> $1::vector)) AS similarity
		FROM users
		WHERE id = $2 AND ($3 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $3) AND face_embedding IS NOT NULL
	`
	var similarity float64
	err := r.db.QueryRow(ctx, query, embeddingStr, id, tenantID).Scan(&similarity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0.0, errors.New("user face is not enrolled or user not found")
		}
		return 0.0, err
	}
	return similarity, nil
}

func (r *employeeRepository) Create(ctx context.Context, employee *model.Employee) error {
	query := `
		INSERT INTO users (
			id, tenant_id, office_id, role, name, email, password_hash, user_code, is_active, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at
	`
	if employee.ID == uuid.Nil {
		employee.ID = uuid.New()
	}
	if employee.CreatedAt.IsZero() {
		employee.CreatedAt = time.Now()
	}
	if employee.Role == "" {
		employee.Role = "user"
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
		employee.Role,
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
		INSERT INTO users (
			id, tenant_id, office_id, role, name, email, password_hash, user_code, is_active, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
		if emp.Role == "" {
			emp.Role = "user"
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
			emp.Role,
			emp.Name,
			emp.Email,
			emp.PasswordHash,
			code,
			emp.IsActive,
			emp.CreatedAt,
		).Scan(&emp.CreatedAt)
		if err != nil {
			var idStr string
			if emp.Email != nil {
				idStr = *emp.Email
			} else {
				idStr = emp.UserCode
			}
			return nil, fmt.Errorf("failed to insert user (%s): %w", idStr, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit bulk insert: %w", err)
	}

	return employees, nil
}

func (r *employeeRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1 AND ($2 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $2)`
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

	query := `DELETE FROM users WHERE id = ANY($1) AND ($2 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $2)`
	cmdTag, err := r.db.Exec(ctx, query, ids, tenantID)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return NewEmployeeRepository(db)
}
