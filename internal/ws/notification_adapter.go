package ws

import (
	"context"

	"github.com/google/uuid"
)

// NotificationServiceAdapter адаптирует NotificationService для использования в Hub.
type NotificationServiceAdapter struct {
	service interface {
		CreateNotificationForWS(ctx context.Context, userID uuid.UUID, event string, data interface{}) error
	}
}

// NewNotificationServiceAdapter создаёт новый адаптер.
func NewNotificationServiceAdapter(service interface {
	CreateNotificationForWS(ctx context.Context, userID uuid.UUID, event string, data interface{}) error
}) *NotificationServiceAdapter {
	return &NotificationServiceAdapter{service: service}
}

// CreateNotification реализует интерфейс NotificationSaver.
func (a *NotificationServiceAdapter) CreateNotification(ctx context.Context, userID uuid.UUID, event string, data interface{}) error {
	return a.service.CreateNotificationForWS(ctx, userID, event, data)
}

