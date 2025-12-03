package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

type mockPaymentRepo struct {
	mock.Mock
}

func (m *mockPaymentRepo) GetBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalance, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserBalance), args.Error(1)
}

func (m *mockPaymentRepo) Deposit(ctx context.Context, userID uuid.UUID, amount float64, description string) (*models.Transaction, error) {
	args := m.Called(ctx, userID, amount, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *mockPaymentRepo) CreateEscrow(ctx context.Context, orderID, clientID, freelancerID uuid.UUID, amount float64) (*models.Escrow, error) {
	args := m.Called(ctx, orderID, clientID, freelancerID, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Escrow), args.Error(1)
}

func (m *mockPaymentRepo) ReleaseEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Escrow), args.Error(1)
}

func (m *mockPaymentRepo) RefundEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Escrow), args.Error(1)
}

func (m *mockPaymentRepo) GetEscrowByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Escrow), args.Error(1)
}

func (m *mockPaymentRepo) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Transaction, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]models.Transaction), args.Error(1)
}

func TestPaymentService_GetBalance(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	userID := uuid.New()

	expected := &models.UserBalance{UserID: userID, Available: 1000, Frozen: 500}
	repo.On("GetBalance", ctx, userID).Return(expected, nil)

	balance, err := svc.GetBalance(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, expected, balance)
	repo.AssertExpectations(t)
}

func TestPaymentService_Deposit_Success(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	userID := uuid.New()

	expected := &models.Transaction{ID: uuid.New(), Amount: 1000}
	repo.On("Deposit", ctx, userID, float64(1000), "Пополнение баланса").Return(expected, nil)

	tx, err := svc.Deposit(ctx, userID, 1000)
	assert.NoError(t, err)
	assert.Equal(t, expected, tx)
}

func TestPaymentService_Deposit_InvalidAmount(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	userID := uuid.New()

	_, err := svc.Deposit(ctx, userID, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "положительной")

	_, err = svc.Deposit(ctx, userID, -100)
	assert.Error(t, err)
}

func TestPaymentService_CreateEscrow_Success(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()

	orderID := uuid.New()
	clientID := uuid.New()
	freelancerID := uuid.New()

	expected := &models.Escrow{ID: uuid.New(), Amount: 5000, Status: models.EscrowStatusHeld}
	repo.On("CreateEscrow", ctx, orderID, clientID, freelancerID, float64(5000)).Return(expected, nil)

	escrow, err := svc.CreateEscrow(ctx, orderID, clientID, freelancerID, 5000)
	assert.NoError(t, err)
	assert.Equal(t, expected, escrow)
}

func TestPaymentService_CreateEscrow_InsufficientFunds(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()

	orderID := uuid.New()
	clientID := uuid.New()
	freelancerID := uuid.New()

	repo.On("CreateEscrow", ctx, orderID, clientID, freelancerID, float64(5000)).Return(nil, repository.ErrInsufficientFunds)

	_, err := svc.CreateEscrow(ctx, orderID, clientID, freelancerID, 5000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "недостаточно средств")
}

func TestPaymentService_CreateEscrow_InvalidAmount(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()

	_, err := svc.CreateEscrow(ctx, uuid.New(), uuid.New(), uuid.New(), 0)
	assert.Error(t, err)
}

func TestPaymentService_ReleaseEscrow(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	orderID := uuid.New()

	expected := &models.Escrow{Status: models.EscrowStatusReleased}
	repo.On("ReleaseEscrow", ctx, orderID).Return(expected, nil)

	escrow, err := svc.ReleaseEscrow(ctx, orderID)
	assert.NoError(t, err)
	assert.Equal(t, models.EscrowStatusReleased, escrow.Status)
}

func TestPaymentService_RefundEscrow(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	orderID := uuid.New()

	expected := &models.Escrow{Status: models.EscrowStatusRefunded}
	repo.On("RefundEscrow", ctx, orderID).Return(expected, nil)

	escrow, err := svc.RefundEscrow(ctx, orderID)
	assert.NoError(t, err)
	assert.Equal(t, models.EscrowStatusRefunded, escrow.Status)
}

func TestPaymentService_ListTransactions(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	userID := uuid.New()

	expected := []models.Transaction{{ID: uuid.New()}, {ID: uuid.New()}}
	repo.On("ListTransactions", ctx, userID, 20, 0).Return(expected, nil)

	txs, err := svc.ListTransactions(ctx, userID, 20, 0)
	assert.NoError(t, err)
	assert.Len(t, txs, 2)
}

func TestPaymentService_ListTransactions_DefaultLimit(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	userID := uuid.New()

	repo.On("ListTransactions", ctx, userID, 20, 0).Return([]models.Transaction{}, nil)

	_, err := svc.ListTransactions(ctx, userID, 0, 0)
	assert.NoError(t, err)
}

func TestPaymentService_GetEscrow(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	orderID := uuid.New()

	expected := &models.Escrow{ID: uuid.New(), OrderID: orderID}
	repo.On("GetEscrowByOrderID", ctx, orderID).Return(expected, nil)

	escrow, err := svc.GetEscrow(ctx, orderID)
	assert.NoError(t, err)
	assert.Equal(t, expected, escrow)
}

func TestPaymentService_GetEscrow_NotFound(t *testing.T) {
	repo := new(mockPaymentRepo)
	svc := NewPaymentService(repo)
	ctx := context.Background()
	orderID := uuid.New()

	repo.On("GetEscrowByOrderID", ctx, orderID).Return(nil, errors.New("not found"))

	_, err := svc.GetEscrow(ctx, orderID)
	assert.Error(t, err)
}
