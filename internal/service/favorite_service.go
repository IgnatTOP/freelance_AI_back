package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

var ErrInvalidTargetType = errors.New("invalid target type, must be 'order' or 'freelancer'")

type FavoriteService struct {
	repo *repository.FavoriteRepository
}

func NewFavoriteService(r *repository.FavoriteRepository) *FavoriteService {
	return &FavoriteService{repo: r}
}

func (s *FavoriteService) AddFavorite(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) (*models.Favorite, error) {
	if targetType != models.FavoriteTypeOrder && targetType != models.FavoriteTypeFreelancer {
		return nil, ErrInvalidTargetType
	}
	return s.repo.Add(ctx, userID, targetType, targetID)
}

func (s *FavoriteService) RemoveFavorite(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) error {
	return s.repo.Remove(ctx, userID, targetType, targetID)
}

func (s *FavoriteService) ListFavorites(ctx context.Context, userID uuid.UUID, targetType string, limit, offset int) ([]models.Favorite, error) {
	return s.repo.ListByUser(ctx, userID, targetType, limit, offset)
}

func (s *FavoriteService) IsFavorite(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) (bool, error) {
	return s.repo.Exists(ctx, userID, targetType, targetID)
}
