package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// ErrNotificationNotFound возвращается, когда уведомление не найдено.
var ErrNotificationNotFound = errors.New("notification not found")

// NotificationRepository отвечает за работу с уведомлениями.
type NotificationRepository struct {
	db *sqlx.DB
}

// NewNotificationRepository создаёт экземпляр репозитория.
func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create создаёт новое уведомление.
func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	query := `
		INSERT INTO notifications (user_id, payload, is_read)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	if err := r.db.QueryRowxContext(
		ctx,
		query,
		notification.UserID,
		notification.Payload,
		notification.IsRead,
	).Scan(&notification.ID, &notification.CreatedAt); err != nil {
		return fmt.Errorf("notification repository: create %w", err)
	}

	return nil
}

// GetByID возвращает уведомление по идентификатору.
func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	var notification models.Notification
	if err := r.db.GetContext(ctx, &notification, `SELECT * FROM notifications WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("notification repository: get by id %w", err)
	}

	return &notification, nil
}

// List возвращает список уведомлений пользователя с пагинацией.
func (r *NotificationRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]models.Notification, error) {
	query := `
		SELECT * FROM notifications
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argIndex := 2

	if unreadOnly {
		query += fmt.Sprintf(" AND is_read = FALSE")
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	var notifications []models.Notification
	if err := r.db.SelectContext(ctx, &notifications, query, args...); err != nil {
		return nil, fmt.Errorf("notification repository: list %w", err)
	}

	return notifications, nil
}

// MarkAsRead отмечает уведомление как прочитанное.
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `UPDATE notifications SET is_read = TRUE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("notification repository: mark as read %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("notification repository: mark as read rows affected %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// MarkAllAsRead отмечает все уведомления пользователя как прочитанные.
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`, userID)
	if err != nil {
		return fmt.Errorf("notification repository: mark all as read %w", err)
	}

	return nil
}

// Delete удаляет уведомление.
func (r *NotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM notifications WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("notification repository: delete %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("notification repository: delete rows affected %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// CountUnread возвращает количество непрочитанных уведомлений пользователя.
func (r *NotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID); err != nil {
		return 0, fmt.Errorf("notification repository: count unread %w", err)
	}

	return count, nil
}

