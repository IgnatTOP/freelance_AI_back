package models

import (
	"time"

	"github.com/google/uuid"
)

// User описывает сущность пользователя платформы.
type User struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Email        string     `db:"email" json:"email"`
	Username     string     `db:"username" json:"username"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Role         string     `db:"role" json:"role"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// Profile описывает публичный профиль пользователя.
type Profile struct {
	UserID          uuid.UUID  `db:"user_id" json:"user_id"`
	DisplayName     string     `db:"display_name" json:"display_name"`
	Bio             *string    `db:"bio" json:"bio,omitempty"`
	HourlyRate      *float64   `db:"hourly_rate" json:"hourly_rate,omitempty"`
	ExperienceLevel string     `db:"experience_level" json:"experience_level"`
	Skills          []string   `db:"skills" json:"skills"`
	Location        *string    `db:"location" json:"location,omitempty"`
	PhotoID         *uuid.UUID `db:"photo_id" json:"photo_id,omitempty"`
	AISummary       *string    `db:"ai_summary" json:"ai_summary,omitempty"`
	Phone           *string    `db:"phone" json:"phone,omitempty"`
	Telegram        *string    `db:"telegram" json:"telegram,omitempty"`
	Website         *string    `db:"website" json:"website,omitempty"`
	CompanyName     *string    `db:"company_name" json:"company_name,omitempty"`
	INN             *string    `db:"inn" json:"inn,omitempty"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

// Session представляет сохранённую сессию пользователя.
type Session struct {
	ID           uuid.UUID `db:"id" json:"id"`
	UserID       uuid.UUID `db:"user_id" json:"user_id"`
	RefreshToken string    `db:"refresh_token" json:"refresh_token"`
	UserAgent    *string   `db:"user_agent" json:"user_agent,omitempty"`
	IPAddress    *string   `db:"ip_address" json:"ip_address,omitempty"`
	ExpiresAt    time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// Review описывает отзыв пользователя о другом пользователе после завершения заказа.
type Review struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	OrderID    uuid.UUID  `db:"order_id" json:"order_id"`
	ReviewerID uuid.UUID  `db:"reviewer_id" json:"reviewer_id"`
	ReviewedID uuid.UUID  `db:"reviewed_id" json:"reviewed_id"`
	Rating     int        `db:"rating" json:"rating"`
	Comment    *string    `db:"comment" json:"comment,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
}

// PublicProfileStats содержит статистику для публичного профиля.
type PublicProfileStats struct {
	TotalOrders      int     `json:"total_orders"`
	CompletedOrders  int     `json:"completed_orders"`
	AverageRating    float64 `json:"average_rating"`
	TotalReviews     int     `json:"total_reviews"`
	TotalEarnings    float64 `json:"total_earnings,omitempty"`
}


// FreelancerSearchResult результат поиска фрилансера.
type FreelancerSearchResult struct {
	ID              uuid.UUID  `json:"id"`
	Username        string     `json:"username"`
	DisplayName     *string    `json:"display_name,omitempty"`
	Bio             *string    `json:"bio,omitempty"`
	HourlyRate      *float64   `json:"hourly_rate,omitempty"`
	ExperienceLevel *string    `json:"experience_level,omitempty"`
	Skills          []string   `json:"skills,omitempty"`
	Location        *string    `json:"location,omitempty"`
	PhotoID         *uuid.UUID `json:"photo_id,omitempty"`
	AvgRating       float64    `json:"avg_rating"`
	ReviewCount     int        `json:"review_count"`
	CreatedAt       time.Time  `json:"created_at"`
}
