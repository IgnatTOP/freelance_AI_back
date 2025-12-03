package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

var ErrDisputeNotFound = errors.New("dispute not found")

type DisputeRepository struct {
	db *sqlx.DB
}

func NewDisputeRepository(db *sqlx.DB) *DisputeRepository {
	return &DisputeRepository{db: db}
}

func (r *DisputeRepository) Create(ctx context.Context, d *models.Dispute) error {
	query := `
		INSERT INTO disputes (escrow_id, order_id, initiator_id, reason, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query, d.EscrowID, d.OrderID, d.InitiatorID, d.Reason, d.Status).
		Scan(&d.ID, &d.CreatedAt)
}

func (r *DisputeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Dispute, error) {
	var d models.Dispute
	err := r.db.GetContext(ctx, &d, `SELECT * FROM disputes WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDisputeNotFound
	}
	return &d, err
}

func (r *DisputeRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Dispute, error) {
	var d models.Dispute
	err := r.db.GetContext(ctx, &d, `SELECT * FROM disputes WHERE order_id = $1`, orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDisputeNotFound
	}
	return &d, err
}

func (r *DisputeRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, resolution string, resolvedBy *uuid.UUID) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE disputes SET status = $2, resolution = $3, resolved_by = $4, resolved_at = $5 WHERE id = $1
	`, id, status, resolution, resolvedBy, now)
	return err
}

func (r *DisputeRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Dispute, error) {
	var disputes []models.Dispute
	err := r.db.SelectContext(ctx, &disputes, `
		SELECT d.* FROM disputes d
		JOIN escrow e ON d.escrow_id = e.id
		WHERE e.client_id = $1 OR e.freelancer_id = $1
		ORDER BY d.created_at DESC LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	return disputes, err
}
