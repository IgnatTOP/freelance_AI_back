package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
)

type CreateOrderRequest struct {
	Title         string              `json:"title" binding:"required"`
	Description   string              `json:"description" binding:"required"`
	BudgetMin     float64             `json:"budget_min" binding:"required,gt=0"`
	BudgetMax     float64             `json:"budget_max" binding:"required,gt=0"`
	DeadlineAt    *string             `json:"deadline_at"`
	Requirements  []RequirementDTO    `json:"requirements"`
	AttachmentIDs []string            `json:"attachment_ids"`
}

type UpdateOrderRequest struct {
	Title         string              `json:"title"`
	Description   string              `json:"description"`
	BudgetMin     float64             `json:"budget_min"`
	BudgetMax     float64             `json:"budget_max"`
	DeadlineAt    *string             `json:"deadline_at"`
	Requirements  []RequirementDTO    `json:"requirements"`
	AttachmentIDs []string            `json:"attachment_ids"`
}

type RequirementDTO struct {
	Skill string `json:"skill" binding:"required"`
	Level string `json:"level"`
}

type OrderResponse struct {
	ID                              uuid.UUID           `json:"id"`
	ClientID                        uuid.UUID           `json:"client_id"`
	Title                           string              `json:"title"`
	Description                     string              `json:"description"`
	BudgetMin                       float64             `json:"budget_min"`
	BudgetMax                       float64             `json:"budget_max"`
	Status                          string              `json:"status"`
	DeadlineAt                      *time.Time          `json:"deadline_at"`
	AISummary                       *string             `json:"ai_summary"`
	BestRecommendationProposalID    *uuid.UUID          `json:"best_recommendation_proposal_id"`
	BestRecommendationJustification *string             `json:"best_recommendation_justification"`
	AIAnalysisUpdatedAt             *time.Time          `json:"ai_analysis_updated_at"`
	CreatedAt                       time.Time           `json:"created_at"`
	UpdatedAt                       time.Time           `json:"updated_at"`
	Requirements                    []RequirementDTO    `json:"requirements"`
	Attachments                     []AttachmentDTO     `json:"attachments"`
}

type AttachmentDTO struct {
	ID        uuid.UUID `json:"id"`
	MediaID   uuid.UUID `json:"media_id"`
	CreatedAt time.Time `json:"created_at"`
}

func ToOrderResponse(order *entity.Order) OrderResponse {
	resp := OrderResponse{
		ID:                              order.ID,
		ClientID:                        order.ClientID,
		Title:                           order.Title,
		Description:                     order.Description,
		BudgetMin:                       order.Budget.Min.Amount,
		BudgetMax:                       order.Budget.Max.Amount,
		Status:                          string(order.Status),
		DeadlineAt:                      order.DeadlineAt,
		AISummary:                       order.AISummary,
		BestRecommendationProposalID:    order.BestRecommendationProposalID,
		BestRecommendationJustification: order.BestRecommendationJustification,
		AIAnalysisUpdatedAt:             order.AIAnalysisUpdatedAt,
		CreatedAt:                       order.CreatedAt,
		UpdatedAt:                       order.UpdatedAt,
		Requirements:                    make([]RequirementDTO, 0, len(order.Requirements)),
		Attachments:                     make([]AttachmentDTO, 0, len(order.Attachments)),
	}
	
	for _, req := range order.Requirements {
		resp.Requirements = append(resp.Requirements, RequirementDTO{
			Skill: req.Skill,
			Level: req.Level,
		})
	}
	
	for _, att := range order.Attachments {
		resp.Attachments = append(resp.Attachments, AttachmentDTO{
			ID:        att.ID,
			MediaID:   att.MediaID,
			CreatedAt: att.CreatedAt,
		})
	}
	
	return resp
}

func ToOrderResponses(orders []*entity.Order) []OrderResponse {
	responses := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, ToOrderResponse(order))
	}
	return responses
}

func ParseDeadline(deadlineStr *string) (*time.Time, error) {
	if deadlineStr == nil || *deadlineStr == "" {
		return nil, nil
	}
	
	deadline, err := time.Parse(time.RFC3339, *deadlineStr)
	if err != nil {
		return nil, err
	}
	
	return &deadline, nil
}

func ParseUUIDs(uuidStrs []string) ([]uuid.UUID, error) {
	uuids := make([]uuid.UUID, 0, len(uuidStrs))
	for _, str := range uuidStrs {
		id, err := uuid.Parse(str)
		if err != nil {
			return nil, err
		}
		uuids = append(uuids, id)
	}
	return uuids, nil
}
