package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

var ErrMinWithdrawalAmount = errors.New("minimum withdrawal amount is 100 RUB")

type WithdrawalService struct {
	repo *repository.WithdrawalRepository
}

func NewWithdrawalService(r *repository.WithdrawalRepository) *WithdrawalService {
	return &WithdrawalService{repo: r}
}

func (s *WithdrawalService) CreateWithdrawal(ctx context.Context, userID uuid.UUID, amount float64, cardLast4, bankName string) (*models.Withdrawal, error) {
	if amount < 100 {
		return nil, ErrMinWithdrawalAmount
	}
	return s.repo.Create(ctx, userID, amount, cardLast4, bankName)
}

func (s *WithdrawalService) GetWithdrawal(ctx context.Context, id uuid.UUID) (*models.Withdrawal, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *WithdrawalService) ListUserWithdrawals(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Withdrawal, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}
