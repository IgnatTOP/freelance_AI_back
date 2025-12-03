package repository

import (
	"context"

	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
)

type AIService interface {
	SummarizeOrder(ctx context.Context, title, description string) (string, error)
	StreamSummarizeOrder(ctx context.Context, title, description string, onDelta func(chunk string) error) error
	GenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string) (string, error)
	StreamGenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string, onDelta func(chunk string) error) error
	ImproveOrderDescription(ctx context.Context, title, description string) (string, error)
	StreamImproveOrderDescription(ctx context.Context, title, description string, onDelta func(chunk string) error) error
	GenerateProposal(ctx context.Context, order *entity.Order, userSkills []string, userExperience string) (string, error)
	StreamGenerateProposal(ctx context.Context, order *entity.Order, userSkills []string, userExperience string, onDelta func(chunk string) error) error
	ProposalFeedback(ctx context.Context, order *entity.Order, coverLetter string) (string, error)
	StreamProposalFeedback(ctx context.Context, order *entity.Order, coverLetter string, onDelta func(chunk string) error) error
}
