package order

import (
	"context"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type GetOrderUseCase struct {
	orderRepo repository.OrderRepository
}

func NewGetOrderUseCase(orderRepo repository.OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{orderRepo: orderRepo}
}

func (uc *GetOrderUseCase) Execute(ctx context.Context, orderID uuid.UUID) (*entity.Order, error) {
	return uc.orderRepo.FindByIDWithDetails(ctx, orderID)
}

type ListOrdersUseCase struct {
	orderRepo repository.OrderRepository
}

func NewListOrdersUseCase(orderRepo repository.OrderRepository) *ListOrdersUseCase {
	return &ListOrdersUseCase{orderRepo: orderRepo}
}

func (uc *ListOrdersUseCase) Execute(ctx context.Context, filter repository.OrderFilter) ([]*entity.Order, int, error) {
	return uc.orderRepo.List(ctx, filter)
}

type DeleteOrderUseCase struct {
	orderRepo repository.OrderRepository
}

func NewDeleteOrderUseCase(orderRepo repository.OrderRepository) *DeleteOrderUseCase {
	return &DeleteOrderUseCase{orderRepo: orderRepo}
}

func (uc *DeleteOrderUseCase) Execute(ctx context.Context, orderID, clientID uuid.UUID) error {
	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	
	if !order.IsOwnedBy(clientID) {
		return apperror.ErrForbidden
	}
	
	return uc.orderRepo.Delete(ctx, orderID)
}
