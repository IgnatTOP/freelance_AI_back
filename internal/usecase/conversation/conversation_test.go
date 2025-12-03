package conversation_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/domain/valueobject"
	"github.com/ignatzorin/freelance-backend/internal/usecase/conversation"
)

type mockConversationRepository struct {
	conversations map[uuid.UUID]*entity.Conversation
}

func newMockConversationRepository() *mockConversationRepository {
	return &mockConversationRepository{conversations: make(map[uuid.UUID]*entity.Conversation)}
}

func (m *mockConversationRepository) Create(ctx context.Context, c *entity.Conversation) error {
	m.conversations[c.ID] = c
	return nil
}

func (m *mockConversationRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Conversation, error) {
	if c, ok := m.conversations[id]; ok {
		return c, nil
	}
	return nil, nil
}

func (m *mockConversationRepository) FindByParticipants(ctx context.Context, orderID, clientID, freelancerID uuid.UUID) (*entity.Conversation, error) {
	for _, c := range m.conversations {
		if c.OrderID == orderID && c.ClientID == clientID && c.FreelancerID == freelancerID {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockConversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Conversation, error) {
	var result []*entity.Conversation
	for _, c := range m.conversations {
		if c.ClientID == userID || c.FreelancerID == userID {
			result = append(result, c)
		}
	}
	return result, nil
}

type mockMessageRepository struct {
	messages map[uuid.UUID]*entity.Message
}

func newMockMessageRepository() *mockMessageRepository {
	return &mockMessageRepository{messages: make(map[uuid.UUID]*entity.Message)}
}

func (m *mockMessageRepository) Create(ctx context.Context, msg *entity.Message) error {
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockMessageRepository) Update(ctx context.Context, msg *entity.Message) error {
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockMessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.messages, id)
	return nil
}

func (m *mockMessageRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error) {
	if msg, ok := m.messages[id]; ok {
		return msg, nil
	}
	return nil, nil
}

func (m *mockMessageRepository) FindByConversationID(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	var result []*entity.Message
	for _, msg := range m.messages {
		if msg.ConversationID == conversationID {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *mockMessageRepository) GetLastMessage(ctx context.Context, conversationID uuid.UUID) (*entity.Message, error) {
	return nil, nil
}

func (m *mockMessageRepository) AddReaction(ctx context.Context, reaction *entity.MessageReaction) error {
	return nil
}

func (m *mockMessageRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID) error {
	return nil
}

func (m *mockMessageRepository) GetReactions(ctx context.Context, messageID uuid.UUID) ([]*entity.MessageReaction, error) {
	return nil, nil
}

type mockOrderRepository struct {
	orders map[uuid.UUID]*entity.Order
}

func newMockOrderRepository() *mockOrderRepository {
	return &mockOrderRepository{orders: make(map[uuid.UUID]*entity.Order)}
}

func (m *mockOrderRepository) Create(ctx context.Context, o *entity.Order) error {
	return nil
}

func (m *mockOrderRepository) Update(ctx context.Context, o *entity.Order) error {
	return nil
}

func (m *mockOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	if o, ok := m.orders[id]; ok {
		return o, nil
	}
	return nil, nil
}

func (m *mockOrderRepository) FindByIDWithDetails(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	return m.FindByID(ctx, id)
}

func (m *mockOrderRepository) FindByClientID(ctx context.Context, clientID uuid.UUID) ([]*entity.Order, error) {
	return nil, nil
}

func (m *mockOrderRepository) List(ctx context.Context, filter repository.OrderFilter) ([]*entity.Order, int, error) {
	return nil, 0, nil
}

func (m *mockOrderRepository) CreateRequirement(ctx context.Context, req *entity.OrderRequirement) error {
	return nil
}

func (m *mockOrderRepository) UpdateRequirements(ctx context.Context, orderID uuid.UUID, requirements []entity.OrderRequirement) error {
	return nil
}

func (m *mockOrderRepository) FindRequirements(ctx context.Context, orderID uuid.UUID) ([]entity.OrderRequirement, error) {
	return nil, nil
}

func (m *mockOrderRepository) CreateAttachment(ctx context.Context, att *entity.OrderAttachment) error {
	return nil
}

func (m *mockOrderRepository) UpdateAttachments(ctx context.Context, orderID uuid.UUID, attachments []entity.OrderAttachment) error {
	return nil
}

func (m *mockOrderRepository) FindAttachments(ctx context.Context, orderID uuid.UUID) ([]entity.OrderAttachment, error) {
	return nil, nil
}

func createTestOrder(clientID uuid.UUID) *entity.Order {
	budget, _ := valueobject.NewBudget(100, 500)
	return &entity.Order{
		ID:          uuid.New(),
		ClientID:    clientID,
		Title:       "Test Order",
		Description: "Test Description",
		Budget:      budget,
		Status:      valueobject.OrderStatusPublished,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestSendMessageUseCase_Success(t *testing.T) {
	convRepo := newMockConversationRepository()
	msgRepo := newMockMessageRepository()
	uc := conversation.NewSendMessageUseCase(convRepo, msgRepo)

	clientID := uuid.New()
	freelancerID := uuid.New()
	conv := &entity.Conversation{
		ID:           uuid.New(),
		OrderID:      uuid.New(),
		ClientID:     clientID,
		FreelancerID: freelancerID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	convRepo.conversations[conv.ID] = conv

	msg, err := uc.Execute(context.Background(), conv.ID, clientID, "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg == nil {
		t.Fatal("expected message, got nil")
	}

	if msg.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got '%s'", msg.Content)
	}
}

func TestSendMessageUseCase_NotParticipant(t *testing.T) {
	convRepo := newMockConversationRepository()
	msgRepo := newMockMessageRepository()
	uc := conversation.NewSendMessageUseCase(convRepo, msgRepo)

	clientID := uuid.New()
	freelancerID := uuid.New()
	otherUserID := uuid.New()
	conv := &entity.Conversation{
		ID:           uuid.New(),
		OrderID:      uuid.New(),
		ClientID:     clientID,
		FreelancerID: freelancerID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	convRepo.conversations[conv.ID] = conv

	_, err := uc.Execute(context.Background(), conv.ID, otherUserID, "Hello!")
	if err == nil {
		t.Fatal("expected error for non-participant")
	}
}

func TestUpdateMessageUseCase_Success(t *testing.T) {
	msgRepo := newMockMessageRepository()
	uc := conversation.NewUpdateMessageUseCase(msgRepo)

	senderID := uuid.New()
	msg := &entity.Message{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		SenderID:       senderID,
		Content:        "Original",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	msgRepo.messages[msg.ID] = msg

	updated, err := uc.Execute(context.Background(), msg.ID, senderID, "Updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Content != "Updated" {
		t.Errorf("expected content 'Updated', got '%s'", updated.Content)
	}

	if !updated.IsEdited {
		t.Error("expected IsEdited to be true")
	}
}

func TestUpdateMessageUseCase_NotOwner(t *testing.T) {
	msgRepo := newMockMessageRepository()
	uc := conversation.NewUpdateMessageUseCase(msgRepo)

	senderID := uuid.New()
	otherUserID := uuid.New()
	msg := &entity.Message{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		SenderID:       senderID,
		Content:        "Original",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	msgRepo.messages[msg.ID] = msg

	_, err := uc.Execute(context.Background(), msg.ID, otherUserID, "Updated")
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
}

func TestDeleteMessageUseCase_Success(t *testing.T) {
	msgRepo := newMockMessageRepository()
	uc := conversation.NewDeleteMessageUseCase(msgRepo)

	senderID := uuid.New()
	msg := &entity.Message{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		SenderID:       senderID,
		Content:        "To delete",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	msgRepo.messages[msg.ID] = msg

	err := uc.Execute(context.Background(), msg.ID, senderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := msgRepo.messages[msg.ID]; ok {
		t.Error("expected message to be deleted")
	}
}

func TestListMyConversationsUseCase_Success(t *testing.T) {
	convRepo := newMockConversationRepository()
	uc := conversation.NewListMyConversationsUseCase(convRepo)

	userID := uuid.New()
	conv1 := &entity.Conversation{
		ID:           uuid.New(),
		OrderID:      uuid.New(),
		ClientID:     userID,
		FreelancerID: uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	conv2 := &entity.Conversation{
		ID:           uuid.New(),
		OrderID:      uuid.New(),
		ClientID:     uuid.New(),
		FreelancerID: userID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	convRepo.conversations[conv1.ID] = conv1
	convRepo.conversations[conv2.ID] = conv2

	result, err := uc.Execute(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(result))
	}
}
