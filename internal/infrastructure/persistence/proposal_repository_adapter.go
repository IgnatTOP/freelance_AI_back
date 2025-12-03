package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/valueobject"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
	"github.com/jmoiron/sqlx"
)

type ProposalRepositoryAdapter struct {
	db *sqlx.DB
}

func NewProposalRepositoryAdapter(db *sqlx.DB) *ProposalRepositoryAdapter {
	return &ProposalRepositoryAdapter{db: db}
}

func (r *ProposalRepositoryAdapter) Create(ctx context.Context, proposal *entity.Proposal) error {
	query := `
		INSERT INTO proposals (id, order_id, freelancer_id, cover_letter, proposed_budget, proposed_deadline, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		proposal.ID, proposal.OrderID, proposal.FreelancerID, proposal.CoverLetter,
		proposal.ProposedBudget, proposal.ProposedDeadline, string(proposal.Status),
		proposal.CreatedAt, proposal.UpdatedAt,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать предложение")
	}
	return nil
}

func (r *ProposalRepositoryAdapter) Update(ctx context.Context, proposal *entity.Proposal) error {
	query := `
		UPDATE proposals SET cover_letter = $2, proposed_budget = $3, proposed_deadline = $4,
		status = $5, ai_feedback = $6, ai_analysis_for_client = $7, ai_analysis_for_client_at = $8,
		completed_by_freelancer_at = $9, updated_at = $10
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		proposal.ID, proposal.CoverLetter, proposal.ProposedBudget, proposal.ProposedDeadline,
		string(proposal.Status), proposal.AIFeedback, proposal.AIAnalysisForClient,
		proposal.AIAnalysisForClientAt, proposal.CompletedByFreelancerAt, proposal.UpdatedAt,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось обновить предложение")
	}
	return nil
}

func (r *ProposalRepositoryAdapter) FindByID(ctx context.Context, id uuid.UUID) (*entity.Proposal, error) {
	var p proposalRow
	query := `
		SELECT id, order_id, freelancer_id, cover_letter, proposed_budget, proposed_deadline,
		status, ai_feedback, ai_analysis_for_client, ai_analysis_for_client_at,
		completed_by_freelancer_at, created_at, updated_at
		FROM proposals WHERE id = $1
	`
	if err := r.db.GetContext(ctx, &p, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.ErrProposalNotFound
		}
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить предложение")
	}
	return p.toEntity(), nil
}

func (r *ProposalRepositoryAdapter) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.Proposal, error) {
	var rows []proposalRow
	query := `
		SELECT id, order_id, freelancer_id, cover_letter, proposed_budget, proposed_deadline,
		status, ai_feedback, ai_analysis_for_client, ai_analysis_for_client_at,
		completed_by_freelancer_at, created_at, updated_at
		FROM proposals WHERE order_id = $1 ORDER BY created_at DESC
	`
	if err := r.db.SelectContext(ctx, &rows, query, orderID); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить предложения")
	}
	return toProposalEntities(rows), nil
}

func (r *ProposalRepositoryAdapter) FindByFreelancerID(ctx context.Context, freelancerID uuid.UUID) ([]*entity.Proposal, error) {
	var rows []proposalRow
	query := `
		SELECT id, order_id, freelancer_id, cover_letter, proposed_budget, proposed_deadline,
		status, ai_feedback, ai_analysis_for_client, ai_analysis_for_client_at,
		completed_by_freelancer_at, created_at, updated_at
		FROM proposals WHERE freelancer_id = $1 ORDER BY created_at DESC
	`
	if err := r.db.SelectContext(ctx, &rows, query, freelancerID); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить предложения")
	}
	return toProposalEntities(rows), nil
}

func (r *ProposalRepositoryAdapter) FindByOrderAndFreelancer(ctx context.Context, orderID, freelancerID uuid.UUID) (*entity.Proposal, error) {
	var p proposalRow
	query := `
		SELECT id, order_id, freelancer_id, cover_letter, proposed_budget, proposed_deadline,
		status, ai_feedback, ai_analysis_for_client, ai_analysis_for_client_at,
		completed_by_freelancer_at, created_at, updated_at
		FROM proposals WHERE order_id = $1 AND freelancer_id = $2
	`
	if err := r.db.GetContext(ctx, &p, query, orderID, freelancerID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить предложение")
	}
	return p.toEntity(), nil
}

func (r *ProposalRepositoryAdapter) GetLastUpdateTime(ctx context.Context, orderID uuid.UUID) (*time.Time, error) {
	var t *time.Time
	query := `SELECT MAX(updated_at) FROM proposals WHERE order_id = $1`
	if err := r.db.GetContext(ctx, &t, query, orderID); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить время обновления")
	}
	return t, nil
}

type proposalRow struct {
	ID                      uuid.UUID  `db:"id"`
	OrderID                 uuid.UUID  `db:"order_id"`
	FreelancerID            uuid.UUID  `db:"freelancer_id"`
	CoverLetter             string     `db:"cover_letter"`
	ProposedBudget          float64    `db:"proposed_budget"`
	ProposedDeadline        *time.Time `db:"proposed_deadline"`
	Status                  string     `db:"status"`
	AIFeedback              *string    `db:"ai_feedback"`
	AIAnalysisForClient     *string    `db:"ai_analysis_for_client"`
	AIAnalysisForClientAt   *time.Time `db:"ai_analysis_for_client_at"`
	CompletedByFreelancerAt *time.Time `db:"completed_by_freelancer_at"`
	CreatedAt               time.Time  `db:"created_at"`
	UpdatedAt               time.Time  `db:"updated_at"`
}

func (p *proposalRow) toEntity() *entity.Proposal {
	status, _ := valueobject.NewProposalStatus(p.Status)
	return &entity.Proposal{
		ID:                      p.ID,
		OrderID:                 p.OrderID,
		FreelancerID:            p.FreelancerID,
		CoverLetter:             p.CoverLetter,
		ProposedBudget:          p.ProposedBudget,
		ProposedDeadline:        p.ProposedDeadline,
		Status:                  status,
		AIFeedback:              p.AIFeedback,
		AIAnalysisForClient:     p.AIAnalysisForClient,
		AIAnalysisForClientAt:   p.AIAnalysisForClientAt,
		CompletedByFreelancerAt: p.CompletedByFreelancerAt,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
	}
}

func toProposalEntities(rows []proposalRow) []*entity.Proposal {
	result := make([]*entity.Proposal, len(rows))
	for i, row := range rows {
		result[i] = row.toEntity()
	}
	return result
}
