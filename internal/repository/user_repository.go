package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// ErrUserNotFound возвращается, когда запись пользователя не найдена.
var ErrUserNotFound = errors.New("user not found")

// UserRepository отвечает за работу с таблицами users, profiles и user_sessions.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository создаёт экземпляр репозитория.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create создаёт нового пользователя с базовым профилем.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, username, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
		RETURNING id, created_at, updated_at
	`

	if err := r.db.QueryRowxContext(
		ctx, query,
		user.Email, user.Username, user.PasswordHash, user.Role,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return fmt.Errorf("user repository: create %w", err)
	}

	return nil
}

// GetByEmail возвращает пользователя по email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, username, password_hash, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	if err := r.db.GetContext(ctx, &user, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user repository: get by email %w", err)
	}

	return &user, nil
}

// GetByID возвращает пользователя по идентификатору.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, username, password_hash, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user repository: get by id %w", err)
	}

	return &user, nil
}

// UpsertProfile создаёт или обновляет профиль пользователя.
func (r *UserRepository) UpsertProfile(ctx context.Context, profile *models.Profile) error {
	query := `
		INSERT INTO profiles (user_id, display_name, bio, hourly_rate, experience_level, skills, location, photo_id, ai_summary, phone, telegram, website, company_name, inn, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET display_name = EXCLUDED.display_name,
			bio = EXCLUDED.bio,
			hourly_rate = EXCLUDED.hourly_rate,
			experience_level = EXCLUDED.experience_level,
			skills = EXCLUDED.skills,
			location = EXCLUDED.location,
			photo_id = EXCLUDED.photo_id,
			ai_summary = EXCLUDED.ai_summary,
			phone = EXCLUDED.phone,
			telegram = EXCLUDED.telegram,
			website = EXCLUDED.website,
			company_name = EXCLUDED.company_name,
			inn = EXCLUDED.inn,
			updated_at = NOW()
		RETURNING user_id, display_name, bio, hourly_rate, experience_level, skills, location, photo_id, ai_summary, phone, telegram, website, company_name, inn, updated_at
	`

	var skills pq.StringArray
	row := r.db.QueryRowxContext(
		ctx,
		query,
		profile.UserID,
		profile.DisplayName,
		profile.Bio,
		profile.HourlyRate,
		profile.ExperienceLevel,
		pq.Array(profile.Skills),
		profile.Location,
		profile.PhotoID,
		profile.AISummary,
		profile.Phone,
		profile.Telegram,
		profile.Website,
		profile.CompanyName,
		profile.INN,
	)

	if err := row.Scan(
		&profile.UserID,
		&profile.DisplayName,
		&profile.Bio,
		&profile.HourlyRate,
		&profile.ExperienceLevel,
		&skills,
		&profile.Location,
		&profile.PhotoID,
		&profile.AISummary,
		&profile.Phone,
		&profile.Telegram,
		&profile.Website,
		&profile.CompanyName,
		&profile.INN,
		&profile.UpdatedAt,
	); err != nil {
		return fmt.Errorf("user repository: upsert profile %w", err)
	}

	profile.Skills = []string(skills)

	return nil
}

// GetProfile возвращает профиль пользователя.
func (r *UserRepository) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	query := `SELECT user_id, display_name, bio, hourly_rate, experience_level, skills, location, photo_id, ai_summary, phone, telegram, website, company_name, inn, updated_at FROM profiles WHERE user_id = $1`
	
	var profile models.Profile
	var skills pq.StringArray
	
	if err := r.db.QueryRowxContext(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.DisplayName,
		&profile.Bio,
		&profile.HourlyRate,
		&profile.ExperienceLevel,
		&skills,
		&profile.Location,
		&profile.PhotoID,
		&profile.AISummary,
		&profile.Phone,
		&profile.Telegram,
		&profile.Website,
		&profile.CompanyName,
		&profile.INN,
		&profile.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user repository: get profile %w", err)
	}

	profile.Skills = []string(skills)

	return &profile, nil
}

// CreateSession сохраняет новую сессию пользователя.
func (r *UserRepository) CreateSession(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO user_sessions (user_id, refresh_token, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	if err := r.db.QueryRowxContext(
		ctx,
		query,
		session.UserID,
		session.RefreshToken,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt); err != nil {
		return fmt.Errorf("user repository: create session %w", err)
	}

	return nil
}

// DeleteSession удаляет сессию по refresh токену.
func (r *UserRepository) DeleteSession(ctx context.Context, refreshToken string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE refresh_token = $1`, refreshToken); err != nil {
		return fmt.Errorf("user repository: delete session %w", err)
	}

	return nil
}

// UpdateLastLoginAt обновляет время последнего входа пользователя.
func (r *UserRepository) UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("user repository: update last login at %w", err)
	}

	return nil
}

// ListSessions возвращает список всех активных сессий пользователя.
func (r *UserRepository) ListSessions(ctx context.Context, userID uuid.UUID) ([]models.Session, error) {
	query := `
		SELECT id, user_id, refresh_token, user_agent, ip_address, expires_at, created_at
		FROM user_sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	var sessions []models.Session
	if err := r.db.SelectContext(ctx, &sessions, query, userID); err != nil {
		return nil, fmt.Errorf("user repository: list sessions %w", err)
	}

	return sessions, nil
}

// DeleteSessionByID удаляет сессию по идентификатору.
func (r *UserRepository) DeleteSessionByID(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("user repository: delete session by id %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("user repository: delete session by id rows affected %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user repository: session not found")
	}

	return nil
}

// DeleteAllSessionsExcept удаляет все сессии пользователя кроме указанной.
func (r *UserRepository) DeleteAllSessionsExcept(ctx context.Context, userID uuid.UUID, exceptRefreshToken string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = $1 AND refresh_token != $2`, userID, exceptRefreshToken)
	if err != nil {
		return fmt.Errorf("user repository: delete all sessions except %w", err)
	}

	return nil
}

// GetReviewsForUser возвращает все отзывы о пользователе.
func (r *UserRepository) GetReviewsForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Review, error) {
	query := `
		SELECT id, order_id, reviewer_id, reviewed_id, rating, comment, created_at, updated_at
		FROM reviews
		WHERE reviewed_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var reviews []models.Review
	if err := r.db.SelectContext(ctx, &reviews, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("user repository: get reviews for user %w", err)
	}

	return reviews, nil
}

// GetUserStats возвращает статистику пользователя для публичного профиля.
func (r *UserRepository) GetUserStats(ctx context.Context, userID uuid.UUID) (*models.PublicProfileStats, error) {
	stats := &models.PublicProfileStats{}

	// Подсчитываем общее количество заказов (как заказчик)
	var clientOrders int
	clientOrdersQuery := `SELECT COUNT(*) FROM orders WHERE client_id = $1`
	if err := r.db.GetContext(ctx, &clientOrders, clientOrdersQuery, userID); err != nil {
		return nil, fmt.Errorf("user repository: get client orders %w", err)
	}

	// Подсчитываем общее количество заказов (как исполнитель)
	var freelancerOrders int
	freelancerOrdersQuery := `
		SELECT COUNT(DISTINCT o.id)
		FROM orders o
		INNER JOIN proposals p ON o.id = p.order_id
		WHERE p.freelancer_id = $1 AND p.status = 'accepted'
	`
	if err := r.db.GetContext(ctx, &freelancerOrders, freelancerOrdersQuery, userID); err != nil {
		return nil, fmt.Errorf("user repository: get freelancer orders %w", err)
	}
	stats.TotalOrders = clientOrders + freelancerOrders

	// Подсчитываем завершённые заказы (как заказчик)
	var clientCompleted int
	clientCompletedQuery := `SELECT COUNT(*) FROM orders WHERE client_id = $1 AND status = 'completed'`
	if err := r.db.GetContext(ctx, &clientCompleted, clientCompletedQuery, userID); err != nil {
		return nil, fmt.Errorf("user repository: get client completed %w", err)
	}

	// Подсчитываем завершённые заказы (как исполнитель)
	var freelancerCompleted int
	freelancerCompletedQuery := `
		SELECT COUNT(DISTINCT o.id)
		FROM orders o
		INNER JOIN proposals p ON o.id = p.order_id
		WHERE p.freelancer_id = $1 AND p.status = 'accepted' AND o.status = 'completed'
	`
	if err := r.db.GetContext(ctx, &freelancerCompleted, freelancerCompletedQuery, userID); err != nil {
		return nil, fmt.Errorf("user repository: get freelancer completed %w", err)
	}
	stats.CompletedOrders = clientCompleted + freelancerCompleted

	// Подсчитываем средний рейтинг и количество отзывов
	ratingQuery := `
		SELECT 
			COALESCE(AVG(rating), 0) as average_rating,
			COUNT(*) as total_reviews
		FROM reviews
		WHERE reviewed_id = $1
	`
	if err := r.db.QueryRowContext(ctx, ratingQuery, userID).Scan(&stats.AverageRating, &stats.TotalReviews); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			stats.AverageRating = 0
			stats.TotalReviews = 0
		} else {
			return nil, fmt.Errorf("user repository: get rating stats %w", err)
		}
	}

	// Округляем средний рейтинг до 2 знаков после запятой
	stats.AverageRating = float64(int(stats.AverageRating*100)) / 100

	return stats, nil
}

// ListFreelancers возвращает список всех активных фрилансеров с их профилями.
func (r *UserRepository) ListFreelancers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT u.id, u.email, u.username, u.password_hash, u.role, u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		WHERE u.role = 'freelancer' AND u.is_active = TRUE
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2
	`

	var users []*models.User
	if err := r.db.SelectContext(ctx, &users, query, limit, offset); err != nil {
		return nil, fmt.Errorf("user repository: list freelancers %w", err)
	}

	return users, nil
}

// CountFreelancers возвращает общее количество активных фрилансеров.
func (r *UserRepository) CountFreelancers(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE role = 'freelancer' AND is_active = TRUE`
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("user repository: count freelancers %w", err)
	}
	return count, nil
}

// GetCompletedOrdersForUser возвращает завершённые заказы пользователя (как заказчика и как исполнителя).
func (r *UserRepository) GetCompletedOrdersForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error) {
	query := `
		SELECT DISTINCT o.id, o.client_id, o.title, o.description, o.budget_min, o.budget_max, o.status, o.deadline_at, o.ai_summary, o.created_at, o.updated_at
		FROM orders o
		WHERE (o.client_id = $1 OR EXISTS (
			SELECT 1 FROM proposals p
			WHERE p.order_id = o.id AND p.freelancer_id = $1 AND p.status = 'accepted'
		))
		AND o.status = 'completed'
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`

	var orders []models.Order
	if err := r.db.SelectContext(ctx, &orders, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("user repository: get completed orders %w", err)
	}

	return orders, nil
}

// UpdateRole обновляет роль пользователя.
func (r *UserRepository) UpdateRole(ctx context.Context, userID uuid.UUID, role string) error {
	// Валидация роли
	validRoles := map[string]bool{
		"client":     true,
		"freelancer": true,
		"admin":      true,
	}
	if !validRoles[role] {
		return fmt.Errorf("user repository: invalid role %s", role)
	}

	query := `
		UPDATE users
		SET role = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, email, username, role, is_active, created_at, updated_at
	`

	var user models.User
	if err := r.db.QueryRowxContext(ctx, query, role, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("user repository: update role %w", err)
	}

	return nil
}


// FreelancerSearchParams параметры поиска фрилансеров.
type FreelancerSearchParams struct {
	Query           string
	Skills          []string
	MinHourlyRate   *float64
	MaxHourlyRate   *float64
	ExperienceLevel string
	Location        string
	MinRating       *float64
	Limit           int
	Offset          int
}

// SearchFreelancers ищет фрилансеров по параметрам.
func (r *UserRepository) SearchFreelancers(ctx context.Context, params FreelancerSearchParams) ([]*models.FreelancerSearchResult, error) {
	query := `
		SELECT 
			u.id, u.username, u.created_at,
			p.display_name, p.bio, p.hourly_rate, p.experience_level, p.skills, p.location, p.photo_id,
			COALESCE(AVG(rv.rating), 0) as avg_rating,
			COUNT(rv.id) as review_count
		FROM users u
		LEFT JOIN profiles p ON u.id = p.user_id
		LEFT JOIN reviews rv ON u.id = rv.reviewed_id
		WHERE u.role = 'freelancer' AND u.is_active = TRUE
	`
	args := []interface{}{}
	argNum := 1

	if params.Query != "" {
		query += fmt.Sprintf(` AND (p.display_name ILIKE $%d OR p.bio ILIKE $%d OR u.username ILIKE $%d)`, argNum, argNum, argNum)
		args = append(args, "%"+params.Query+"%")
		argNum++
	}
	if len(params.Skills) > 0 {
		query += fmt.Sprintf(` AND p.skills && $%d`, argNum)
		args = append(args, pq.Array(params.Skills))
		argNum++
	}
	if params.MinHourlyRate != nil {
		query += fmt.Sprintf(` AND p.hourly_rate >= $%d`, argNum)
		args = append(args, *params.MinHourlyRate)
		argNum++
	}
	if params.MaxHourlyRate != nil {
		query += fmt.Sprintf(` AND p.hourly_rate <= $%d`, argNum)
		args = append(args, *params.MaxHourlyRate)
		argNum++
	}
	if params.ExperienceLevel != "" {
		query += fmt.Sprintf(` AND p.experience_level = $%d`, argNum)
		args = append(args, params.ExperienceLevel)
		argNum++
	}
	if params.Location != "" {
		query += fmt.Sprintf(` AND p.location ILIKE $%d`, argNum)
		args = append(args, "%"+params.Location+"%")
		argNum++
	}

	query += ` GROUP BY u.id, u.username, u.created_at, p.display_name, p.bio, p.hourly_rate, p.experience_level, p.skills, p.location, p.photo_id`

	if params.MinRating != nil {
		query += fmt.Sprintf(` HAVING COALESCE(AVG(rv.rating), 0) >= $%d`, argNum)
		args = append(args, *params.MinRating)
		argNum++
	}

	query += ` ORDER BY avg_rating DESC, review_count DESC`
	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argNum, argNum+1)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("user repository: search freelancers %w", err)
	}
	defer rows.Close()

	var results []*models.FreelancerSearchResult
	for rows.Next() {
		var r models.FreelancerSearchResult
		var skills pq.StringArray
		if err := rows.Scan(
			&r.ID, &r.Username, &r.CreatedAt,
			&r.DisplayName, &r.Bio, &r.HourlyRate, &r.ExperienceLevel, &skills, &r.Location, &r.PhotoID,
			&r.AvgRating, &r.ReviewCount,
		); err != nil {
			return nil, err
		}
		r.Skills = []string(skills)
		results = append(results, &r)
	}
	return results, nil
}
