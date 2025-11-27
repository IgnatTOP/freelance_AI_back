package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// PortfolioRepository описывает взаимодействие сервиса с хранилищем портфолио.
type PortfolioRepository interface {
	Create(ctx context.Context, item *models.PortfolioItem, mediaIDs []uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PortfolioItem, error)
	List(ctx context.Context, userID uuid.UUID) ([]models.PortfolioItem, error)
	Update(ctx context.Context, item *models.PortfolioItem, mediaIDs []uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ListMedia(ctx context.Context, portfolioID uuid.UUID) ([]models.MediaFile, error)
}

// PortfolioService содержит бизнес-логику работы с портфолио.
type PortfolioService struct {
	repo PortfolioRepository
}

// NewPortfolioService создаёт новый сервис портфолио.
func NewPortfolioService(repo PortfolioRepository) *PortfolioService {
	return &PortfolioService{repo: repo}
}

// CreatePortfolioItem создаёт новую работу в портфолио.
func (s *PortfolioService) CreatePortfolioItem(ctx context.Context, userID uuid.UUID, title string, description *string, coverMediaID *uuid.UUID, aiTags []string, externalLink *string, mediaIDs []uuid.UUID) (*models.PortfolioItem, error) {
	if title == "" {
		return nil, fmt.Errorf("portfolio service: заголовок работы не может быть пустым")
	}

	item := &models.PortfolioItem{
		UserID:       userID,
		Title:        title,
		Description:  description,
		CoverMediaID: coverMediaID,
		AITags:       aiTags,
		ExternalLink: externalLink,
	}

	if err := s.repo.Create(ctx, item, mediaIDs); err != nil {
		return nil, err
	}

	return item, nil
}

// GetPortfolioItem возвращает работу по идентификатору.
func (s *PortfolioService) GetPortfolioItem(ctx context.Context, id uuid.UUID) (*models.PortfolioItem, error) {
	return s.repo.GetByID(ctx, id)
}

// ListPortfolioItems возвращает список работ пользователя.
func (s *PortfolioService) ListPortfolioItems(ctx context.Context, userID uuid.UUID) ([]models.PortfolioItem, error) {
	return s.repo.List(ctx, userID)
}

// UpdatePortfolioItem обновляет работу в портфолио.
func (s *PortfolioService) UpdatePortfolioItem(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description *string, coverMediaID *uuid.UUID, aiTags []string, externalLink *string, mediaIDs []uuid.UUID) (*models.PortfolioItem, error) {
	if title == "" {
		return nil, fmt.Errorf("portfolio service: заголовок работы не может быть пустым")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.UserID != userID {
		return nil, fmt.Errorf("portfolio service: у вас нет прав на изменение этой работы")
	}

	existing.Title = title
	existing.Description = description
	existing.CoverMediaID = coverMediaID
	existing.AITags = aiTags
	existing.ExternalLink = externalLink

	if err := s.repo.Update(ctx, existing, mediaIDs); err != nil {
		return nil, err
	}

	return existing, nil
}

// DeletePortfolioItem удаляет работу из портфолио.
func (s *PortfolioService) DeletePortfolioItem(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.UserID != userID {
		return fmt.Errorf("portfolio service: у вас нет прав на удаление этой работы")
	}

	return s.repo.Delete(ctx, id, userID)
}

// ListPortfolioMedia возвращает список медиа для работы.
func (s *PortfolioService) ListPortfolioMedia(ctx context.Context, portfolioID uuid.UUID) ([]models.MediaFile, error) {
	return s.repo.ListMedia(ctx, portfolioID)
}

