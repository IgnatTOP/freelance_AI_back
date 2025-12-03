package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrEscrowNotFound    = errors.New("escrow not found")
)

type PaymentRepository struct {
	db *sqlx.DB
}

func NewPaymentRepository(db *sqlx.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// GetBalance возвращает баланс пользователя, создаёт если не существует.
func (r *PaymentRepository) GetBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalance, error) {
	var balance models.UserBalance
	query := `
		INSERT INTO user_balances (user_id, available, frozen)
		VALUES ($1, 0, 0)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = NOW()
		RETURNING user_id, available, frozen, updated_at
	`
	if err := r.db.GetContext(ctx, &balance, query, userID); err != nil {
		return nil, fmt.Errorf("payment repository: get balance %w", err)
	}
	return &balance, nil
}

// Deposit пополняет баланс пользователя.
func (r *PaymentRepository) Deposit(ctx context.Context, userID uuid.UUID, amount float64, description string) (*models.Transaction, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Обновляем баланс
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_balances (user_id, available, frozen)
		VALUES ($1, $2, 0)
		ON CONFLICT (user_id) DO UPDATE SET available = user_balances.available + $2, updated_at = NOW()
	`, userID, amount)
	if err != nil {
		return nil, fmt.Errorf("payment repository: deposit update balance %w", err)
	}

	// Создаём транзакцию
	var transaction models.Transaction
	err = tx.GetContext(ctx, &transaction, `
		INSERT INTO transactions (user_id, type, amount, status, description, completed_at)
		VALUES ($1, 'deposit', $2, 'completed', $3, NOW())
		RETURNING id, user_id, order_id, type, amount, status, description, created_at, completed_at
	`, userID, amount, description)
	if err != nil {
		return nil, fmt.Errorf("payment repository: deposit create transaction %w", err)
	}

	return &transaction, tx.Commit()
}

// CreateEscrow создаёт escrow и замораживает средства клиента.
func (r *PaymentRepository) CreateEscrow(ctx context.Context, orderID, clientID, freelancerID uuid.UUID, amount float64) (*models.Escrow, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Проверяем баланс клиента
	var balance models.UserBalance
	err = tx.GetContext(ctx, &balance, `SELECT user_id, available, frozen FROM user_balances WHERE user_id = $1 FOR UPDATE`, clientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInsufficientFunds
		}
		return nil, err
	}
	if balance.Available < amount {
		return nil, ErrInsufficientFunds
	}

	// Замораживаем средства
	_, err = tx.ExecContext(ctx, `
		UPDATE user_balances SET available = available - $2, frozen = frozen + $2, updated_at = NOW()
		WHERE user_id = $1
	`, clientID, amount)
	if err != nil {
		return nil, err
	}

	// Создаём escrow
	var escrow models.Escrow
	err = tx.GetContext(ctx, &escrow, `
		INSERT INTO escrow (order_id, client_id, freelancer_id, amount, status)
		VALUES ($1, $2, $3, $4, 'held')
		RETURNING id, order_id, client_id, freelancer_id, amount, status, created_at, released_at
	`, orderID, clientID, freelancerID, amount)
	if err != nil {
		return nil, err
	}

	// Транзакция заморозки
	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (user_id, order_id, type, amount, status, description, completed_at)
		VALUES ($1, $2, 'escrow_hold', $3, 'completed', 'Заморозка средств для заказа', NOW())
	`, clientID, orderID, amount)
	if err != nil {
		return nil, err
	}

	return &escrow, tx.Commit()
}

// ReleaseEscrow освобождает средства в пользу фрилансера.
func (r *PaymentRepository) ReleaseEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var escrow models.Escrow
	err = tx.GetContext(ctx, &escrow, `SELECT * FROM escrow WHERE order_id = $1 AND status = 'held' FOR UPDATE`, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEscrowNotFound
		}
		return nil, err
	}

	// Снимаем заморозку у клиента
	_, err = tx.ExecContext(ctx, `
		UPDATE user_balances SET frozen = frozen - $2, updated_at = NOW()
		WHERE user_id = $1
	`, escrow.ClientID, escrow.Amount)
	if err != nil {
		return nil, err
	}

	// Начисляем фрилансеру
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_balances (user_id, available, frozen)
		VALUES ($1, $2, 0)
		ON CONFLICT (user_id) DO UPDATE SET available = user_balances.available + $2, updated_at = NOW()
	`, escrow.FreelancerID, escrow.Amount)
	if err != nil {
		return nil, err
	}

	// Обновляем escrow
	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE escrow SET status = 'released', released_at = $2 WHERE id = $1`, escrow.ID, now)
	if err != nil {
		return nil, err
	}
	escrow.Status = models.EscrowStatusReleased
	escrow.ReleasedAt = &now

	// Транзакция освобождения
	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (user_id, order_id, type, amount, status, description, completed_at)
		VALUES ($1, $2, 'escrow_release', $3, 'completed', 'Получение оплаты за заказ', NOW())
	`, escrow.FreelancerID, orderID, escrow.Amount)
	if err != nil {
		return nil, err
	}

	return &escrow, tx.Commit()
}

// RefundEscrow возвращает средства клиенту.
func (r *PaymentRepository) RefundEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var escrow models.Escrow
	err = tx.GetContext(ctx, &escrow, `SELECT * FROM escrow WHERE order_id = $1 AND status = 'held' FOR UPDATE`, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEscrowNotFound
		}
		return nil, err
	}

	// Возвращаем средства клиенту
	_, err = tx.ExecContext(ctx, `
		UPDATE user_balances SET available = available + $2, frozen = frozen - $2, updated_at = NOW()
		WHERE user_id = $1
	`, escrow.ClientID, escrow.Amount)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE escrow SET status = 'refunded', released_at = $2 WHERE id = $1`, escrow.ID, now)
	if err != nil {
		return nil, err
	}
	escrow.Status = models.EscrowStatusRefunded
	escrow.ReleasedAt = &now

	// Транзакция возврата
	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (user_id, order_id, type, amount, status, description, completed_at)
		VALUES ($1, $2, 'escrow_refund', $3, 'completed', 'Возврат средств за отменённый заказ', NOW())
	`, escrow.ClientID, orderID, escrow.Amount)
	if err != nil {
		return nil, err
	}

	return &escrow, tx.Commit()
}

// GetEscrowByOrderID возвращает escrow по ID заказа.
func (r *PaymentRepository) GetEscrowByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	var escrow models.Escrow
	err := r.db.GetContext(ctx, &escrow, `SELECT * FROM escrow WHERE order_id = $1`, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEscrowNotFound
		}
		return nil, err
	}
	return &escrow, nil
}

// ListTransactions возвращает историю транзакций пользователя.
func (r *PaymentRepository) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := r.db.SelectContext(ctx, &transactions, `
		SELECT id, user_id, order_id, type, amount, status, description, created_at, completed_at
		FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	return transactions, err
}
