package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/valueobject"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type Proposal struct {
	ID                      uuid.UUID
	OrderID                 uuid.UUID
	FreelancerID            uuid.UUID
	CoverLetter             string
	ProposedBudget          float64
	ProposedDeadline        *time.Time
	Status                  valueobject.ProposalStatus
	AIFeedback              *string
	AIAnalysisForClient     *string
	AIAnalysisForClientAt   *time.Time
	CompletedByFreelancerAt *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func NewProposal(orderID, freelancerID uuid.UUID, coverLetter string, proposedBudget float64, proposedDeadline *time.Time) (*Proposal, error) {
	if coverLetter == "" {
		return nil, apperror.New(apperror.ErrCodeValidation, "сопроводительное письмо обязательно")
	}
	if proposedBudget <= 0 {
		return nil, apperror.New(apperror.ErrCodeValidation, "предложенный бюджет должен быть положительным")
	}
	if proposedDeadline != nil && proposedDeadline.Before(time.Now()) {
		return nil, apperror.New(apperror.ErrCodeValidation, "предложенный дедлайн не может быть в прошлом")
	}
	
	return &Proposal{
		ID:               uuid.New(),
		OrderID:          orderID,
		FreelancerID:     freelancerID,
		CoverLetter:      coverLetter,
		ProposedBudget:   proposedBudget,
		ProposedDeadline: proposedDeadline,
		Status:           valueobject.ProposalStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil
}

func (p *Proposal) Accept() error {
	if p.Status != valueobject.ProposalStatusPending {
		return apperror.New(apperror.ErrCodeBadRequest, "можно принять только ожидающее предложение")
	}
	p.Status = valueobject.ProposalStatusAccepted
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Proposal) Reject() error {
	if p.Status != valueobject.ProposalStatusPending {
		return apperror.New(apperror.ErrCodeBadRequest, "можно отклонить только ожидающее предложение")
	}
	p.Status = valueobject.ProposalStatusRejected
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Proposal) MarkCompletedByFreelancer() {
	now := time.Now()
	p.CompletedByFreelancerAt = &now
	p.UpdatedAt = now
}

func (p *Proposal) SetAIFeedback(feedback string) {
	p.AIFeedback = &feedback
	p.UpdatedAt = time.Now()
}

func (p *Proposal) SetAIAnalysisForClient(analysis string) {
	p.AIAnalysisForClient = &analysis
	now := time.Now()
	p.AIAnalysisForClientAt = &now
	p.UpdatedAt = now
}

func (p *Proposal) IsOwnedBy(userID uuid.UUID) bool {
	return p.FreelancerID == userID
}

func (p *Proposal) IsPending() bool {
	return p.Status == valueobject.ProposalStatusPending
}

func (p *Proposal) IsAccepted() bool {
	return p.Status == valueobject.ProposalStatusAccepted
}
