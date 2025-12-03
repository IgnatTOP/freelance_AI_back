package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OrderHistory struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	OrderID   uuid.UUID       `db:"order_id" json:"order_id"`
	UserID    *uuid.UUID      `db:"user_id" json:"user_id,omitempty"`
	Action    string          `db:"action" json:"action"`
	OldValue  json.RawMessage `db:"old_value" json:"old_value,omitempty"`
	NewValue  json.RawMessage `db:"new_value" json:"new_value,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}
