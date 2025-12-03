package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/valueobject"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type Order struct {
	ID                              uuid.UUID
	ClientID                        uuid.UUID
	Title                           string
	Description                     string
	Budget                          valueobject.Budget
	Status                          valueobject.OrderStatus
	DeadlineAt                      *time.Time
	AISummary                       *string
	BestRecommendationProposalID    *uuid.UUID
	BestRecommendationJustification *string
	AIAnalysisUpdatedAt             *time.Time
	CreatedAt                       time.Time
	UpdatedAt                       time.Time
	
	Requirements []OrderRequirement
	Attachments  []OrderAttachment
}

type OrderRequirement struct {
	ID      uuid.UUID
	OrderID uuid.UUID
	Skill   string
	Level   string
}

type OrderAttachment struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	MediaID   uuid.UUID
	CreatedAt time.Time
}

func NewOrder(clientID uuid.UUID, title, description string, budgetMin, budgetMax float64, deadline *time.Time) (*Order, error) {
	if title == "" {
		return nil, apperror.New(apperror.ErrCodeValidation, "название заказа обязательно")
	}
	if description == "" {
		return nil, apperror.New(apperror.ErrCodeValidation, "описание заказа обязательно")
	}
	
	budget, err := valueobject.NewBudget(budgetMin, budgetMax)
	if err != nil {
		return nil, err
	}
	
	if deadline != nil && deadline.Before(time.Now()) {
		return nil, apperror.New(apperror.ErrCodeValidation, "дедлайн не может быть в прошлом")
	}
	
	return &Order{
		ID:          uuid.New(),
		ClientID:    clientID,
		Title:       title,
		Description: description,
		Budget:      budget,
		Status:      valueobject.OrderStatusDraft,
		DeadlineAt:  deadline,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (o *Order) Publish() error {
	if !o.Status.CanTransitionTo(valueobject.OrderStatusPublished) {
		return apperror.New(apperror.ErrCodeBadRequest, "невозможно опубликовать заказ в текущем статусе")
	}
	o.Status = valueobject.OrderStatusPublished
	o.UpdatedAt = time.Now()
	return nil
}

func (o *Order) StartWork() error {
	if !o.Status.CanTransitionTo(valueobject.OrderStatusInProgress) {
		return apperror.New(apperror.ErrCodeBadRequest, "невозможно начать работу в текущем статусе")
	}
	o.Status = valueobject.OrderStatusInProgress
	o.UpdatedAt = time.Now()
	return nil
}

func (o *Order) Complete() error {
	if !o.Status.CanTransitionTo(valueobject.OrderStatusCompleted) {
		return apperror.New(apperror.ErrCodeBadRequest, "невозможно завершить заказ в текущем статусе")
	}
	o.Status = valueobject.OrderStatusCompleted
	o.UpdatedAt = time.Now()
	return nil
}

func (o *Order) Cancel() error {
	if !o.Status.CanTransitionTo(valueobject.OrderStatusCancelled) {
		return apperror.New(apperror.ErrCodeBadRequest, "невозможно отменить заказ в текущем статусе")
	}
	o.Status = valueobject.OrderStatusCancelled
	o.UpdatedAt = time.Now()
	return nil
}

func (o *Order) Update(title, description string, budgetMin, budgetMax float64, deadline *time.Time) error {
	if title != "" {
		o.Title = title
	}
	if description != "" {
		o.Description = description
	}
	
	if budgetMin > 0 || budgetMax > 0 {
		budget, err := valueobject.NewBudget(budgetMin, budgetMax)
		if err != nil {
			return err
		}
		o.Budget = budget
	}
	
	if deadline != nil {
		if deadline.Before(time.Now()) {
			return apperror.New(apperror.ErrCodeValidation, "дедлайн не может быть в прошлом")
		}
		o.DeadlineAt = deadline
	}
	
	o.UpdatedAt = time.Now()
	return nil
}

func (o *Order) SetAISummary(summary string) {
	o.AISummary = &summary
	now := time.Now()
	o.AIAnalysisUpdatedAt = &now
	o.UpdatedAt = now
}

func (o *Order) SetBestRecommendation(proposalID uuid.UUID, justification string) {
	o.BestRecommendationProposalID = &proposalID
	o.BestRecommendationJustification = &justification
	now := time.Now()
	o.AIAnalysisUpdatedAt = &now
	o.UpdatedAt = now
}

func (o *Order) IsOwnedBy(userID uuid.UUID) bool {
	return o.ClientID == userID
}
