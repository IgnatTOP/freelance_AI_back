package models

import (
	"time"

	"github.com/google/uuid"
)

// Order описывает заказ на разработку или услугу.
type Order struct {
	ID                              uuid.UUID  `db:"id" json:"id"`
	ClientID                        uuid.UUID  `db:"client_id" json:"client_id"`
	Title                           string     `db:"title" json:"title"`
	Description                     string     `db:"description" json:"description"`
	BudgetMin                       *float64   `db:"budget_min" json:"budget_min,omitempty"`
	BudgetMax                       *float64   `db:"budget_max" json:"budget_max,omitempty"`
	Status                          string     `db:"status" json:"status"`
	DeadlineAt                      *time.Time `db:"deadline_at" json:"deadline_at,omitempty"`
	AISummary                       *string    `db:"ai_summary" json:"ai_summary,omitempty"`
	BestRecommendationProposalID    *uuid.UUID `db:"best_recommendation_proposal_id" json:"best_recommendation_proposal_id,omitempty"`
	BestRecommendationJustification *string    `db:"best_recommendation_justification" json:"best_recommendation_justification,omitempty"`
	AIAnalysisUpdatedAt             *time.Time `db:"ai_analysis_updated_at" json:"ai_analysis_updated_at,omitempty"`
	CreatedAt                       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt                       time.Time  `db:"updated_at" json:"updated_at"`
	Attachments                     []OrderAttachment `json:"attachments,omitempty"`
	ProposalsCount                  *int       `db:"proposals_count" json:"proposals_count,omitempty"`
}

// OrderRequirement хранит информацию о требуемых навыках.
type OrderRequirement struct {
	ID      uuid.UUID `db:"id" json:"id"`
	OrderID uuid.UUID `db:"order_id" json:"order_id"`
	Skill   string    `db:"skill" json:"skill"`
	Level   string    `db:"level" json:"level"`
}

// OrderAttachment описывает файл, прикреплённый к заказу.
type OrderAttachment struct {
	ID      uuid.UUID  `db:"id" json:"id"`
	OrderID uuid.UUID  `db:"order_id" json:"order_id"`
	MediaID uuid.UUID  `db:"media_id" json:"media_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Media   *MediaFile `json:"media,omitempty"`
}

// Proposal представляет отклик фрилансера на заказ.
type Proposal struct {
	ID             uuid.UUID `db:"id" json:"id"`
	OrderID        uuid.UUID `db:"order_id" json:"order_id"`
	FreelancerID   uuid.UUID `db:"freelancer_id" json:"freelancer_id"`
	CoverLetter    string    `db:"cover_letter" json:"cover_letter"`
	ProposedAmount *float64  `db:"proposed_amount" json:"proposed_amount,omitempty"`
	Status         string    `db:"status" json:"status"`
	AIFeedback     *string   `db:"ai_feedback" json:"ai_feedback,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}
