package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

var ErrTemplateNotOwned = errors.New("template does not belong to user")

type ProposalTemplateService struct {
	repo *repository.ProposalTemplateRepository
}

func NewProposalTemplateService(r *repository.ProposalTemplateRepository) *ProposalTemplateService {
	return &ProposalTemplateService{repo: r}
}

func (s *ProposalTemplateService) Create(ctx context.Context, userID uuid.UUID, title, content string) (*models.ProposalTemplate, error) {
	t := &models.ProposalTemplate{UserID: userID, Title: title, Content: content}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *ProposalTemplateService) List(ctx context.Context, userID uuid.UUID) ([]models.ProposalTemplate, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *ProposalTemplateService) Update(ctx context.Context, userID, templateID uuid.UUID, title, content string) error {
	t, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return err
	}
	if t.UserID != userID {
		return ErrTemplateNotOwned
	}
	return s.repo.Update(ctx, templateID, title, content)
}

func (s *ProposalTemplateService) Delete(ctx context.Context, userID, templateID uuid.UUID) error {
	t, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return err
	}
	if t.UserID != userID {
		return ErrTemplateNotOwned
	}
	return s.repo.Delete(ctx, templateID)
}
