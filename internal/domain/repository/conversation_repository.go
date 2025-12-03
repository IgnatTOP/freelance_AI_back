package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
)

type ConversationRepository interface {
	Create(ctx context.Context, conv *entity.Conversation) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Conversation, error)
	FindByParticipants(ctx context.Context, orderID, clientID, freelancerID uuid.UUID) (*entity.Conversation, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Conversation, error)
}

type MessageRepository interface {
	Create(ctx context.Context, msg *entity.Message) error
	Update(ctx context.Context, msg *entity.Message) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error)
	FindByConversationID(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*entity.Message, error)
	GetLastMessage(ctx context.Context, conversationID uuid.UUID) (*entity.Message, error)
	
	AddReaction(ctx context.Context, reaction *entity.MessageReaction) error
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID) error
	GetReactions(ctx context.Context, messageID uuid.UUID) ([]*entity.MessageReaction, error)
}
