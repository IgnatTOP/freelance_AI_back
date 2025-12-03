package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

type VerificationService struct {
	repo *repository.VerificationRepository
}

func NewVerificationService(r *repository.VerificationRepository) *VerificationService {
	return &VerificationService{repo: r}
}

func (s *VerificationService) SendEmailCode(ctx context.Context, userID uuid.UUID) (string, error) {
	code := generateCode()
	expiresAt := time.Now().Add(15 * time.Minute)
	_, err := s.repo.CreateCode(ctx, userID, models.VerificationTypeEmail, code, expiresAt)
	if err != nil {
		return "", err
	}
	// TODO: отправить email с кодом
	return code, nil
}

func (s *VerificationService) SendPhoneCode(ctx context.Context, userID uuid.UUID) (string, error) {
	code := generateCode()
	expiresAt := time.Now().Add(5 * time.Minute)
	_, err := s.repo.CreateCode(ctx, userID, models.VerificationTypePhone, code, expiresAt)
	if err != nil {
		return "", err
	}
	// TODO: отправить SMS с кодом
	return code, nil
}

func (s *VerificationService) VerifyCode(ctx context.Context, userID uuid.UUID, codeType, code string) (bool, error) {
	return s.repo.VerifyCode(ctx, userID, codeType, code)
}

func (s *VerificationService) GetStatus(ctx context.Context, userID uuid.UUID) (map[string]bool, error) {
	email, phone, identity, err := s.repo.GetUserVerificationStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	return map[string]bool{
		"email_verified":    email,
		"phone_verified":    phone,
		"identity_verified": identity,
	}, nil
}

func generateCode() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])*10000+int(b[1])*100+int(b[2])%100)[:6]
}
