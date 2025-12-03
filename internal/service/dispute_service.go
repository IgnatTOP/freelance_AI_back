package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

var (
	ErrDisputeAlreadyExists = errors.New("dispute already exists for this order")
	ErrNotParticipant       = errors.New("user is not a participant of this escrow")
	ErrEscrowNotHeld        = errors.New("escrow is not in held status")
)

type DisputeService struct {
	disputeRepo *repository.DisputeRepository
	paymentRepo *repository.PaymentRepository
}

func NewDisputeService(dr *repository.DisputeRepository, pr *repository.PaymentRepository) *DisputeService {
	return &DisputeService{disputeRepo: dr, paymentRepo: pr}
}

func (s *DisputeService) CreateDispute(ctx context.Context, orderID, initiatorID uuid.UUID, reason string) (*models.Dispute, error) {
	// Проверяем escrow
	escrow, err := s.paymentRepo.GetEscrowByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if escrow.Status != models.EscrowStatusHeld {
		return nil, ErrEscrowNotHeld
	}
	if escrow.ClientID != initiatorID && escrow.FreelancerID != initiatorID {
		return nil, ErrNotParticipant
	}

	// Проверяем, нет ли уже спора
	existing, err := s.disputeRepo.GetByOrderID(ctx, orderID)
	if err == nil && existing != nil {
		return nil, ErrDisputeAlreadyExists
	}

	d := &models.Dispute{
		EscrowID:    escrow.ID,
		OrderID:     orderID,
		InitiatorID: initiatorID,
		Reason:      reason,
		Status:      models.DisputeStatusOpen,
	}
	if err := s.disputeRepo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *DisputeService) GetDispute(ctx context.Context, orderID uuid.UUID) (*models.Dispute, error) {
	return s.disputeRepo.GetByOrderID(ctx, orderID)
}

func (s *DisputeService) ListUserDisputes(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Dispute, error) {
	return s.disputeRepo.ListByUser(ctx, userID, limit, offset)
}
