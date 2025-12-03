package models

import (
	"time"

	"github.com/google/uuid"
)

// Category представляет категорию заказов.
type Category struct {
	ID          uuid.UUID   `db:"id" json:"id"`
	Slug        string      `db:"slug" json:"slug"`
	Name        string      `db:"name" json:"name"`
	Description *string     `db:"description" json:"description,omitempty"`
	Icon        *string     `db:"icon" json:"icon,omitempty"`
	ParentID    *uuid.UUID  `db:"parent_id" json:"parent_id,omitempty"`
	SortOrder   int         `db:"sort_order" json:"sort_order"`
	IsActive    bool        `db:"is_active" json:"is_active"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
	Children    []Category  `json:"children,omitempty"`
}

// Skill представляет предустановленный навык.
type Skill struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	Slug       string     `db:"slug" json:"slug"`
	Name       string     `db:"name" json:"name"`
	CategoryID *uuid.UUID `db:"category_id" json:"category_id,omitempty"`
	IsActive   bool       `db:"is_active" json:"is_active"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}
