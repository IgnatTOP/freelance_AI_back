package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

var ErrVerificationCodeNotFound = errors.New("verification code not found")

type VerificationRepository struct {
	db *sqlx.DB
}

func NewVerificationRepository(db *sqlx.DB) *VerificationRepository {
	return &VerificationRepository{db: db}
}

func (r *VerificationRepository) CreateCode(ctx context.Context, userID uuid.UUID, codeType, code string, expiresAt time.Time) (*models.VerificationCode, error) {
	var vc models.VerificationCode
	err := r.db.GetContext(ctx, &vc, `
		INSERT INTO verification_codes (user_id, type, code, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING *
	`, userID, codeType, code, expiresAt)
	return &vc, err
}

func (r *VerificationRepository) VerifyCode(ctx context.Context, userID uuid.UUID, codeType, code string) (bool, error) {
	var vc models.VerificationCode
	err := r.db.GetContext(ctx, &vc, `
		SELECT * FROM verification_codes 
		WHERE user_id = $1 AND type = $2 AND code = $3 AND used = false AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, userID, codeType, code)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Помечаем код как использованный
	_, err = r.db.ExecContext(ctx, `UPDATE verification_codes SET used = true WHERE id = $1`, vc.ID)
	if err != nil {
		return false, err
	}

	// Обновляем статус верификации пользователя
	field := "email_verified"
	if codeType == models.VerificationTypePhone {
		field = "phone_verified"
	}
	_, err = r.db.ExecContext(ctx, `UPDATE users SET `+field+` = true WHERE id = $1`, userID)
	return err == nil, err
}

func (r *VerificationRepository) GetUserVerificationStatus(ctx context.Context, userID uuid.UUID) (emailVerified, phoneVerified, identityVerified bool, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT email_verified, phone_verified, identity_verified FROM users WHERE id = $1
	`, userID).Scan(&emailVerified, &phoneVerified, &identityVerified)
	return
}
