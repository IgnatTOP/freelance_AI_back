package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

var ErrFavoriteNotFound = errors.New("favorite not found")

type FavoriteRepository struct {
	db *sqlx.DB
}

func NewFavoriteRepository(db *sqlx.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

func (r *FavoriteRepository) Add(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) (*models.Favorite, error) {
	var f models.Favorite
	err := r.db.GetContext(ctx, &f, `
		INSERT INTO favorites (user_id, target_type, target_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, target_type, target_id) DO UPDATE SET created_at = favorites.created_at
		RETURNING *
	`, userID, targetType, targetID)
	return &f, err
}

func (r *FavoriteRepository) Remove(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM favorites WHERE user_id = $1 AND target_type = $2 AND target_id = $3
	`, userID, targetType, targetID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrFavoriteNotFound
	}
	return nil
}

func (r *FavoriteRepository) ListByUser(ctx context.Context, userID uuid.UUID, targetType string, limit, offset int) ([]models.Favorite, error) {
	var favorites []models.Favorite
	query := `SELECT * FROM favorites WHERE user_id = $1`
	args := []interface{}{userID}
	if targetType != "" {
		query += ` AND target_type = $2`
		args = append(args, targetType)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + string(rune('0'+len(args)+1)) + ` OFFSET $` + string(rune('0'+len(args)+2))
	args = append(args, limit, offset)
	err := r.db.SelectContext(ctx, &favorites, query, args...)
	return favorites, err
}

func (r *FavoriteRepository) Exists(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND target_type = $2 AND target_id = $3)
	`, userID, targetType, targetID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return exists, err
}
