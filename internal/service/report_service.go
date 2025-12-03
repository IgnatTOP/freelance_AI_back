package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

var ErrInvalidReportTarget = errors.New("invalid target type")

type ReportService struct {
	repo *repository.ReportRepository
}

func NewReportService(r *repository.ReportRepository) *ReportService {
	return &ReportService{repo: r}
}

func (s *ReportService) CreateReport(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string, description *string) (*models.Report, error) {
	validTypes := map[string]bool{
		models.ReportTargetUser:    true,
		models.ReportTargetOrder:   true,
		models.ReportTargetMessage: true,
		models.ReportTargetReview:  true,
	}
	if !validTypes[targetType] {
		return nil, ErrInvalidReportTarget
	}

	r := &models.Report{
		ReporterID:  reporterID,
		TargetType:  targetType,
		TargetID:    targetID,
		Reason:      reason,
		Description: description,
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *ReportService) GetReport(ctx context.Context, id uuid.UUID) (*models.Report, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ReportService) ListMyReports(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Report, error) {
	return s.repo.ListByReporter(ctx, userID, limit, offset)
}
