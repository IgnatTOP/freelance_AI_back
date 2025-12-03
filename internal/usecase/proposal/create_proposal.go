package proposal

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type CreateProposalInput struct {
	OrderID          uuid.UUID
	FreelancerID     uuid.UUID
	CoverLetter      string
	ProposedBudget   float64
	ProposedDeadline *time.Time
}

type CreateProposalUseCase struct {
	proposalRepo repository.ProposalRepository
	orderRepo    repository.OrderRepository
}

func NewCreateProposalUseCase(proposalRepo repository.ProposalRepository, orderRepo repository.OrderRepository) *CreateProposalUseCase {
	return &CreateProposalUseCase{
		proposalRepo: proposalRepo,
		orderRepo:    orderRepo,
	}
}

func (uc *CreateProposalUseCase) Execute(ctx context.Context, input CreateProposalInput) (*entity.Proposal, error) {
	order, err := uc.orderRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}
	
	if order.ClientID == input.FreelancerID {
		return nil, apperror.New(apperror.ErrCodeBadRequest, "нельзя откликнуться на собственный заказ")
	}
	
	existing, err := uc.proposalRepo.FindByOrderAndFreelancer(ctx, input.OrderID, input.FreelancerID)
	if err == nil && existing != nil {
		return nil, apperror.New(apperror.ErrCodeConflict, "вы уже откликнулись на этот заказ")
	}
	
	if !order.Budget.IsInRange(input.ProposedBudget) {
		return nil, apperror.New(apperror.ErrCodeValidation, "предложенный бюджет выходит за рамки бюджета заказа")
	}
	
	proposal, err := entity.NewProposal(
		input.OrderID,
		input.FreelancerID,
		input.CoverLetter,
		input.ProposedBudget,
		input.ProposedDeadline,
	)
	if err != nil {
		return nil, err
	}
	
	if err := uc.proposalRepo.Create(ctx, proposal); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать предложение")
	}
	
	return proposal, nil
}
