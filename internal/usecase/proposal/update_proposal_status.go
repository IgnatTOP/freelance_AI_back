package proposal

import (
	"context"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/domain/valueobject"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type UpdateProposalStatusUseCase struct {
	proposalRepo repository.ProposalRepository
	orderRepo    repository.OrderRepository
}

func NewUpdateProposalStatusUseCase(proposalRepo repository.ProposalRepository, orderRepo repository.OrderRepository) *UpdateProposalStatusUseCase {
	return &UpdateProposalStatusUseCase{
		proposalRepo: proposalRepo,
		orderRepo:    orderRepo,
	}
}

func (uc *UpdateProposalStatusUseCase) Execute(ctx context.Context, proposalID, clientID uuid.UUID, newStatus string) (*entity.Proposal, error) {
	proposal, err := uc.proposalRepo.FindByID(ctx, proposalID)
	if err != nil {
		return nil, err
	}
	
	order, err := uc.orderRepo.FindByID(ctx, proposal.OrderID)
	if err != nil {
		return nil, err
	}
	
	if !order.IsOwnedBy(clientID) {
		return nil, apperror.ErrForbidden
	}
	
	switch newStatus {
	case string(valueobject.ProposalStatusAccepted):
		if err := proposal.Accept(); err != nil {
			return nil, err
		}
		
		if err := order.StartWork(); err != nil {
			return nil, err
		}
		
		if err := uc.orderRepo.Update(ctx, order); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось обновить статус заказа")
		}
		
	case string(valueobject.ProposalStatusRejected):
		if err := proposal.Reject(); err != nil {
			return nil, err
		}
		
	default:
		return nil, apperror.New(apperror.ErrCodeValidation, "некорректный статус предложения")
	}
	
	if err := uc.proposalRepo.Update(ctx, proposal); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось обновить предложение")
	}
	
	return proposal, nil
}
