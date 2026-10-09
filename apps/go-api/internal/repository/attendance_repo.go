package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttendanceSummary struct {
	Date            string `json:"date"`
	TotalUsers      int    `json:"total_users"`
	TotalCheckIns   int    `json:"total_check_ins"`
	ValidCheckIns   int    `json:"valid_check_ins"`
	InvalidCheckIns int    `json:"invalid_check_ins"`
}

type AttendanceRepository interface {
	Create(ctx context.Context, log *model.AttendanceLog) error
	GetByEmployeeID(ctx context.Context, employeeID uuid.UUID, limit int) ([]*model.AttendanceLog, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*model.AttendanceLog, error)
	ListLogs(ctx context.Context, tenantID uuid.UUID, limit, offset int, startDate, endDate *time.Time, userID *uuid.UUID, isValid *bool) ([]*model.AttendanceLog, int, error)
	GetSummary(ctx context.Context, tenantID uuid.UUID, targetDate time.Time) (*AttendanceSummary, error)
}

type attendanceRepository struct {
	db *pgxpool.Pool
}

func NewAttendanceRepository(db *pgxpool.Pool) AttendanceRepository {
	return &attendanceRepository{db: db}
}

func (r *attendanceRepository) Create(ctx context.Context, log *model.AttendanceLog) error {
	query := `
		INSERT INTO attendance_logs (
			tenant_id, user_id, clock_time, attendance_type, similarity_score,
			latitude, longitude, distance_meters, device_id, photo_url, is_valid
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) RETURNING id, created_at
	`
	uid := log.UserID
	if uid == uuid.Nil {
		uid = log.EmployeeID
	}
	log.UserID = uid
	log.EmployeeID = uid

	return r.db.QueryRow(
		ctx, query,
		log.TenantID, uid, log.ClockTime, log.AttendanceType, log.SimilarityScore,
		log.Latitude, log.Longitude, log.DistanceMeters, log.DeviceID, log.PhotoURL, log.IsValid,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *attendanceRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*model.AttendanceLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, tenant_id, user_id, clock_time, attendance_type, similarity_score,
		       latitude, longitude, distance_meters, device_id, photo_url, is_valid, created_at
		FROM attendance_logs
		WHERE user_id = $1
		ORDER BY clock_time DESC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.AttendanceLog
	for rows.Next() {
		var l model.AttendanceLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.UserID, &l.ClockTime, &l.AttendanceType, &l.SimilarityScore,
			&l.Latitude, &l.Longitude, &l.DistanceMeters, &l.DeviceID, &l.PhotoURL, &l.IsValid, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		l.EmployeeID = l.UserID
		logs = append(logs, &l)
	}
	return logs, nil
}

func (r *attendanceRepository) GetByEmployeeID(ctx context.Context, employeeID uuid.UUID, limit int) ([]*model.AttendanceLog, error) {
	return r.GetByUserID(ctx, employeeID, limit)
}

func (r *attendanceRepository) ListLogs(ctx context.Context, tenantID uuid.UUID, limit, offset int, startDate, endDate *time.Time, userID *uuid.UUID, isValid *bool) ([]*model.AttendanceLog, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	whereClauses := []string{"($1 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $1)"}
	args := []interface{}{tenantID}
	argIdx := 2

	if startDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("clock_time >= $%d", argIdx))
		args = append(args, *startDate)
		argIdx++
	}

	if endDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("clock_time <= $%d", argIdx))
		args = append(args, *endDate)
		argIdx++
	}

	if userID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *userID)
		argIdx++
	}

	if isValid != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_valid = $%d", argIdx))
		args = append(args, *isValid)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM attendance_logs WHERE %s", whereSQL)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, user_id, clock_time, attendance_type, similarity_score,
		       latitude, longitude, distance_meters, device_id, photo_url, is_valid, created_at
		FROM attendance_logs
		WHERE %s
		ORDER BY clock_time DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*model.AttendanceLog
	for rows.Next() {
		var l model.AttendanceLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.UserID, &l.ClockTime, &l.AttendanceType, &l.SimilarityScore,
			&l.Latitude, &l.Longitude, &l.DistanceMeters, &l.DeviceID, &l.PhotoURL, &l.IsValid, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		l.EmployeeID = l.UserID
		logs = append(logs, &l)
	}

	return logs, total, nil
}

func (r *attendanceRepository) GetSummary(ctx context.Context, tenantID uuid.UUID, targetDate time.Time) (*AttendanceSummary, error) {
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var totalUsers int
	userCountQuery := `
		SELECT COUNT(*) 
		FROM users 
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $1) AND role = 'user' AND is_active = true
	`
	if err := r.db.QueryRow(ctx, userCountQuery, tenantID).Scan(&totalUsers); err != nil {
		return nil, err
	}

	summaryQuery := `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE is_valid = true),
			COUNT(*) FILTER (WHERE is_valid = false)
		FROM attendance_logs
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR tenant_id = $1)
		  AND clock_time >= $2 AND clock_time < $3
	`
	var totalLogs, validLogs, invalidLogs int
	if err := r.db.QueryRow(ctx, summaryQuery, tenantID, startOfDay, endOfDay).Scan(
		&totalLogs, &validLogs, &invalidLogs,
	); err != nil {
		return nil, err
	}

	return &AttendanceSummary{
		Date:            targetDate.Format("2006-01-02"),
		TotalUsers:      totalUsers,
		TotalCheckIns:   totalLogs,
		ValidCheckIns:   validLogs,
		InvalidCheckIns: invalidLogs,
	}, nil
}
