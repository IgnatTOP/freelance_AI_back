package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

var ErrReportNotFound = errors.New("report not found")

type ReportRepository struct {
	db *sqlx.DB
}

func NewReportRepository(db *sqlx.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, report *models.Report) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO reports (reporter_id, target_type, target_id, reason, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at
	`, report.ReporterID, report.TargetType, report.TargetID, report.Reason, report.Description).
		Scan(&report.ID, &report.Status, &report.CreatedAt)
}

func (r *ReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Report, error) {
	var report models.Report
	err := r.db.GetContext(ctx, &report, `SELECT * FROM reports WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrReportNotFound
	}
	return &report, err
}

func (r *ReportRepository) ListByReporter(ctx context.Context, reporterID uuid.UUID, limit, offset int) ([]models.Report, error) {
	var reports []models.Report
	err := r.db.SelectContext(ctx, &reports, `
		SELECT * FROM reports WHERE reporter_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, reporterID, limit, offset)
	return reports, err
}

func (r *ReportRepository) ListPending(ctx context.Context, limit, offset int) ([]models.Report, error) {
	var reports []models.Report
	err := r.db.SelectContext(ctx, &reports, `
		SELECT * FROM reports WHERE status = 'pending' ORDER BY created_at ASC LIMIT $1 OFFSET $2
	`, limit, offset)
	return reports, err
}
