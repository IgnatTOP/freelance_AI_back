package order

import (
	"context"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type PublishOrderUseCase struct {
	orderRepo repository.OrderRepository
}

func NewPublishOrderUseCase(orderRepo repository.OrderRepository) *PublishOrderUseCase {
	return &PublishOrderUseCase{orderRepo: orderRepo}
}

func (uc *PublishOrderUseCase) Execute(ctx context.Context, orderID, clientID uuid.UUID) (*entity.Order, error) {
	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !order.IsOwnedBy(clientID) {
		return nil, apperror.ErrForbidden
	}

	if err := order.Publish(); err != nil {
		return nil, err
	}

	if err := uc.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

type CancelOrderUseCase struct {
	orderRepo repository.OrderRepository
}

func NewCancelOrderUseCase(orderRepo repository.OrderRepository) *CancelOrderUseCase {
	return &CancelOrderUseCase{orderRepo: orderRepo}
}

func (uc *CancelOrderUseCase) Execute(ctx context.Context, orderID, clientID uuid.UUID) (*entity.Order, error) {
	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !order.IsOwnedBy(clientID) {
		return nil, apperror.ErrForbidden
	}

	if err := order.Cancel(); err != nil {
		return nil, err
	}

	if err := uc.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

type CompleteOrderUseCase struct {
	orderRepo repository.OrderRepository
}

func NewCompleteOrderUseCase(orderRepo repository.OrderRepository) *CompleteOrderUseCase {
	return &CompleteOrderUseCase{orderRepo: orderRepo}
}

func (uc *CompleteOrderUseCase) Execute(ctx context.Context, orderID, clientID uuid.UUID) (*entity.Order, error) {
	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !order.IsOwnedBy(clientID) {
		return nil, apperror.ErrForbidden
	}

	if err := order.Complete(); err != nil {
		return nil, err
	}

	if err := uc.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

type ListMyOrdersUseCase struct {
	orderRepo repository.OrderRepository
}

func NewListMyOrdersUseCase(orderRepo repository.OrderRepository) *ListMyOrdersUseCase {
	return &ListMyOrdersUseCase{orderRepo: orderRepo}
}

func (uc *ListMyOrdersUseCase) Execute(ctx context.Context, clientID uuid.UUID) ([]*entity.Order, error) {
	return uc.orderRepo.FindByClientID(ctx, clientID)
}
