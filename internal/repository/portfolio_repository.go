package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// ErrPortfolioItemNotFound возвращается, когда работа портфолио не найдена.
var ErrPortfolioItemNotFound = errors.New("portfolio item not found")

// PortfolioRepository отвечает за работу с портфолио.
type PortfolioRepository struct {
	db *sqlx.DB
}

// NewPortfolioRepository создаёт экземпляр репозитория.
func NewPortfolioRepository(db *sqlx.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

// Create создаёт новую работу в портфолио.
func (r *PortfolioRepository) Create(ctx context.Context, item *models.PortfolioItem, mediaIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("portfolio repository: begin tx %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		INSERT INTO portfolio_items (user_id, title, description, cover_media_id, ai_tags, external_link)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	if err = tx.QueryRowxContext(
		ctx,
		query,
		item.UserID,
		item.Title,
		item.Description,
		item.CoverMediaID,
		pq.Array(item.AITags),
		item.ExternalLink,
	).Scan(&item.ID, &item.CreatedAt); err != nil {
		return fmt.Errorf("portfolio repository: insert item %w", err)
	}

	if len(mediaIDs) > 0 {
		// Batch INSERT для media (устранение N+1)
		mediaQuery := `INSERT INTO portfolio_media (portfolio_id, media_id, position) VALUES `
		mediaValues := make([]interface{}, 0, len(mediaIDs)*3)

		for i, mediaID := range mediaIDs {
			if i > 0 {
				mediaQuery += ", "
			}
			mediaQuery += fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
			mediaValues = append(mediaValues, item.ID, mediaID, i)
		}
		mediaQuery += " ON CONFLICT DO NOTHING"

		if _, err = tx.ExecContext(ctx, mediaQuery, mediaValues...); err != nil {
			return fmt.Errorf("portfolio repository: batch insert media %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("portfolio repository: commit %w", err)
	}

	return nil
}

// GetByID возвращает работу по идентификатору.
func (r *PortfolioRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PortfolioItem, error) {
	query := `
		SELECT id, user_id, title, description, cover_media_id, ai_tags, external_link, created_at
		FROM portfolio_items
		WHERE id = $1
	`
	
	var item models.PortfolioItem
	var aiTags pq.StringArray
	
	if err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&item.ID,
		&item.UserID,
		&item.Title,
		&item.Description,
		&item.CoverMediaID,
		&aiTags,
		&item.ExternalLink,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPortfolioItemNotFound
		}
		return nil, fmt.Errorf("portfolio repository: get by id %w", err)
	}
	
	item.AITags = []string(aiTags)
	return &item, nil
}

// List возвращает список работ пользователя.
func (r *PortfolioRepository) List(ctx context.Context, userID uuid.UUID) ([]models.PortfolioItem, error) {
	query := `
		SELECT id, user_id, title, description, cover_media_id, ai_tags, external_link, created_at
		FROM portfolio_items
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("portfolio repository: list query %w", err)
	}
	defer rows.Close()

	var items []models.PortfolioItem
	for rows.Next() {
		var item models.PortfolioItem
		var aiTags pq.StringArray
		
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Title,
			&item.Description,
			&item.CoverMediaID,
			&aiTags,
			&item.ExternalLink,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("portfolio repository: list scan %w", err)
		}
		
		item.AITags = []string(aiTags)
		items = append(items, item)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("portfolio repository: list rows %w", err)
	}

	return items, nil
}

// Update обновляет работу в портфолио.
func (r *PortfolioRepository) Update(ctx context.Context, item *models.PortfolioItem, mediaIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("portfolio repository: begin tx %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		UPDATE portfolio_items
		SET title = $1,
		    description = $2,
		    cover_media_id = $3,
		    ai_tags = $4,
		    external_link = $5
		WHERE id = $6 AND user_id = $7
		RETURNING created_at
	`

	var createdAt time.Time
	err = tx.QueryRowxContext(
		ctx,
		query,
		item.Title,
		item.Description,
		item.CoverMediaID,
		pq.Array(item.AITags),
		item.ExternalLink,
		item.ID,
		item.UserID,
	).Scan(&createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPortfolioItemNotFound
		}
		return fmt.Errorf("portfolio repository: update item %w", err)
	}
	item.CreatedAt = createdAt

	// Удаляем старые медиа
	if _, err = tx.ExecContext(ctx, `DELETE FROM portfolio_media WHERE portfolio_id = $1`, item.ID); err != nil {
		return fmt.Errorf("portfolio repository: clear media %w", err)
	}

	// Добавляем новые медиа (batch INSERT для устранения N+1)
	if len(mediaIDs) > 0 {
		mediaQuery := `INSERT INTO portfolio_media (portfolio_id, media_id, position) VALUES `
		mediaValues := make([]interface{}, 0, len(mediaIDs)*3)

		for i, mediaID := range mediaIDs {
			if i > 0 {
				mediaQuery += ", "
			}
			mediaQuery += fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
			mediaValues = append(mediaValues, item.ID, mediaID, i)
		}
		mediaQuery += " ON CONFLICT DO NOTHING"

		if _, err = tx.ExecContext(ctx, mediaQuery, mediaValues...); err != nil {
			return fmt.Errorf("portfolio repository: batch insert media %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("portfolio repository: commit %w", err)
	}

	return nil
}

// Delete удаляет работу из портфолио.
func (r *PortfolioRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM portfolio_items WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("portfolio repository: delete %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("portfolio repository: delete rows affected %w", err)
	}

	if rowsAffected == 0 {
		return ErrPortfolioItemNotFound
	}

	return nil
}

// ListMedia возвращает список медиа для работы.
func (r *PortfolioRepository) ListMedia(ctx context.Context, portfolioID uuid.UUID) ([]models.MediaFile, error) {
	query := `
		SELECT
			mf.id,
			mf.user_id,
			mf.file_path,
			mf.file_type,
			mf.file_size,
			mf.is_public,
			mf.created_at
		FROM portfolio_media pm
		JOIN media_files mf ON mf.id = pm.media_id
		WHERE pm.portfolio_id = $1
		ORDER BY pm.position
	`

	var media []models.MediaFile
	if err := r.db.SelectContext(ctx, &media, query, portfolioID); err != nil {
		return nil, fmt.Errorf("portfolio repository: list media %w", err)
	}

	return media, nil
}

