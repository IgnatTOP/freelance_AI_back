package ai

import (
	"context"

	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	oldAI "github.com/ignatzorin/freelance-backend/internal/ai"
	"github.com/ignatzorin/freelance-backend/internal/models"
)

type AIServiceAdapter struct {
	client *oldAI.Client
}

func NewAIServiceAdapter(client *oldAI.Client) *AIServiceAdapter {
	if client == nil {
		return nil
	}
	return &AIServiceAdapter{client: client}
}

func (a *AIServiceAdapter) SummarizeOrder(ctx context.Context, title, description string) (string, error) {
	return a.client.SummarizeOrder(ctx, title, description)
}

func (a *AIServiceAdapter) StreamSummarizeOrder(ctx context.Context, title, description string, onDelta func(chunk string) error) error {
	return a.client.StreamSummarizeOrder(ctx, title, description, onDelta)
}

func (a *AIServiceAdapter) GenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string) (string, error) {
	return a.client.GenerateOrderDescription(ctx, title, briefDescription, skills)
}

func (a *AIServiceAdapter) StreamGenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string, onDelta func(chunk string) error) error {
	return a.client.StreamGenerateOrderDescription(ctx, title, briefDescription, skills, onDelta)
}

func (a *AIServiceAdapter) ImproveOrderDescription(ctx context.Context, title, description string) (string, error) {
	return a.client.ImproveOrderDescription(ctx, title, description)
}

func (a *AIServiceAdapter) StreamImproveOrderDescription(ctx context.Context, title, description string, onDelta func(chunk string) error) error {
	return a.client.StreamImproveOrderDescription(ctx, title, description, onDelta)
}

func (a *AIServiceAdapter) GenerateProposal(ctx context.Context, order *entity.Order, userSkills []string, userExperience string) (string, error) {
	oldOrder := toOldOrder(order)
	return a.client.GenerateProposal(ctx, oldOrder, nil, userSkills, userExperience, nil)
}

func (a *AIServiceAdapter) StreamGenerateProposal(ctx context.Context, order *entity.Order, userSkills []string, userExperience string, onDelta func(chunk string) error) error {
	oldOrder := toOldOrder(order)
	return a.client.StreamGenerateProposal(ctx, oldOrder, nil, userSkills, userExperience, nil, onDelta)
}

func (a *AIServiceAdapter) ProposalFeedback(ctx context.Context, order *entity.Order, coverLetter string) (string, error) {
	oldOrder := toOldOrder(order)
	return a.client.ProposalFeedback(ctx, oldOrder, coverLetter)
}

func (a *AIServiceAdapter) StreamProposalFeedback(ctx context.Context, order *entity.Order, coverLetter string, onDelta func(chunk string) error) error {
	oldOrder := toOldOrder(order)
	return a.client.StreamProposalFeedback(ctx, oldOrder, coverLetter, onDelta)
}

func toOldOrder(order *entity.Order) *models.Order {
	budgetMin := order.Budget.Min.Amount
	budgetMax := order.Budget.Max.Amount
	return &models.Order{
		ID:          order.ID,
		ClientID:    order.ClientID,
		Title:       order.Title,
		Description: order.Description,
		BudgetMin:   &budgetMin,
		BudgetMax:   &budgetMax,
		Status:      string(order.Status),
		DeadlineAt:  order.DeadlineAt,
		AISummary:   order.AISummary,
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
	}
}
