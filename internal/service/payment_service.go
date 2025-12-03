package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

type PaymentRepository interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalance, error)
	Deposit(ctx context.Context, userID uuid.UUID, amount float64, description string) (*models.Transaction, error)
	CreateEscrow(ctx context.Context, orderID, clientID, freelancerID uuid.UUID, amount float64) (*models.Escrow, error)
	ReleaseEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error)
	RefundEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error)
	GetEscrowByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error)
	ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Transaction, error)
}

type PaymentService struct {
	repo PaymentRepository
}

func NewPaymentService(repo PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

// GetBalance возвращает баланс пользователя.
func (s *PaymentService) GetBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalance, error) {
	return s.repo.GetBalance(ctx, userID)
}

// Deposit пополняет баланс.
func (s *PaymentService) Deposit(ctx context.Context, userID uuid.UUID, amount float64) (*models.Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("сумма должна быть положительной")
	}
	return s.repo.Deposit(ctx, userID, amount, "Пополнение баланса")
}

// CreateEscrow создаёт защищённую сделку.
func (s *PaymentService) CreateEscrow(ctx context.Context, orderID, clientID, freelancerID uuid.UUID, amount float64) (*models.Escrow, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("сумма должна быть положительной")
	}
	escrow, err := s.repo.CreateEscrow(ctx, orderID, clientID, freelancerID, amount)
	if err != nil {
		if err == repository.ErrInsufficientFunds {
			return nil, fmt.Errorf("недостаточно средств на балансе")
		}
		return nil, err
	}
	return escrow, nil
}

// ReleaseEscrow освобождает средства фрилансеру после завершения заказа.
func (s *PaymentService) ReleaseEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	return s.repo.ReleaseEscrow(ctx, orderID)
}

// RefundEscrow возвращает средства клиенту при отмене заказа.
func (s *PaymentService) RefundEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	return s.repo.RefundEscrow(ctx, orderID)
}

// GetEscrow возвращает escrow по заказу.
func (s *PaymentService) GetEscrow(ctx context.Context, orderID uuid.UUID) (*models.Escrow, error) {
	return s.repo.GetEscrowByOrderID(ctx, orderID)
}

// ListTransactions возвращает историю транзакций.
func (s *PaymentService) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Transaction, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListTransactions(ctx, userID, limit, offset)
}
