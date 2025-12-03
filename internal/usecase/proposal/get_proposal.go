package proposal

import (
	"context"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
)

type GetProposalUseCase struct {
	proposalRepo repository.ProposalRepository
}

func NewGetProposalUseCase(proposalRepo repository.ProposalRepository) *GetProposalUseCase {
	return &GetProposalUseCase{proposalRepo: proposalRepo}
}

func (uc *GetProposalUseCase) Execute(ctx context.Context, proposalID uuid.UUID) (*entity.Proposal, error) {
	return uc.proposalRepo.FindByID(ctx, proposalID)
}

type ListProposalsUseCase struct {
	proposalRepo repository.ProposalRepository
}

func NewListProposalsUseCase(proposalRepo repository.ProposalRepository) *ListProposalsUseCase {
	return &ListProposalsUseCase{proposalRepo: proposalRepo}
}

func (uc *ListProposalsUseCase) Execute(ctx context.Context, orderID uuid.UUID) ([]*entity.Proposal, error) {
	return uc.proposalRepo.FindByOrderID(ctx, orderID)
}

type ListMyProposalsUseCase struct {
	proposalRepo repository.ProposalRepository
}

func NewListMyProposalsUseCase(proposalRepo repository.ProposalRepository) *ListMyProposalsUseCase {
	return &ListMyProposalsUseCase{proposalRepo: proposalRepo}
}

func (uc *ListMyProposalsUseCase) Execute(ctx context.Context, freelancerID uuid.UUID) ([]*entity.Proposal, error) {
	return uc.proposalRepo.FindByFreelancerID(ctx, freelancerID)
}

type GetMyProposalForOrderUseCase struct {
	proposalRepo repository.ProposalRepository
}

func NewGetMyProposalForOrderUseCase(proposalRepo repository.ProposalRepository) *GetMyProposalForOrderUseCase {
	return &GetMyProposalForOrderUseCase{proposalRepo: proposalRepo}
}

func (uc *GetMyProposalForOrderUseCase) Execute(ctx context.Context, orderID, freelancerID uuid.UUID) (*entity.Proposal, error) {
	return uc.proposalRepo.FindByOrderAndFreelancer(ctx, orderID, freelancerID)
}
