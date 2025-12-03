package ai_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/usecase/ai"
)

type mockAIService struct {
	generateDescriptionResult string
	improveDescriptionResult  string
	summarizeResult           string
	generateProposalResult    string
	proposalFeedbackResult    string
	err                       error
}

func (m *mockAIService) SummarizeOrder(ctx context.Context, title, description string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.summarizeResult, nil
}

func (m *mockAIService) StreamSummarizeOrder(ctx context.Context, title, description string, onDelta func(chunk string) error) error {
	if m.err != nil {
		return m.err
	}
	return onDelta(m.summarizeResult)
}

func (m *mockAIService) GenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.generateDescriptionResult, nil
}

func (m *mockAIService) StreamGenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string, onDelta func(chunk string) error) error {
	if m.err != nil {
		return m.err
	}
	return onDelta(m.generateDescriptionResult)
}

func (m *mockAIService) ImproveOrderDescription(ctx context.Context, title, description string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.improveDescriptionResult, nil
}

func (m *mockAIService) StreamImproveOrderDescription(ctx context.Context, title, description string, onDelta func(chunk string) error) error {
	if m.err != nil {
		return m.err
	}
	return onDelta(m.improveDescriptionResult)
}

func (m *mockAIService) GenerateProposal(ctx context.Context, order *entity.Order, userSkills []string, userExperience string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.generateProposalResult, nil
}

func (m *mockAIService) StreamGenerateProposal(ctx context.Context, order *entity.Order, userSkills []string, userExperience string, onDelta func(chunk string) error) error {
	if m.err != nil {
		return m.err
	}
	return onDelta(m.generateProposalResult)
}

func (m *mockAIService) ProposalFeedback(ctx context.Context, order *entity.Order, coverLetter string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.proposalFeedbackResult, nil
}

func (m *mockAIService) StreamProposalFeedback(ctx context.Context, order *entity.Order, coverLetter string, onDelta func(chunk string) error) error {
	if m.err != nil {
		return m.err
	}
	return onDelta(m.proposalFeedbackResult)
}

func TestGenerateOrderDescriptionUseCase_Success(t *testing.T) {
	mockAI := &mockAIService{
		generateDescriptionResult: "Профессиональное описание заказа",
	}
	uc := ai.NewGenerateOrderDescriptionUseCase(mockAI)

	result, err := uc.Execute(context.Background(), "Test Title", "Brief description", []string{"Go", "PostgreSQL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Профессиональное описание заказа" {
		t.Errorf("expected 'Профессиональное описание заказа', got '%s'", result)
	}
}

func TestGenerateOrderDescriptionUseCase_NilService(t *testing.T) {
	uc := ai.NewGenerateOrderDescriptionUseCase(nil)

	_, err := uc.Execute(context.Background(), "Test Title", "Brief description", nil)
	if err == nil {
		t.Fatal("expected error for nil AI service")
	}
}

func TestImproveOrderDescriptionUseCase_Success(t *testing.T) {
	mockAI := &mockAIService{
		improveDescriptionResult: "Улучшенное описание заказа",
	}
	uc := ai.NewImproveOrderDescriptionUseCase(mockAI)

	result, err := uc.Execute(context.Background(), "Test Title", "Original description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Улучшенное описание заказа" {
		t.Errorf("expected 'Улучшенное описание заказа', got '%s'", result)
	}
}

func TestGenerateOrderDescriptionUseCase_Stream(t *testing.T) {
	mockAI := &mockAIService{
		generateDescriptionResult: "Streaming result",
	}
	uc := ai.NewGenerateOrderDescriptionUseCase(mockAI)

	var received string
	err := uc.ExecuteStream(context.Background(), "Test", "Brief", nil, func(chunk string) error {
		received = chunk
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received != "Streaming result" {
		t.Errorf("expected 'Streaming result', got '%s'", received)
	}
}

// Mock order repository for SummarizeOrderUseCase
type mockOrderRepoForAI struct {
	order *entity.Order
}

func (m *mockOrderRepoForAI) FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	return m.order, nil
}

func (m *mockOrderRepoForAI) FindByIDWithDetails(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	return m.order, nil
}

func (m *mockOrderRepoForAI) Update(ctx context.Context, order *entity.Order) error {
	return nil
}

// Implement other required methods with empty implementations
func (m *mockOrderRepoForAI) Create(ctx context.Context, order *entity.Order) error { return nil }
func (m *mockOrderRepoForAI) Delete(ctx context.Context, id uuid.UUID) error        { return nil }
func (m *mockOrderRepoForAI) FindByClientID(ctx context.Context, clientID uuid.UUID) ([]*entity.Order, error) {
	return nil, nil
}
func (m *mockOrderRepoForAI) List(ctx context.Context, filter interface{}) ([]*entity.Order, int, error) {
	return nil, 0, nil
}
func (m *mockOrderRepoForAI) CreateRequirement(ctx context.Context, req *entity.OrderRequirement) error {
	return nil
}
func (m *mockOrderRepoForAI) UpdateRequirements(ctx context.Context, orderID uuid.UUID, requirements []entity.OrderRequirement) error {
	return nil
}
func (m *mockOrderRepoForAI) FindRequirements(ctx context.Context, orderID uuid.UUID) ([]entity.OrderRequirement, error) {
	return nil, nil
}
func (m *mockOrderRepoForAI) CreateAttachment(ctx context.Context, att *entity.OrderAttachment) error {
	return nil
}
func (m *mockOrderRepoForAI) UpdateAttachments(ctx context.Context, orderID uuid.UUID, attachments []entity.OrderAttachment) error {
	return nil
}
func (m *mockOrderRepoForAI) FindAttachments(ctx context.Context, orderID uuid.UUID) ([]entity.OrderAttachment, error) {
	return nil, nil
}
