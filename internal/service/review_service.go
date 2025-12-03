package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

type ReviewRepository interface {
	Create(ctx context.Context, review *models.Review) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Review, error)
	GetByOrderAndReviewer(ctx context.Context, orderID, reviewerID uuid.UUID) (*models.Review, error)
	ListByReviewedID(ctx context.Context, reviewedID uuid.UUID, limit, offset int) ([]models.Review, error)
	ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.Review, error)
	GetAverageRating(ctx context.Context, userID uuid.UUID) (float64, int, error)
	Update(ctx context.Context, review *models.Review) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type OrderRepoForReview interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
}

type ReviewService struct {
	repo   ReviewRepository
	orders OrderRepoForReview
}

func NewReviewService(repo ReviewRepository, orders OrderRepoForReview) *ReviewService {
	return &ReviewService{repo: repo, orders: orders}
}

// CreateReview создаёт отзыв после завершения заказа.
func (s *ReviewService) CreateReview(ctx context.Context, orderID, reviewerID uuid.UUID, rating int, comment *string) (*models.Review, error) {
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("рейтинг должен быть от 1 до 5")
	}

	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("заказ не найден")
	}

	if order.Status != models.OrderStatusCompleted {
		return nil, fmt.Errorf("отзыв можно оставить только после завершения заказа")
	}

	// Определяем, кому оставляется отзыв
	var reviewedID uuid.UUID
	if reviewerID == order.ClientID {
		if order.FreelancerID == nil {
			return nil, fmt.Errorf("фрилансер не назначен на заказ")
		}
		reviewedID = *order.FreelancerID
	} else if order.FreelancerID != nil && reviewerID == *order.FreelancerID {
		reviewedID = order.ClientID
	} else {
		return nil, fmt.Errorf("вы не участник этого заказа")
	}

	// Проверяем, не оставлял ли уже отзыв
	existing, err := s.repo.GetByOrderAndReviewer(ctx, orderID, reviewerID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("вы уже оставили отзыв на этот заказ")
	}

	review := &models.Review{
		OrderID:    orderID,
		ReviewerID: reviewerID,
		ReviewedID: reviewedID,
		Rating:     rating,
		Comment:    comment,
	}

	if err := s.repo.Create(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

// GetReview возвращает отзыв по ID.
func (s *ReviewService) GetReview(ctx context.Context, id uuid.UUID) (*models.Review, error) {
	return s.repo.GetByID(ctx, id)
}

// ListUserReviews возвращает отзывы о пользователе.
func (s *ReviewService) ListUserReviews(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Review, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListByReviewedID(ctx, userID, limit, offset)
}

// ListOrderReviews возвращает отзывы по заказу.
func (s *ReviewService) ListOrderReviews(ctx context.Context, orderID uuid.UUID) ([]models.Review, error) {
	return s.repo.ListByOrderID(ctx, orderID)
}

// GetUserRating возвращает средний рейтинг и количество отзывов.
func (s *ReviewService) GetUserRating(ctx context.Context, userID uuid.UUID) (float64, int, error) {
	return s.repo.GetAverageRating(ctx, userID)
}

// CanLeaveReview проверяет, может ли пользователь оставить отзыв.
func (s *ReviewService) CanLeaveReview(ctx context.Context, orderID, userID uuid.UUID) (bool, error) {
	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return false, nil
	}
	if order.Status != models.OrderStatusCompleted {
		return false, nil
	}
	if userID != order.ClientID && (order.FreelancerID == nil || userID != *order.FreelancerID) {
		return false, nil
	}
	existing, err := s.repo.GetByOrderAndReviewer(ctx, orderID, userID)
	if err != nil {
		return false, err
	}
	return existing == nil, nil
}
