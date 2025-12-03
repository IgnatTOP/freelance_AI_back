package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type Conversation struct {
	ID           uuid.UUID
	OrderID      uuid.UUID
	ClientID     uuid.UUID
	FreelancerID uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewConversation(orderID, clientID, freelancerID uuid.UUID) (*Conversation, error) {
	if clientID == freelancerID {
		return nil, apperror.New(apperror.ErrCodeValidation, "нельзя создать беседу с самим собой")
	}
	return &Conversation{
		ID:           uuid.New(),
		OrderID:      orderID,
		ClientID:     clientID,
		FreelancerID: freelancerID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (c *Conversation) IsParticipant(userID uuid.UUID) bool {
	return c.ClientID == userID || c.FreelancerID == userID
}

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        string
	IsEdited       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewMessage(conversationID, senderID uuid.UUID, content string) (*Message, error) {
	if content == "" {
		return nil, apperror.New(apperror.ErrCodeValidation, "сообщение не может быть пустым")
	}
	return &Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
		IsEdited:       false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (m *Message) Update(content string) error {
	if content == "" {
		return apperror.New(apperror.ErrCodeValidation, "сообщение не может быть пустым")
	}
	m.Content = content
	m.IsEdited = true
	m.UpdatedAt = time.Now()
	return nil
}

func (m *Message) IsOwnedBy(userID uuid.UUID) bool {
	return m.SenderID == userID
}

type MessageReaction struct {
	ID        uuid.UUID
	MessageID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
	CreatedAt time.Time
}

func NewMessageReaction(messageID, userID uuid.UUID, emoji string) (*MessageReaction, error) {
	if emoji == "" {
		return nil, apperror.New(apperror.ErrCodeValidation, "emoji обязателен")
	}
	return &MessageReaction{
		ID:        uuid.New(),
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: time.Now(),
	}, nil
}
