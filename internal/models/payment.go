package models

import (
	"time"

	"github.com/google/uuid"
)

// Статусы escrow
const (
	EscrowStatusHeld     = "held"
	EscrowStatusReleased = "released"
	EscrowStatusRefunded = "refunded"
	EscrowStatusDisputed = "disputed"
)

// Типы транзакций
const (
	TransactionTypeDeposit       = "deposit"
	TransactionTypeWithdrawal    = "withdrawal"
	TransactionTypeEscrowHold    = "escrow_hold"
	TransactionTypeEscrowRelease = "escrow_release"
	TransactionTypeEscrowRefund  = "escrow_refund"
)

// Статусы транзакций
const (
	TransactionStatusPending   = "pending"
	TransactionStatusCompleted = "completed"
	TransactionStatusFailed    = "failed"
	TransactionStatusCancelled = "cancelled"
)

// UserBalance представляет баланс пользователя.
type UserBalance struct {
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Available float64   `db:"available" json:"available"`
	Frozen    float64   `db:"frozen" json:"frozen"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Transaction представляет финансовую транзакцию.
type Transaction struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	UserID      uuid.UUID  `db:"user_id" json:"user_id"`
	OrderID     *uuid.UUID `db:"order_id" json:"order_id,omitempty"`
	Type        string     `db:"type" json:"type"`
	Amount      float64    `db:"amount" json:"amount"`
	Status      string     `db:"status" json:"status"`
	Description *string    `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
}

// Escrow представляет защищённую сделку.
type Escrow struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	OrderID      uuid.UUID  `db:"order_id" json:"order_id"`
	ClientID     uuid.UUID  `db:"client_id" json:"client_id"`
	FreelancerID uuid.UUID  `db:"freelancer_id" json:"freelancer_id"`
	Amount       float64    `db:"amount" json:"amount"`
	Status       string     `db:"status" json:"status"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	ReleasedAt   *time.Time `db:"released_at" json:"released_at,omitempty"`
}
