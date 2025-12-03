package dto

import (
	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/models"
)

// OrderResponse represents an order with its requirements and attachments
// This eliminates the duplicate orderResponse structs in handlers
type OrderResponse struct {
	*models.Order
	Requirements []models.OrderRequirement `json:"requirements"`
	Attachments  []models.OrderAttachment  `json:"attachments"`
}

// NewOrderResponse creates an OrderResponse from components
func NewOrderResponse(order *models.Order, requirements []models.OrderRequirement, attachments []models.OrderAttachment) *OrderResponse {
	return &OrderResponse{
		Order:        order,
		Requirements: requirements,
		Attachments:  attachments,
	}
}

// PortfolioItemResponse represents a portfolio item with its media
// This eliminates the duplicate portfolioResponse structs in handlers
type PortfolioItemResponse struct {
	*models.PortfolioItem
	Media []models.MediaFile `json:"media"`
}

// NewPortfolioItemResponse creates a PortfolioItemResponse from components
func NewPortfolioItemResponse(item *models.PortfolioItem, media []models.MediaFile) *PortfolioItemResponse {
	return &PortfolioItemResponse{
		PortfolioItem: item,
		Media:         media,
	}
}

// ConversationResponse represents a conversation with additional details
type ConversationResponse struct {
	*models.Conversation
	OrderTitle       string           `json:"order_title"`
	OtherParticipant *ParticipantInfo `json:"other_participant"`
	LastMessage      *models.Message  `json:"last_message,omitempty"`
	UnreadCount      int              `json:"unread_count"`
}

// ParticipantInfo represents basic info about a conversation participant
type ParticipantInfo struct {
	UserID      uuid.UUID  `json:"user_id"`
	DisplayName string     `json:"display_name"`
	PhotoID     *uuid.UUID `json:"photo_id,omitempty"`
}

// ProposalWithOrderResponse represents a proposal with associated order info
type ProposalWithOrderResponse struct {
	*models.Proposal
	Order *OrderShortInfo `json:"order"`
}

// OrderShortInfo represents basic order information
type OrderShortInfo struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ClientID    uuid.UUID `json:"client_id"`
}

// MessageListResponse represents enriched message list with conversation details
type MessageListResponse struct {
	Messages         []models.Message `json:"messages"`
	ConversationID   uuid.UUID        `json:"conversation_id"`
	OrderTitle       string           `json:"order_title"`
	OtherParticipant *ParticipantInfo `json:"other_participant"`
}

// PaginatedOrdersResponse represents paginated orders list
type PaginatedOrdersResponse struct {
	Data       []models.Order `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// ProposalsListResponse represents proposals with AI recommendation
type ProposalsListResponse struct {
	Proposals                    []models.Proposal `json:"proposals"`
	BestRecommendationProposalID *uuid.UUID        `json:"best_recommendation_proposal_id,omitempty"`
	RecommendationJustification  *string           `json:"recommendation_justification,omitempty"`
}

// UpdateProposalStatusResponse represents response after updating proposal status
type UpdateProposalStatusResponse struct {
	Proposal     *models.Proposal     `json:"proposal"`
	Conversation *models.Conversation `json:"conversation,omitempty"`
	Order        *OrderShortInfo      `json:"order,omitempty"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
