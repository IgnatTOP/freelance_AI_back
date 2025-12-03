package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

type OrderHistoryRepository struct {
	db *sqlx.DB
}

func NewOrderHistoryRepository(db *sqlx.DB) *OrderHistoryRepository {
	return &OrderHistoryRepository{db: db}
}

func (r *OrderHistoryRepository) Add(ctx context.Context, orderID uuid.UUID, userID *uuid.UUID, action string, oldValue, newValue interface{}) error {
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO order_history (order_id, user_id, action, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5)
	`, orderID, userID, action, oldJSON, newJSON)
	return err
}

func (r *OrderHistoryRepository) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]models.OrderHistory, error) {
	var history []models.OrderHistory
	err := r.db.SelectContext(ctx, &history, `
		SELECT * FROM order_history WHERE order_id = $1 ORDER BY created_at ASC
	`, orderID)
	return history, err
}
