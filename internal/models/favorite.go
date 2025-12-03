package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	FavoriteTypeOrder      = "order"
	FavoriteTypeFreelancer = "freelancer"
)

type Favorite struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	TargetType string    `db:"target_type" json:"target_type"`
	TargetID   uuid.UUID `db:"target_id" json:"target_id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
