package conversation

import (
	"context"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type GetOrCreateConversationUseCase struct {
	convRepo  repository.ConversationRepository
	orderRepo repository.OrderRepository
}

func NewGetOrCreateConversationUseCase(convRepo repository.ConversationRepository, orderRepo repository.OrderRepository) *GetOrCreateConversationUseCase {
	return &GetOrCreateConversationUseCase{convRepo: convRepo, orderRepo: orderRepo}
}

func (uc *GetOrCreateConversationUseCase) Execute(ctx context.Context, orderID, userID, participantID uuid.UUID) (*entity.Conversation, error) {
	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	clientID, freelancerID := order.ClientID, participantID
	if userID != order.ClientID {
		clientID, freelancerID = participantID, userID
	}

	conv, err := uc.convRepo.FindByParticipants(ctx, orderID, clientID, freelancerID)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	conv, err = entity.NewConversation(orderID, clientID, freelancerID)
	if err != nil {
		return nil, err
	}

	if err := uc.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

type ListMyConversationsUseCase struct {
	convRepo repository.ConversationRepository
}

func NewListMyConversationsUseCase(convRepo repository.ConversationRepository) *ListMyConversationsUseCase {
	return &ListMyConversationsUseCase{convRepo: convRepo}
}

func (uc *ListMyConversationsUseCase) Execute(ctx context.Context, userID uuid.UUID) ([]*entity.Conversation, error) {
	return uc.convRepo.FindByUserID(ctx, userID)
}

type SendMessageUseCase struct {
	convRepo repository.ConversationRepository
	msgRepo  repository.MessageRepository
}

func NewSendMessageUseCase(convRepo repository.ConversationRepository, msgRepo repository.MessageRepository) *SendMessageUseCase {
	return &SendMessageUseCase{convRepo: convRepo, msgRepo: msgRepo}
}

func (uc *SendMessageUseCase) Execute(ctx context.Context, conversationID, senderID uuid.UUID, content string) (*entity.Message, error) {
	conv, err := uc.convRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if !conv.IsParticipant(senderID) {
		return nil, apperror.ErrForbidden
	}

	msg, err := entity.NewMessage(conversationID, senderID, content)
	if err != nil {
		return nil, err
	}

	if err := uc.msgRepo.Create(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

type ListMessagesUseCase struct {
	convRepo repository.ConversationRepository
	msgRepo  repository.MessageRepository
}

func NewListMessagesUseCase(convRepo repository.ConversationRepository, msgRepo repository.MessageRepository) *ListMessagesUseCase {
	return &ListMessagesUseCase{convRepo: convRepo, msgRepo: msgRepo}
}

func (uc *ListMessagesUseCase) Execute(ctx context.Context, conversationID, userID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	conv, err := uc.convRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if !conv.IsParticipant(userID) {
		return nil, apperror.ErrForbidden
	}

	return uc.msgRepo.FindByConversationID(ctx, conversationID, limit, offset)
}

type UpdateMessageUseCase struct {
	msgRepo repository.MessageRepository
}

func NewUpdateMessageUseCase(msgRepo repository.MessageRepository) *UpdateMessageUseCase {
	return &UpdateMessageUseCase{msgRepo: msgRepo}
}

func (uc *UpdateMessageUseCase) Execute(ctx context.Context, messageID, userID uuid.UUID, content string) (*entity.Message, error) {
	msg, err := uc.msgRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if !msg.IsOwnedBy(userID) {
		return nil, apperror.ErrForbidden
	}

	if err := msg.Update(content); err != nil {
		return nil, err
	}

	if err := uc.msgRepo.Update(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

type DeleteMessageUseCase struct {
	msgRepo repository.MessageRepository
}

func NewDeleteMessageUseCase(msgRepo repository.MessageRepository) *DeleteMessageUseCase {
	return &DeleteMessageUseCase{msgRepo: msgRepo}
}

func (uc *DeleteMessageUseCase) Execute(ctx context.Context, messageID, userID uuid.UUID) error {
	msg, err := uc.msgRepo.FindByID(ctx, messageID)
	if err != nil {
		return err
	}

	if !msg.IsOwnedBy(userID) {
		return apperror.ErrForbidden
	}

	return uc.msgRepo.Delete(ctx, messageID)
}

type AddReactionUseCase struct {
	msgRepo repository.MessageRepository
}

func NewAddReactionUseCase(msgRepo repository.MessageRepository) *AddReactionUseCase {
	return &AddReactionUseCase{msgRepo: msgRepo}
}

func (uc *AddReactionUseCase) Execute(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*entity.MessageReaction, error) {
	reaction, err := entity.NewMessageReaction(messageID, userID, emoji)
	if err != nil {
		return nil, err
	}

	if err := uc.msgRepo.AddReaction(ctx, reaction); err != nil {
		return nil, err
	}
	return reaction, nil
}

type RemoveReactionUseCase struct {
	msgRepo repository.MessageRepository
}

func NewRemoveReactionUseCase(msgRepo repository.MessageRepository) *RemoveReactionUseCase {
	return &RemoveReactionUseCase{msgRepo: msgRepo}
}

func (uc *RemoveReactionUseCase) Execute(ctx context.Context, messageID, userID uuid.UUID) error {
	return uc.msgRepo.RemoveReaction(ctx, messageID, userID)
}
