package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
)

type CreateProposalRequest struct {
	CoverLetter      string  `json:"cover_letter" binding:"required"`
	ProposedBudget   float64 `json:"proposed_budget" binding:"required,gt=0"`
	ProposedDeadline *string `json:"proposed_deadline"`
}

type UpdateProposalStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=accepted rejected"`
}

type ProposalResponse struct {
	ID                      uuid.UUID  `json:"id"`
	OrderID                 uuid.UUID  `json:"order_id"`
	FreelancerID            uuid.UUID  `json:"freelancer_id"`
	CoverLetter             string     `json:"cover_letter"`
	ProposedBudget          float64    `json:"proposed_budget"`
	ProposedDeadline        *time.Time `json:"proposed_deadline"`
	Status                  string     `json:"status"`
	AIFeedback              *string    `json:"ai_feedback"`
	AIAnalysisForClient     *string    `json:"ai_analysis_for_client"`
	AIAnalysisForClientAt   *time.Time `json:"ai_analysis_for_client_at"`
	CompletedByFreelancerAt *time.Time `json:"completed_by_freelancer_at"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

func ToProposalResponse(proposal *entity.Proposal) ProposalResponse {
	return ProposalResponse{
		ID:                      proposal.ID,
		OrderID:                 proposal.OrderID,
		FreelancerID:            proposal.FreelancerID,
		CoverLetter:             proposal.CoverLetter,
		ProposedBudget:          proposal.ProposedBudget,
		ProposedDeadline:        proposal.ProposedDeadline,
		Status:                  string(proposal.Status),
		AIFeedback:              proposal.AIFeedback,
		AIAnalysisForClient:     proposal.AIAnalysisForClient,
		AIAnalysisForClientAt:   proposal.AIAnalysisForClientAt,
		CompletedByFreelancerAt: proposal.CompletedByFreelancerAt,
		CreatedAt:               proposal.CreatedAt,
		UpdatedAt:               proposal.UpdatedAt,
	}
}

func ToProposalResponses(proposals []*entity.Proposal) []ProposalResponse {
	responses := make([]ProposalResponse, 0, len(proposals))
	for _, proposal := range proposals {
		responses = append(responses, ToProposalResponse(proposal))
	}
	return responses
}
