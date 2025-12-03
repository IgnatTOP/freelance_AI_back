package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ReportStatusPending     = "pending"
	ReportStatusReviewed    = "reviewed"
	ReportStatusActionTaken = "action_taken"
	ReportStatusDismissed   = "dismissed"

	ReportTargetUser    = "user"
	ReportTargetOrder   = "order"
	ReportTargetMessage = "message"
	ReportTargetReview  = "review"
)

type Report struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	ReporterID  uuid.UUID  `db:"reporter_id" json:"reporter_id"`
	TargetType  string     `db:"target_type" json:"target_type"`
	TargetID    uuid.UUID  `db:"target_id" json:"target_id"`
	Reason      string     `db:"reason" json:"reason"`
	Description *string    `db:"description" json:"description,omitempty"`
	Status      string     `db:"status" json:"status"`
	ReviewedBy  *uuid.UUID `db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time `db:"reviewed_at" json:"reviewed_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}
