package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
)

type OrderRepository interface {
	Create(ctx context.Context, order *entity.Order) error
	Update(ctx context.Context, order *entity.Order) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error)
	FindByIDWithDetails(ctx context.Context, id uuid.UUID) (*entity.Order, error)
	FindByClientID(ctx context.Context, clientID uuid.UUID) ([]*entity.Order, error)
	List(ctx context.Context, filter OrderFilter) ([]*entity.Order, int, error)
	
	CreateRequirement(ctx context.Context, req *entity.OrderRequirement) error
	UpdateRequirements(ctx context.Context, orderID uuid.UUID, requirements []entity.OrderRequirement) error
	FindRequirements(ctx context.Context, orderID uuid.UUID) ([]entity.OrderRequirement, error)
	
	CreateAttachment(ctx context.Context, att *entity.OrderAttachment) error
	UpdateAttachments(ctx context.Context, orderID uuid.UUID, attachments []entity.OrderAttachment) error
	FindAttachments(ctx context.Context, orderID uuid.UUID) ([]entity.OrderAttachment, error)
}

type OrderFilter struct {
	Status     string
	Skills     []string
	BudgetMin  *float64
	BudgetMax  *float64
	Search     string
	SortBy     string
	SortOrder  string
	Limit      int
	Offset     int
}
