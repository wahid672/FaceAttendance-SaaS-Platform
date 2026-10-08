package repository

import (
	"context"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttendanceRepository interface {
	Create(ctx context.Context, log *model.AttendanceLog) error
	GetByEmployeeID(ctx context.Context, employeeID uuid.UUID, limit int) ([]*model.AttendanceLog, error)
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
			tenant_id, employee_id, clock_time, attendance_type, similarity_score,
			latitude, longitude, distance_meters, device_id, photo_url, is_valid
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) RETURNING id, created_at
	`
	return r.db.QueryRow(
		ctx, query,
		log.TenantID, log.EmployeeID, log.ClockTime, log.AttendanceType, log.SimilarityScore,
		log.Latitude, log.Longitude, log.DistanceMeters, log.DeviceID, log.PhotoURL, log.IsValid,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *attendanceRepository) GetByEmployeeID(ctx context.Context, employeeID uuid.UUID, limit int) ([]*model.AttendanceLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, tenant_id, employee_id, clock_time, attendance_type, similarity_score,
		       latitude, longitude, distance_meters, device_id, photo_url, is_valid, created_at
		FROM attendance_logs
		WHERE employee_id = $1
		ORDER BY clock_time DESC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, employeeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.AttendanceLog
	for rows.Next() {
		var l model.AttendanceLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.EmployeeID, &l.ClockTime, &l.AttendanceType, &l.SimilarityScore,
			&l.Latitude, &l.Longitude, &l.DistanceMeters, &l.DeviceID, &l.PhotoURL, &l.IsValid, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, nil
}
