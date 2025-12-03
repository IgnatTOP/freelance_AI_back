package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
)

type ProposalRepository interface {
	Create(ctx context.Context, proposal *entity.Proposal) error
	Update(ctx context.Context, proposal *entity.Proposal) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Proposal, error)
	FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.Proposal, error)
	FindByFreelancerID(ctx context.Context, freelancerID uuid.UUID) ([]*entity.Proposal, error)
	FindByOrderAndFreelancer(ctx context.Context, orderID, freelancerID uuid.UUID) (*entity.Proposal, error)
	GetLastUpdateTime(ctx context.Context, orderID uuid.UUID) (*time.Time, error)
}
