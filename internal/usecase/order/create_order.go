package order

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type CreateOrderInput struct {
	ClientID      uuid.UUID
	Title         string
	Description   string
	BudgetMin     float64
	BudgetMax     float64
	DeadlineAt    *time.Time
	Requirements  []RequirementInput
	AttachmentIDs []uuid.UUID
}

type RequirementInput struct {
	Skill string
	Level string
}

type CreateOrderUseCase struct {
	orderRepo repository.OrderRepository
}

func NewCreateOrderUseCase(orderRepo repository.OrderRepository) *CreateOrderUseCase {
	return &CreateOrderUseCase{orderRepo: orderRepo}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, input CreateOrderInput) (*entity.Order, error) {
	order, err := entity.NewOrder(
		input.ClientID,
		input.Title,
		input.Description,
		input.BudgetMin,
		input.BudgetMax,
		input.DeadlineAt,
	)
	if err != nil {
		return nil, err
	}
	
	for _, req := range input.Requirements {
		if req.Level == "" {
			req.Level = "middle"
		}
		order.Requirements = append(order.Requirements, entity.OrderRequirement{
			ID:      uuid.New(),
			OrderID: order.ID,
			Skill:   req.Skill,
			Level:   req.Level,
		})
	}
	
	for _, mediaID := range input.AttachmentIDs {
		order.Attachments = append(order.Attachments, entity.OrderAttachment{
			ID:        uuid.New(),
			OrderID:   order.ID,
			MediaID:   mediaID,
			CreatedAt: time.Now(),
		})
	}
	
	if err := uc.orderRepo.Create(ctx, order); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать заказ")
	}
	
	return order, nil
}
