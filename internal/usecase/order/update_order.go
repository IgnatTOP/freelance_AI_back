package order

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type UpdateOrderInput struct {
	OrderID       uuid.UUID
	ClientID      uuid.UUID
	Title         string
	Description   string
	BudgetMin     float64
	BudgetMax     float64
	DeadlineAt    *time.Time
	Requirements  []RequirementInput
	AttachmentIDs []uuid.UUID
}

type UpdateOrderUseCase struct {
	orderRepo repository.OrderRepository
}

func NewUpdateOrderUseCase(orderRepo repository.OrderRepository) *UpdateOrderUseCase {
	return &UpdateOrderUseCase{orderRepo: orderRepo}
}

func (uc *UpdateOrderUseCase) Execute(ctx context.Context, input UpdateOrderInput) (*entity.Order, error) {
	order, err := uc.orderRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}
	
	if !order.IsOwnedBy(input.ClientID) {
		return nil, apperror.ErrForbidden
	}
	
	if err := order.Update(input.Title, input.Description, input.BudgetMin, input.BudgetMax, input.DeadlineAt); err != nil {
		return nil, err
	}
	
	if input.Requirements != nil {
		order.Requirements = make([]entity.OrderRequirement, 0, len(input.Requirements))
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
		
		if err := uc.orderRepo.UpdateRequirements(ctx, order.ID, order.Requirements); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось обновить требования")
		}
	}
	
	if input.AttachmentIDs != nil {
		order.Attachments = make([]entity.OrderAttachment, 0, len(input.AttachmentIDs))
		for _, mediaID := range input.AttachmentIDs {
			order.Attachments = append(order.Attachments, entity.OrderAttachment{
				ID:        uuid.New(),
				OrderID:   order.ID,
				MediaID:   mediaID,
				CreatedAt: time.Now(),
			})
		}
		
		if err := uc.orderRepo.UpdateAttachments(ctx, order.ID, order.Attachments); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось обновить вложения")
		}
	}
	
	if err := uc.orderRepo.Update(ctx, order); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось обновить заказ")
	}
	
	return order, nil
}
