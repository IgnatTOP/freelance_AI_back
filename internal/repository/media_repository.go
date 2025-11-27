package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// MediaRepository работает с таблицей media_files.
type MediaRepository struct {
	db *sqlx.DB
}

// NewMediaRepository создаёт экземпляр.
func NewMediaRepository(db *sqlx.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

// ErrMediaNotFound сигнализирует об отсутствии файла.
var ErrMediaNotFound = errors.New("media not found")

// Create сохраняет запись о файле.
func (r *MediaRepository) Create(ctx context.Context, media *models.MediaFile) error {
	query := `
		INSERT INTO media_files (user_id, file_path, file_type, file_size, is_public)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	if err := r.db.QueryRowxContext(
		ctx,
		query,
		media.UserID,
		media.FilePath,
		media.FileType,
		media.FileSize,
		media.IsPublic,
	).Scan(&media.ID, &media.CreatedAt); err != nil {
		return fmt.Errorf("media repository: create %w", err)
	}

	return nil
}

// GetByID возвращает запись о файле.
func (r *MediaRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.MediaFile, error) {
	var media models.MediaFile
	if err := r.db.GetContext(ctx, &media, `SELECT * FROM media_files WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMediaNotFound
		}
		return nil, fmt.Errorf("media repository: get by id %w", err)
	}
	return &media, nil
}

// Delete удаляет запись о файле.
func (r *MediaRepository) Delete(ctx context.Context, mediaID uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM media_files WHERE id = $1`, mediaID); err != nil {
		return fmt.Errorf("media repository: delete %w", err)
	}
	return nil
}
