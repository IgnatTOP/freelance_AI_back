package order_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/usecase/order"
)

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
	var result []*entity.Order
	for _, o := range m.orders {
		if o.ClientID == clientID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (m *mockOrderRepository) List(ctx context.Context, filter repository.OrderFilter) ([]*entity.Order, int, error) {
	var result []*entity.Order
	for _, o := range m.orders {
		result = append(result, o)
	}
	return result, len(result), nil
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

func TestCreateOrderUseCase_Success(t *testing.T) {
	repo := newMockOrderRepository()
	uc := order.NewCreateOrderUseCase(repo)

	input := order.CreateOrderInput{
		ClientID:    uuid.New(),
		Title:       "Test Order",
		Description: "Test Description",
		BudgetMin:   100,
		BudgetMax:   200,
	}

	result, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected order, got nil")
	}

	if result.Title != input.Title {
		t.Errorf("expected title %s, got %s", input.Title, result.Title)
	}

	if result.Description != input.Description {
		t.Errorf("expected description %s, got %s", input.Description, result.Description)
	}

	if result.Budget.Min.Amount != input.BudgetMin {
		t.Errorf("expected budget min %f, got %f", input.BudgetMin, result.Budget.Min.Amount)
	}
}

func TestCreateOrderUseCase_WithRequirements(t *testing.T) {
	repo := newMockOrderRepository()
	uc := order.NewCreateOrderUseCase(repo)

	input := order.CreateOrderInput{
		ClientID:    uuid.New(),
		Title:       "Test Order",
		Description: "Test Description",
		BudgetMin:   100,
		BudgetMax:   200,
		Requirements: []order.RequirementInput{
			{Skill: "Go", Level: "senior"},
			{Skill: "PostgreSQL", Level: "middle"},
		},
	}

	result, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Requirements) != 2 {
		t.Errorf("expected 2 requirements, got %d", len(result.Requirements))
	}
}

func TestCreateOrderUseCase_WithDeadline(t *testing.T) {
	repo := newMockOrderRepository()
	uc := order.NewCreateOrderUseCase(repo)

	deadline := time.Now().Add(24 * time.Hour)
	input := order.CreateOrderInput{
		ClientID:    uuid.New(),
		Title:       "Test Order",
		Description: "Test Description",
		BudgetMin:   100,
		BudgetMax:   200,
		DeadlineAt:  &deadline,
	}

	result, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DeadlineAt == nil {
		t.Fatal("expected deadline, got nil")
	}
}

func TestCreateOrderUseCase_EmptyTitle(t *testing.T) {
	repo := newMockOrderRepository()
	uc := order.NewCreateOrderUseCase(repo)

	input := order.CreateOrderInput{
		ClientID:    uuid.New(),
		Title:       "",
		Description: "Test Description",
		BudgetMin:   100,
		BudgetMax:   200,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestCreateOrderUseCase_InvalidBudget(t *testing.T) {
	repo := newMockOrderRepository()
	uc := order.NewCreateOrderUseCase(repo)

	input := order.CreateOrderInput{
		ClientID:    uuid.New(),
		Title:       "Test Order",
		Description: "Test Description",
		BudgetMin:   200,
		BudgetMax:   100,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid budget")
	}
}
