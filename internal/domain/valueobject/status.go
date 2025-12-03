package valueobject

import "github.com/ignatzorin/freelance-backend/internal/pkg/apperror"

type OrderStatus string

const (
	OrderStatusDraft      OrderStatus = "draft"
	OrderStatusPublished  OrderStatus = "published"
	OrderStatusInProgress OrderStatus = "in_progress"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusDraft, OrderStatusPublished, OrderStatusInProgress, OrderStatusCompleted, OrderStatusCancelled:
		return true
	}
	return false
}

func (s OrderStatus) CanTransitionTo(newStatus OrderStatus) bool {
	transitions := map[OrderStatus][]OrderStatus{
		OrderStatusDraft:      {OrderStatusPublished, OrderStatusCancelled},
		OrderStatusPublished:  {OrderStatusInProgress, OrderStatusCancelled},
		OrderStatusInProgress: {OrderStatusCompleted, OrderStatusCancelled},
		OrderStatusCompleted:  {},
		OrderStatusCancelled:  {},
	}
	
	allowed, ok := transitions[s]
	if !ok {
		return false
	}
	
	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}
	return false
}

func NewOrderStatus(status string) (OrderStatus, error) {
	s := OrderStatus(status)
	if !s.IsValid() {
		return "", apperror.New(apperror.ErrCodeValidation, "некорректный статус заказа")
	}
	return s, nil
}

type ProposalStatus string

const (
	ProposalStatusPending  ProposalStatus = "pending"
	ProposalStatusAccepted ProposalStatus = "accepted"
	ProposalStatusRejected ProposalStatus = "rejected"
)

func (s ProposalStatus) IsValid() bool {
	switch s {
	case ProposalStatusPending, ProposalStatusAccepted, ProposalStatusRejected:
		return true
	}
	return false
}

func NewProposalStatus(status string) (ProposalStatus, error) {
	s := ProposalStatus(status)
	if !s.IsValid() {
		return "", apperror.New(apperror.ErrCodeValidation, "некорректный статус предложения")
	}
	return s, nil
}
