package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

type mockReviewRepo struct {
	mock.Mock
}

func (m *mockReviewRepo) Create(ctx context.Context, review *models.Review) error {
	args := m.Called(ctx, review)
	if args.Error(0) == nil {
		review.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *mockReviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Review, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Review), args.Error(1)
}

func (m *mockReviewRepo) GetByOrderAndReviewer(ctx context.Context, orderID, reviewerID uuid.UUID) (*models.Review, error) {
	args := m.Called(ctx, orderID, reviewerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Review), args.Error(1)
}

func (m *mockReviewRepo) ListByReviewedID(ctx context.Context, reviewedID uuid.UUID, limit, offset int) ([]models.Review, error) {
	args := m.Called(ctx, reviewedID, limit, offset)
	return args.Get(0).([]models.Review), args.Error(1)
}

func (m *mockReviewRepo) ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.Review, error) {
	args := m.Called(ctx, orderID)
	return args.Get(0).([]models.Review), args.Error(1)
}

func (m *mockReviewRepo) GetAverageRating(ctx context.Context, userID uuid.UUID) (float64, int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Int(1), args.Error(2)
}

func (m *mockReviewRepo) Update(ctx context.Context, review *models.Review) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *mockReviewRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockOrderRepoForReview struct {
	mock.Mock
}

func (m *mockOrderRepoForReview) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func TestReviewService_CreateReview_Success(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	clientID := uuid.New()
	freelancerID := uuid.New()
	orderID := uuid.New()

	order := &models.Order{
		ID:           orderID,
		ClientID:     clientID,
		FreelancerID: &freelancerID,
		Status:       models.OrderStatusCompleted,
	}

	orderRepo.On("GetByID", ctx, orderID).Return(order, nil)
	reviewRepo.On("GetByOrderAndReviewer", ctx, orderID, clientID).Return(nil, nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*models.Review")).Return(nil)

	comment := "Отличная работа!"
	review, err := svc.CreateReview(ctx, orderID, clientID, 5, &comment)

	assert.NoError(t, err)
	assert.NotNil(t, review)
	assert.Equal(t, freelancerID, review.ReviewedID)
	assert.Equal(t, 5, review.Rating)
}

func TestReviewService_CreateReview_InvalidRating(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	_, err := svc.CreateReview(ctx, uuid.New(), uuid.New(), 0, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "от 1 до 5")

	_, err = svc.CreateReview(ctx, uuid.New(), uuid.New(), 6, nil)
	assert.Error(t, err)
}

func TestReviewService_CreateReview_OrderNotCompleted(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	orderID := uuid.New()
	order := &models.Order{ID: orderID, Status: models.OrderStatusInProgress}
	orderRepo.On("GetByID", ctx, orderID).Return(order, nil)

	_, err := svc.CreateReview(ctx, orderID, uuid.New(), 5, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "после завершения")
}

func TestReviewService_CreateReview_AlreadyReviewed(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	clientID := uuid.New()
	freelancerID := uuid.New()
	orderID := uuid.New()

	order := &models.Order{
		ID:           orderID,
		ClientID:     clientID,
		FreelancerID: &freelancerID,
		Status:       models.OrderStatusCompleted,
	}

	existingReview := &models.Review{ID: uuid.New()}

	orderRepo.On("GetByID", ctx, orderID).Return(order, nil)
	reviewRepo.On("GetByOrderAndReviewer", ctx, orderID, clientID).Return(existingReview, nil)

	_, err := svc.CreateReview(ctx, orderID, clientID, 5, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "уже оставили")
}

func TestReviewService_CreateReview_NotParticipant(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	clientID := uuid.New()
	freelancerID := uuid.New()
	orderID := uuid.New()
	randomUserID := uuid.New()

	order := &models.Order{
		ID:           orderID,
		ClientID:     clientID,
		FreelancerID: &freelancerID,
		Status:       models.OrderStatusCompleted,
	}

	orderRepo.On("GetByID", ctx, orderID).Return(order, nil)

	_, err := svc.CreateReview(ctx, orderID, randomUserID, 5, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "не участник")
}

func TestReviewService_ListUserReviews(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	userID := uuid.New()
	expected := []models.Review{{ID: uuid.New()}, {ID: uuid.New()}}
	reviewRepo.On("ListByReviewedID", ctx, userID, 20, 0).Return(expected, nil)

	reviews, err := svc.ListUserReviews(ctx, userID, 20, 0)
	assert.NoError(t, err)
	assert.Len(t, reviews, 2)
}

func TestReviewService_ListOrderReviews(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	orderID := uuid.New()
	expected := []models.Review{{ID: uuid.New()}}
	reviewRepo.On("ListByOrderID", ctx, orderID).Return(expected, nil)

	reviews, err := svc.ListOrderReviews(ctx, orderID)
	assert.NoError(t, err)
	assert.Len(t, reviews, 1)
}

func TestReviewService_GetUserRating(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	userID := uuid.New()
	reviewRepo.On("GetAverageRating", ctx, userID).Return(4.5, 10, nil)

	avg, count, err := svc.GetUserRating(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 4.5, avg)
	assert.Equal(t, 10, count)
}

func TestReviewService_CanLeaveReview_True(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	clientID := uuid.New()
	freelancerID := uuid.New()
	orderID := uuid.New()

	order := &models.Order{
		ID:           orderID,
		ClientID:     clientID,
		FreelancerID: &freelancerID,
		Status:       models.OrderStatusCompleted,
	}

	orderRepo.On("GetByID", ctx, orderID).Return(order, nil)
	reviewRepo.On("GetByOrderAndReviewer", ctx, orderID, clientID).Return(nil, nil)

	canReview, err := svc.CanLeaveReview(ctx, orderID, clientID)
	assert.NoError(t, err)
	assert.True(t, canReview)
}

func TestReviewService_CanLeaveReview_False_AlreadyReviewed(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	clientID := uuid.New()
	freelancerID := uuid.New()
	orderID := uuid.New()

	order := &models.Order{
		ID:           orderID,
		ClientID:     clientID,
		FreelancerID: &freelancerID,
		Status:       models.OrderStatusCompleted,
	}

	existingReview := &models.Review{ID: uuid.New()}

	orderRepo.On("GetByID", ctx, orderID).Return(order, nil)
	reviewRepo.On("GetByOrderAndReviewer", ctx, orderID, clientID).Return(existingReview, nil)

	canReview, err := svc.CanLeaveReview(ctx, orderID, clientID)
	assert.NoError(t, err)
	assert.False(t, canReview)
}

func TestReviewService_CanLeaveReview_False_NotCompleted(t *testing.T) {
	reviewRepo := new(mockReviewRepo)
	orderRepo := new(mockOrderRepoForReview)
	svc := NewReviewService(reviewRepo, orderRepo)
	ctx := context.Background()

	orderID := uuid.New()
	order := &models.Order{ID: orderID, Status: models.OrderStatusInProgress}
	orderRepo.On("GetByID", ctx, orderID).Return(order, nil)

	canReview, err := svc.CanLeaveReview(ctx, orderID, uuid.New())
	assert.NoError(t, err)
	assert.False(t, canReview)
}
