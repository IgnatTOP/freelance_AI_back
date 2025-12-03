package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	WithdrawalStatusPending    = "pending"
	WithdrawalStatusProcessing = "processing"
	WithdrawalStatusCompleted  = "completed"
	WithdrawalStatusRejected   = "rejected"
)

type Withdrawal struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	UserID          uuid.UUID  `db:"user_id" json:"user_id"`
	Amount          float64    `db:"amount" json:"amount"`
	Status          string     `db:"status" json:"status"`
	CardLast4       *string    `db:"card_last4" json:"card_last4,omitempty"`
	BankName        *string    `db:"bank_name" json:"bank_name,omitempty"`
	RejectionReason *string    `db:"rejection_reason" json:"rejection_reason,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	ProcessedAt     *time.Time `db:"processed_at" json:"processed_at,omitempty"`
}
