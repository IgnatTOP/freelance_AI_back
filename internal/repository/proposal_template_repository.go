package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

var ErrTemplateNotFound = errors.New("template not found")

type ProposalTemplateRepository struct {
	db *sqlx.DB
}

func NewProposalTemplateRepository(db *sqlx.DB) *ProposalTemplateRepository {
	return &ProposalTemplateRepository{db: db}
}

func (r *ProposalTemplateRepository) Create(ctx context.Context, t *models.ProposalTemplate) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO proposal_templates (user_id, title, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, t.UserID, t.Title, t.Content).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *ProposalTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ProposalTemplate, error) {
	var t models.ProposalTemplate
	err := r.db.GetContext(ctx, &t, `SELECT * FROM proposal_templates WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	return &t, err
}

func (r *ProposalTemplateRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.ProposalTemplate, error) {
	var templates []models.ProposalTemplate
	err := r.db.SelectContext(ctx, &templates, `
		SELECT * FROM proposal_templates WHERE user_id = $1 ORDER BY updated_at DESC
	`, userID)
	return templates, err
}

func (r *ProposalTemplateRepository) Update(ctx context.Context, id uuid.UUID, title, content string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE proposal_templates SET title = $2, content = $3, updated_at = NOW() WHERE id = $1
	`, id, title, content)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (r *ProposalTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM proposal_templates WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrTemplateNotFound
	}
	return nil
}
