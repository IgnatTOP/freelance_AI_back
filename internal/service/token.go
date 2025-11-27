package service

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// TokenPair хранит пару access/refresh токенов.
type TokenPair struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    time.Duration `json:"expires_in"`
}

// TokenManager отвечает за выпуск и проверку JWT.
type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewTokenManager создаёт менеджер токенов.
func NewTokenManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

// GeneratePair выпускает новую пару токенов.
func (m *TokenManager) GeneratePair(user *models.User) (*TokenPair, time.Time, time.Time, error) {
	now := time.Now()
	accessExp := now.Add(m.accessTTL)
	refreshExp := now.Add(m.refreshTTL)

	accessToken, err := m.createToken(user, accessExp, m.accessSecret)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	refreshToken, err := m.createRefreshToken(user, refreshExp)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    m.accessTTL,
	}, accessExp, refreshExp, nil
}

// ParseRefresh проверяет refresh токен и возвращает клеймы.
func (m *TokenManager) ParseRefresh(token string) (*jwt.RegisteredClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		return m.refreshSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := parsed.Claims.(*jwt.RegisteredClaims); ok && parsed.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

// ParseAccess извлекает userID и роль из access токена.
func (m *TokenManager) ParseAccess(token string) (uuid.UUID, string, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return m.accessSecret, nil
	})
	if err != nil || !parsed.Valid {
		return uuid.Nil, "", err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", jwt.ErrTokenInvalidClaims
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, "", jwt.ErrTokenInvalidClaims
	}

	role, _ := claims["role"].(string)

	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, "", err
	}

	return userID, role, nil
}

// createToken формирует access токен.
func (m *TokenManager) createToken(user *models.User, exp time.Time, secret []byte) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": user.Role,
		"iat":  time.Now().Unix(),
		"exp":  exp.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// createRefresh формирует refresh токен со случайным ID.
func (m *TokenManager) createRefreshToken(user *models.User, exp time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(exp),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.refreshSecret)
}
