package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
)

type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type AddReactionRequest struct {
	Emoji string `json:"emoji" binding:"required"`
}

type ConversationResponse struct {
	ID           uuid.UUID `json:"id"`
	OrderID      uuid.UUID `json:"order_id"`
	ClientID     uuid.UUID `json:"client_id"`
	FreelancerID uuid.UUID `json:"freelancer_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type MessageResponse struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Content        string    `json:"content"`
	IsEdited       bool      `json:"is_edited"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ReactionResponse struct {
	ID        uuid.UUID `json:"id"`
	MessageID uuid.UUID `json:"message_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

func ToConversationResponse(conv *entity.Conversation) ConversationResponse {
	return ConversationResponse{
		ID:           conv.ID,
		OrderID:      conv.OrderID,
		ClientID:     conv.ClientID,
		FreelancerID: conv.FreelancerID,
		CreatedAt:    conv.CreatedAt,
		UpdatedAt:    conv.UpdatedAt,
	}
}

func ToConversationResponses(convs []*entity.Conversation) []ConversationResponse {
	result := make([]ConversationResponse, len(convs))
	for i, conv := range convs {
		result[i] = ToConversationResponse(conv)
	}
	return result
}

func ToMessageResponse(msg *entity.Message) MessageResponse {
	return MessageResponse{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		IsEdited:       msg.IsEdited,
		CreatedAt:      msg.CreatedAt,
		UpdatedAt:      msg.UpdatedAt,
	}
}

func ToMessageResponses(msgs []*entity.Message) []MessageResponse {
	result := make([]MessageResponse, len(msgs))
	for i, msg := range msgs {
		result[i] = ToMessageResponse(msg)
	}
	return result
}

func ToReactionResponse(r *entity.MessageReaction) ReactionResponse {
	return ReactionResponse{
		ID:        r.ID,
		MessageID: r.MessageID,
		UserID:    r.UserID,
		Emoji:     r.Emoji,
		CreatedAt: r.CreatedAt,
	}
}
