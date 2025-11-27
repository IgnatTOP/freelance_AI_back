package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// NotificationRepository описывает взаимодействие сервиса с хранилищем уведомлений.
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]models.Notification, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
}

// NotificationService содержит бизнес-логику работы с уведомлениями.
type NotificationService struct {
	repo NotificationRepository
}

// NewNotificationService создаёт новый сервис уведомлений.
func NewNotificationService(repo NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// CreateNotification создаёт новое уведомление.
func (s *NotificationService) CreateNotification(ctx context.Context, userID uuid.UUID, event string, data interface{}) (*models.Notification, error) {
	payload := map[string]interface{}{
		"event": event,
		"data":  data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("notification service: marshal payload %w", err)
	}

	notification := &models.Notification{
		UserID:  userID,
		Payload: payloadBytes,
		IsRead:  false,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return nil, err
	}

	return notification, nil
}

// GetNotification возвращает уведомление по идентификатору.
func (s *NotificationService) GetNotification(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	return s.repo.GetByID(ctx, id)
}

// ListNotifications возвращает список уведомлений пользователя.
func (s *NotificationService) ListNotifications(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]models.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.List(ctx, userID, limit, offset, unreadOnly)
}

// MarkAsRead отмечает уведомление как прочитанное.
func (s *NotificationService) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if notification.UserID != userID {
		return fmt.Errorf("notification service: у вас нет прав на это уведомление")
	}

	return s.repo.MarkAsRead(ctx, id)
}

// MarkAllAsRead отмечает все уведомления пользователя как прочитанные.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

// DeleteNotification удаляет уведомление.
func (s *NotificationService) DeleteNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if notification.UserID != userID {
		return fmt.Errorf("notification service: у вас нет прав на это уведомление")
	}

	return s.repo.Delete(ctx, id)
}

// CountUnread возвращает количество непрочитанных уведомлений.
func (s *NotificationService) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

// CreateNotificationForWS создаёт уведомление (для использования в WebSocket hub).
func (s *NotificationService) CreateNotificationForWS(ctx context.Context, userID uuid.UUID, event string, data interface{}) error {
	_, err := s.CreateNotification(ctx, userID, event, data)
	return err
}

