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

var ErrWithdrawalNotFound = errors.New("withdrawal not found")

type WithdrawalRepository struct {
	db *sqlx.DB
}

func NewWithdrawalRepository(db *sqlx.DB) *WithdrawalRepository {
	return &WithdrawalRepository{db: db}
}

func (r *WithdrawalRepository) Create(ctx context.Context, userID uuid.UUID, amount float64, cardLast4, bankName string) (*models.Withdrawal, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Проверяем и списываем баланс
	var available float64
	err = tx.GetContext(ctx, &available, `SELECT available FROM user_balances WHERE user_id = $1 FOR UPDATE`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInsufficientFunds
		}
		return nil, err
	}
	if available < amount {
		return nil, ErrInsufficientFunds
	}

	_, err = tx.ExecContext(ctx, `UPDATE user_balances SET available = available - $2, updated_at = NOW() WHERE user_id = $1`, userID, amount)
	if err != nil {
		return nil, err
	}

	var w models.Withdrawal
	err = tx.GetContext(ctx, &w, `
		INSERT INTO withdrawals (user_id, amount, card_last4, bank_name)
		VALUES ($1, $2, $3, $4)
		RETURNING *
	`, userID, amount, cardLast4, bankName)
	if err != nil {
		return nil, err
	}

	return &w, tx.Commit()
}

func (r *WithdrawalRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Withdrawal, error) {
	var w models.Withdrawal
	err := r.db.GetContext(ctx, &w, `SELECT * FROM withdrawals WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWithdrawalNotFound
	}
	return &w, err
}

func (r *WithdrawalRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	err := r.db.SelectContext(ctx, &withdrawals, `
		SELECT * FROM withdrawals WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	return withdrawals, err
}

func (r *WithdrawalRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, rejectionReason *string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE withdrawals SET status = $2, rejection_reason = $3, processed_at = $4 WHERE id = $1
	`, id, status, rejectionReason, now)
	return err
}
