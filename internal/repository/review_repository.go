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

var ErrReviewNotFound = errors.New("review not found")

type ReviewRepository struct {
	db *sqlx.DB
}

func NewReviewRepository(db *sqlx.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// Create создаёт отзыв.
func (r *ReviewRepository) Create(ctx context.Context, review *models.Review) error {
	query := `
		INSERT INTO reviews (order_id, reviewer_id, reviewed_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowxContext(ctx, query,
		review.OrderID, review.ReviewerID, review.ReviewedID, review.Rating, review.Comment,
	).Scan(&review.ID, &review.CreatedAt, &review.UpdatedAt)
}

// GetByID возвращает отзыв по ID.
func (r *ReviewRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Review, error) {
	var review models.Review
	err := r.db.GetContext(ctx, &review, `SELECT * FROM reviews WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return &review, nil
}

// GetByOrderAndReviewer проверяет, оставлял ли пользователь отзыв на заказ.
func (r *ReviewRepository) GetByOrderAndReviewer(ctx context.Context, orderID, reviewerID uuid.UUID) (*models.Review, error) {
	var review models.Review
	err := r.db.GetContext(ctx, &review, `SELECT * FROM reviews WHERE order_id = $1 AND reviewer_id = $2`, orderID, reviewerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &review, nil
}

// ListByReviewedID возвращает отзывы о пользователе.
func (r *ReviewRepository) ListByReviewedID(ctx context.Context, reviewedID uuid.UUID, limit, offset int) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.SelectContext(ctx, &reviews, `
		SELECT * FROM reviews WHERE reviewed_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, reviewedID, limit, offset)
	return reviews, err
}

// ListByOrderID возвращает отзывы по заказу.
func (r *ReviewRepository) ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.SelectContext(ctx, &reviews, `SELECT * FROM reviews WHERE order_id = $1`, orderID)
	return reviews, err
}

// GetAverageRating возвращает средний рейтинг пользователя.
func (r *ReviewRepository) GetAverageRating(ctx context.Context, userID uuid.UUID) (float64, int, error) {
	var result struct {
		Avg   sql.NullFloat64 `db:"avg"`
		Count int             `db:"count"`
	}
	err := r.db.GetContext(ctx, &result, `
		SELECT COALESCE(AVG(rating), 0) as avg, COUNT(*) as count FROM reviews WHERE reviewed_id = $1
	`, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("review repository: get average rating %w", err)
	}
	return result.Avg.Float64, result.Count, nil
}

// Update обновляет отзыв.
func (r *ReviewRepository) Update(ctx context.Context, review *models.Review) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE reviews SET rating = $2, comment = $3, updated_at = NOW() WHERE id = $1
	`, review.ID, review.Rating, review.Comment)
	return err
}

// Delete удаляет отзыв.
func (r *ReviewRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM reviews WHERE id = $1`, id)
	return err
}
