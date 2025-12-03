package proposal_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/domain/valueobject"
	"github.com/ignatzorin/freelance-backend/internal/usecase/proposal"
)

type mockProposalRepository struct {
	proposals map[uuid.UUID]*entity.Proposal
}

func newMockProposalRepository() *mockProposalRepository {
	return &mockProposalRepository{proposals: make(map[uuid.UUID]*entity.Proposal)}
}

func (m *mockProposalRepository) Create(ctx context.Context, p *entity.Proposal) error {
	m.proposals[p.ID] = p
	return nil
}

func (m *mockProposalRepository) Update(ctx context.Context, p *entity.Proposal) error {
	m.proposals[p.ID] = p
	return nil
}

func (m *mockProposalRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Proposal, error) {
	if p, ok := m.proposals[id]; ok {
		return p, nil
	}
	return nil, nil
}

func (m *mockProposalRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.Proposal, error) {
	var result []*entity.Proposal
	for _, p := range m.proposals {
		if p.OrderID == orderID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockProposalRepository) FindByFreelancerID(ctx context.Context, freelancerID uuid.UUID) ([]*entity.Proposal, error) {
	var result []*entity.Proposal
	for _, p := range m.proposals {
		if p.FreelancerID == freelancerID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockProposalRepository) FindByOrderAndFreelancer(ctx context.Context, orderID, freelancerID uuid.UUID) (*entity.Proposal, error) {
	for _, p := range m.proposals {
		if p.OrderID == orderID && p.FreelancerID == freelancerID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockProposalRepository) GetLastUpdateTime(ctx context.Context, orderID uuid.UUID) (*time.Time, error) {
	return nil, nil
}

type mockOrderRepository struct {
	orders map[uuid.UUID]*entity.Order
}

func newMockOrderRepository() *mockOrderRepository {
	return &mockOrderRepository{orders: make(map[uuid.UUID]*entity.Order)}
}

func (m *mockOrderRepository) Create(ctx context.Context, o *entity.Order) error {
	m.orders[o.ID] = o
	return nil
}

func (m *mockOrderRepository) Update(ctx context.Context, o *entity.Order) error {
	m.orders[o.ID] = o
	return nil
}

func (m *mockOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.orders, id)
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

func TestCreateProposalUseCase_Success(t *testing.T) {
	proposalRepo := newMockProposalRepository()
	orderRepo := newMockOrderRepository()
	uc := proposal.NewCreateProposalUseCase(proposalRepo, orderRepo)

	clientID := uuid.New()
	freelancerID := uuid.New()
	order := createTestOrder(clientID)
	orderRepo.orders[order.ID] = order

	input := proposal.CreateProposalInput{
		OrderID:        order.ID,
		FreelancerID:   freelancerID,
		CoverLetter:    "I am interested in this project",
		ProposedBudget: 150,
	}

	result, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected proposal, got nil")
	}

	if result.CoverLetter != input.CoverLetter {
		t.Errorf("expected cover letter %s, got %s", input.CoverLetter, result.CoverLetter)
	}

	if result.ProposedBudget != input.ProposedBudget {
		t.Errorf("expected budget %f, got %f", input.ProposedBudget, result.ProposedBudget)
	}
}

func TestCreateProposalUseCase_OwnOrder(t *testing.T) {
	proposalRepo := newMockProposalRepository()
	orderRepo := newMockOrderRepository()
	uc := proposal.NewCreateProposalUseCase(proposalRepo, orderRepo)

	clientID := uuid.New()
	order := createTestOrder(clientID)
	orderRepo.orders[order.ID] = order

	input := proposal.CreateProposalInput{
		OrderID:        order.ID,
		FreelancerID:   clientID,
		CoverLetter:    "I am interested",
		ProposedBudget: 150,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for own order")
	}
}

func TestCreateProposalUseCase_DuplicateProposal(t *testing.T) {
	proposalRepo := newMockProposalRepository()
	orderRepo := newMockOrderRepository()
	uc := proposal.NewCreateProposalUseCase(proposalRepo, orderRepo)

	clientID := uuid.New()
	freelancerID := uuid.New()
	order := createTestOrder(clientID)
	orderRepo.orders[order.ID] = order

	existingProposal := &entity.Proposal{
		ID:           uuid.New(),
		OrderID:      order.ID,
		FreelancerID: freelancerID,
		CoverLetter:  "First proposal",
		Status:       valueobject.ProposalStatusPending,
	}
	proposalRepo.proposals[existingProposal.ID] = existingProposal

	input := proposal.CreateProposalInput{
		OrderID:        order.ID,
		FreelancerID:   freelancerID,
		CoverLetter:    "Second proposal",
		ProposedBudget: 150,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for duplicate proposal")
	}
}

func TestCreateProposalUseCase_BudgetOutOfRange(t *testing.T) {
	proposalRepo := newMockProposalRepository()
	orderRepo := newMockOrderRepository()
	uc := proposal.NewCreateProposalUseCase(proposalRepo, orderRepo)

	clientID := uuid.New()
	freelancerID := uuid.New()
	order := createTestOrder(clientID)
	orderRepo.orders[order.ID] = order

	input := proposal.CreateProposalInput{
		OrderID:        order.ID,
		FreelancerID:   freelancerID,
		CoverLetter:    "I am interested",
		ProposedBudget: 1000,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for budget out of range")
	}
}
