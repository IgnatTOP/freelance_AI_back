package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	DisputeStatusOpen             = "open"
	DisputeStatusUnderReview      = "under_review"
	DisputeStatusResolvedClient   = "resolved_client"
	DisputeStatusResolvedFreelancer = "resolved_freelancer"
	DisputeStatusCancelled        = "cancelled"
)

type Dispute struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	EscrowID    uuid.UUID  `db:"escrow_id" json:"escrow_id"`
	OrderID     uuid.UUID  `db:"order_id" json:"order_id"`
	InitiatorID uuid.UUID  `db:"initiator_id" json:"initiator_id"`
	Reason      string     `db:"reason" json:"reason"`
	Status      string     `db:"status" json:"status"`
	Resolution  *string    `db:"resolution" json:"resolution,omitempty"`
	ResolvedBy  *uuid.UUID `db:"resolved_by" json:"resolved_by,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	ResolvedAt  *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
}
