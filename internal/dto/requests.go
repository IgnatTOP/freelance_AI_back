package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateOrderRequest represents the request to create an order
type CreateOrderRequest struct {
	Title        string                    `json:"title" binding:"required"`
	Description  string                    `json:"description" binding:"required"`
	CategoryID   *string                   `json:"category_id"`
	BudgetMin    *float64                  `json:"budget_min"`
	BudgetMax    *float64                  `json:"budget_max"`
	DeadlineAt   *string                   `json:"deadline_at"`
	Requirements []OrderRequirementRequest `json:"requirements"`
	Attachments  []string                  `json:"attachment_ids"`
}

// OrderRequirementRequest represents a skill requirement for an order
type OrderRequirementRequest struct {
	Skill string `json:"skill" binding:"required"`
	Level string `json:"level"`
}

// UpdateOrderRequest represents the request to update an order
type UpdateOrderRequest struct {
	Title        string                    `json:"title" binding:"required"`
	Description  string                    `json:"description" binding:"required"`
	CategoryID   *string                   `json:"category_id"`
	BudgetMin    *float64                  `json:"budget_min"`
	BudgetMax    *float64                  `json:"budget_max"`
	DeadlineAt   *string                   `json:"deadline_at"`
	Status       string                    `json:"status"`
	Requirements []OrderRequirementRequest `json:"requirements"`
	Attachments  []string                  `json:"attachment_ids"`
}

// CreateProposalRequest represents the request to create a proposal
type CreateProposalRequest struct {
	CoverLetter string   `json:"cover_letter" binding:"required"`
	Amount      *float64 `json:"amount"`
}

// UpdateProposalStatusRequest represents the request to update proposal status
type UpdateProposalStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// SendMessageRequest represents the request to send a message
type SendMessageRequest struct {
	Content         string   `json:"content"`
	ParentMessageID *string  `json:"parent_message_id"`
	Attachments     []string `json:"attachment_ids"`
}

// UpdateMessageRequest represents the request to update a message
type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// AddMessageReactionRequest represents the request to add a reaction
type AddMessageReactionRequest struct {
	Emoji string `json:"emoji" binding:"required"`
}

// CreatePortfolioItemRequest represents the request to create a portfolio item
type CreatePortfolioItemRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description" binding:"required"`
	URL         *string  `json:"url"`
	MediaIDs    []string `json:"media_ids"`
	Tags        []string `json:"tags"`
}

// UpdatePortfolioItemRequest represents the request to update a portfolio item
type UpdatePortfolioItemRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description" binding:"required"`
	URL         *string  `json:"url"`
	MediaIDs    []string `json:"media_ids"`
	Tags        []string `json:"tags"`
}

// UpdateProfileRequest represents the request to update user profile
type UpdateProfileRequest struct {
	DisplayName     string   `json:"display_name" binding:"required"`
	Bio             *string  `json:"bio"`
	HourlyRate      *float64 `json:"hourly_rate"`
	ExperienceLevel string   `json:"experience_level"`
	Skills          []string `json:"skills"`
	Location        *string  `json:"location"`
	PhotoID         *string  `json:"photo_id"`
	Phone           *string  `json:"phone"`
	Telegram        *string  `json:"telegram"`
	Website         *string  `json:"website"`
	CompanyName     *string  `json:"company_name"`
	INN             *string  `json:"inn"`
}

// UpdateRoleRequest represents the request to update user role
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// ParseDeadline converts string deadline to time.Time pointer
func (r *CreateOrderRequest) ParseDeadline() (*time.Time, error) {
	if r.DeadlineAt == nil || *r.DeadlineAt == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *r.DeadlineAt)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// ParseDeadline converts string deadline to time.Time pointer
func (r *UpdateOrderRequest) ParseDeadline() (*time.Time, error) {
	if r.DeadlineAt == nil || *r.DeadlineAt == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *r.DeadlineAt)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// ParseAttachmentIDs converts string UUIDs to uuid.UUID slice
func (r *CreateOrderRequest) ParseAttachmentIDs() ([]uuid.UUID, error) {
	return parseUUIDSlice(r.Attachments)
}

// ParseCategoryID converts string category ID to uuid.UUID pointer
func (r *CreateOrderRequest) ParseCategoryID() (*uuid.UUID, error) {
	if r.CategoryID == nil || *r.CategoryID == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*r.CategoryID)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// ParseAttachmentIDs converts string UUIDs to uuid.UUID slice
func (r *UpdateOrderRequest) ParseAttachmentIDs() ([]uuid.UUID, error) {
	return parseUUIDSlice(r.Attachments)
}

// ParseCategoryID converts string category ID to uuid.UUID pointer
func (r *UpdateOrderRequest) ParseCategoryID() (*uuid.UUID, error) {
	if r.CategoryID == nil || *r.CategoryID == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*r.CategoryID)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// ParseAttachmentIDs converts string UUIDs to uuid.UUID slice
func (r *SendMessageRequest) ParseAttachmentIDs() ([]uuid.UUID, error) {
	return parseUUIDSlice(r.Attachments)
}

// ParseParentMessageID converts string parent message ID to uuid.UUID pointer
func (r *SendMessageRequest) ParseParentMessageID() (*uuid.UUID, error) {
	if r.ParentMessageID == nil || *r.ParentMessageID == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*r.ParentMessageID)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// ParseMediaIDs converts string UUIDs to uuid.UUID slice
func (r *CreatePortfolioItemRequest) ParseMediaIDs() ([]uuid.UUID, error) {
	return parseUUIDSlice(r.MediaIDs)
}

// ParseMediaIDs converts string UUIDs to uuid.UUID slice
func (r *UpdatePortfolioItemRequest) ParseMediaIDs() ([]uuid.UUID, error) {
	return parseUUIDSlice(r.MediaIDs)
}

// ParsePhotoID converts string photo ID to uuid.UUID pointer
func (r *UpdateProfileRequest) ParsePhotoID() (*uuid.UUID, error) {
	if r.PhotoID == nil || *r.PhotoID == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*r.PhotoID)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// parseUUIDSlice is a helper to convert string slice to UUID slice
func parseUUIDSlice(strs []string) ([]uuid.UUID, error) {
	if strs == nil {
		return nil, nil
	}

	var uuids []uuid.UUID
	for _, str := range strs {
		if str == "" {
			continue
		}
		parsed, err := uuid.Parse(str)
		if err != nil {
			return nil, err
		}
		uuids = append(uuids, parsed)
	}
	return uuids, nil
}
