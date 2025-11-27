package models

import (
	"time"

	"github.com/google/uuid"
)

// MediaFile описывает загруженный файл.
type MediaFile struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	UserID    *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	FilePath  string     `db:"file_path" json:"file_path"`
	FileType  string     `db:"file_type" json:"file_type"`
	FileSize  int64      `db:"file_size" json:"file_size"`
	IsPublic  bool       `db:"is_public" json:"is_public"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// PortfolioItem описывает работу в портфолио.
type PortfolioItem struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	UserID       uuid.UUID  `db:"user_id" json:"user_id"`
	Title        string     `db:"title" json:"title"`
	Description  *string    `db:"description" json:"description,omitempty"`
	CoverMediaID *uuid.UUID `db:"cover_media_id" json:"cover_media_id,omitempty"`
	AITags       []string   `db:"ai_tags" json:"ai_tags"`
	ExternalLink *string    `db:"external_link" json:"external_link,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

// PortfolioItemForAI представляет элемент портфолио для AI.
type PortfolioItemForAI struct {
	Title       string
	Description string
	AITags      []string
}
