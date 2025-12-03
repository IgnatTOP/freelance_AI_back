package ai

import (
	"context"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type GenerateOrderDescriptionUseCase struct {
	aiService repository.AIService
}

func NewGenerateOrderDescriptionUseCase(aiService repository.AIService) *GenerateOrderDescriptionUseCase {
	return &GenerateOrderDescriptionUseCase{aiService: aiService}
}

func (uc *GenerateOrderDescriptionUseCase) Execute(ctx context.Context, title, briefDescription string, skills []string) (string, error) {
	if uc.aiService == nil {
		return "", apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}
	return uc.aiService.GenerateOrderDescription(ctx, title, briefDescription, skills)
}

func (uc *GenerateOrderDescriptionUseCase) ExecuteStream(ctx context.Context, title, briefDescription string, skills []string, onDelta func(chunk string) error) error {
	if uc.aiService == nil {
		return apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}
	return uc.aiService.StreamGenerateOrderDescription(ctx, title, briefDescription, skills, onDelta)
}

type ImproveOrderDescriptionUseCase struct {
	aiService repository.AIService
}

func NewImproveOrderDescriptionUseCase(aiService repository.AIService) *ImproveOrderDescriptionUseCase {
	return &ImproveOrderDescriptionUseCase{aiService: aiService}
}

func (uc *ImproveOrderDescriptionUseCase) Execute(ctx context.Context, title, description string) (string, error) {
	if uc.aiService == nil {
		return "", apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}
	return uc.aiService.ImproveOrderDescription(ctx, title, description)
}

func (uc *ImproveOrderDescriptionUseCase) ExecuteStream(ctx context.Context, title, description string, onDelta func(chunk string) error) error {
	if uc.aiService == nil {
		return apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}
	return uc.aiService.StreamImproveOrderDescription(ctx, title, description, onDelta)
}

type SummarizeOrderUseCase struct {
	aiService repository.AIService
	orderRepo repository.OrderRepository
}

func NewSummarizeOrderUseCase(aiService repository.AIService, orderRepo repository.OrderRepository) *SummarizeOrderUseCase {
	return &SummarizeOrderUseCase{aiService: aiService, orderRepo: orderRepo}
}

func (uc *SummarizeOrderUseCase) Execute(ctx context.Context, orderID uuid.UUID) (string, error) {
	if uc.aiService == nil {
		return "", apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}

	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return "", err
	}

	summary, err := uc.aiService.SummarizeOrder(ctx, order.Title, order.Description)
	if err != nil {
		return "", err
	}

	order.SetAISummary(summary)
	if err := uc.orderRepo.Update(ctx, order); err != nil {
		return "", err
	}

	return summary, nil
}

func (uc *SummarizeOrderUseCase) ExecuteStream(ctx context.Context, orderID uuid.UUID, onDelta func(chunk string) error) error {
	if uc.aiService == nil {
		return apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}

	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	return uc.aiService.StreamSummarizeOrder(ctx, order.Title, order.Description, onDelta)
}

type GenerateProposalUseCase struct {
	aiService repository.AIService
	orderRepo repository.OrderRepository
}

func NewGenerateProposalUseCase(aiService repository.AIService, orderRepo repository.OrderRepository) *GenerateProposalUseCase {
	return &GenerateProposalUseCase{aiService: aiService, orderRepo: orderRepo}
}

func (uc *GenerateProposalUseCase) Execute(ctx context.Context, orderID uuid.UUID, userSkills []string, userExperience string) (string, error) {
	if uc.aiService == nil {
		return "", apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}

	order, err := uc.orderRepo.FindByIDWithDetails(ctx, orderID)
	if err != nil {
		return "", err
	}

	return uc.aiService.GenerateProposal(ctx, order, userSkills, userExperience)
}

func (uc *GenerateProposalUseCase) ExecuteStream(ctx context.Context, orderID uuid.UUID, userSkills []string, userExperience string, onDelta func(chunk string) error) error {
	if uc.aiService == nil {
		return apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}

	order, err := uc.orderRepo.FindByIDWithDetails(ctx, orderID)
	if err != nil {
		return err
	}

	return uc.aiService.StreamGenerateProposal(ctx, order, userSkills, userExperience, onDelta)
}

type ProposalFeedbackUseCase struct {
	aiService    repository.AIService
	orderRepo    repository.OrderRepository
	proposalRepo repository.ProposalRepository
}

func NewProposalFeedbackUseCase(aiService repository.AIService, orderRepo repository.OrderRepository, proposalRepo repository.ProposalRepository) *ProposalFeedbackUseCase {
	return &ProposalFeedbackUseCase{aiService: aiService, orderRepo: orderRepo, proposalRepo: proposalRepo}
}

func (uc *ProposalFeedbackUseCase) Execute(ctx context.Context, proposalID uuid.UUID) (string, error) {
	if uc.aiService == nil {
		return "", apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}

	proposal, err := uc.proposalRepo.FindByID(ctx, proposalID)
	if err != nil {
		return "", err
	}

	order, err := uc.orderRepo.FindByID(ctx, proposal.OrderID)
	if err != nil {
		return "", err
	}

	feedback, err := uc.aiService.ProposalFeedback(ctx, order, proposal.CoverLetter)
	if err != nil {
		return "", err
	}

	proposal.SetAIFeedback(feedback)
	if err := uc.proposalRepo.Update(ctx, proposal); err != nil {
		return "", err
	}

	return feedback, nil
}

func (uc *ProposalFeedbackUseCase) ExecuteStream(ctx context.Context, proposalID uuid.UUID, onDelta func(chunk string) error) error {
	if uc.aiService == nil {
		return apperror.New(apperror.ErrCodeBadRequest, "AI сервис недоступен")
	}

	proposal, err := uc.proposalRepo.FindByID(ctx, proposalID)
	if err != nil {
		return err
	}

	order, err := uc.orderRepo.FindByID(ctx, proposal.OrderID)
	if err != nil {
		return err
	}

	return uc.aiService.StreamProposalFeedback(ctx, order, proposal.CoverLetter, onDelta)
}
