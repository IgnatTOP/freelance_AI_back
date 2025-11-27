package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/logger"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/validation"
)

// AuthRepository описывает зависимости AuthService от слоя хранилища.
type AuthRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error)
	UpsertProfile(ctx context.Context, profile *models.Profile) error
	CreateSession(ctx context.Context, session *models.Session) error
	DeleteSession(ctx context.Context, refreshToken string) error
	UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error
	ListSessions(ctx context.Context, userID uuid.UUID) ([]models.Session, error)
	DeleteSessionByID(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) error
	DeleteAllSessionsExcept(ctx context.Context, userID uuid.UUID, exceptRefreshToken string) error
}

// AuthService инкапсулирует бизнес-логику регистрации и аутентификации.
type AuthService struct {
	repo         AuthRepository
	tokenManager *TokenManager
}

// RegisterInput содержит данные пользователя при регистрации.
type RegisterInput struct {
	Email       string
	Password    string
	Username    string
	Role        string
	DisplayName string
}

// LoginInput содержит данные для входа.
type LoginInput struct {
	Email    string
	Password string
}

// AuthResult возвращает итог регистрации или авторизации.
type AuthResult struct {
	User      *models.User
	Profile   *models.Profile
	TokenPair *TokenPair
}

// NewAuthService создаёт сервис аутентификации.
func NewAuthService(repo AuthRepository, tokenManager *TokenManager) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenManager: tokenManager,
	}
}

// Register создаёт нового пользователя и профиль.
func (s *AuthService) Register(ctx context.Context, in RegisterInput, meta map[string]string) (*AuthResult, error) {
	// Валидация email на уровне сервиса
	if err := validation.ValidateEmail(in.Email); err != nil {
		return nil, fmt.Errorf("auth service: %w", err)
	}

	if _, err := s.repo.GetByEmail(ctx, in.Email); err == nil {
		return nil, fmt.Errorf("auth service: email уже зарегистрирован")
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	username := in.Username
	if username == "" {
		username = deriveUsername(in.Email)
	}

	role := in.Role
	if role == "" {
		role = "freelancer"
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth service: не удалось захешировать пароль: %w", err)
	}

	user := &models.User{
		Email:        strings.ToLower(in.Email),
		Username:     username,
		PasswordHash: string(passHash),
		Role:         role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	displayName := in.DisplayName
	if displayName == "" {
		displayName = username
	}

	profile := &models.Profile{
		UserID:          user.ID,
		DisplayName:     displayName,
		ExperienceLevel: "junior",
		Skills:          []string{},
	}

	if err := s.repo.UpsertProfile(ctx, profile); err != nil {
		return nil, err
	}

	tokenPair, _, refreshExp, err := s.tokenManager.GeneratePair(user)
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		UserID:       user.ID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    refreshExp,
	}

	if meta != nil {
		if ua, ok := meta["user_agent"]; ok {
			session.UserAgent = &ua
		}
		if ip, ok := meta["ip"]; ok {
			session.IPAddress = &ip
		}
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return &AuthResult{
		User:      user,
		Profile:   profile,
		TokenPair: tokenPair,
	}, nil
}

// Login проверяет учётные данные и возвращает токены.
func (s *AuthService) Login(ctx context.Context, in LoginInput, meta map[string]string) (*AuthResult, error) {
	// Валидация email на уровне сервиса
	if err := validation.ValidateEmail(in.Email); err != nil {
		return nil, fmt.Errorf("auth service: %w", err)
	}

	user, err := s.repo.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("auth service: неверный email или пароль")
	}

	// Проверка активности пользователя
	if !user.IsActive {
		return nil, fmt.Errorf("auth service: аккаунт заблокирован")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, fmt.Errorf("auth service: неверный email или пароль")
	}

	// Обновляем время последнего входа
	if err := s.repo.UpdateLastLoginAt(ctx, user.ID); err != nil {
		// Логируем ошибку, но не прерываем процесс логина
		if logger.Log != nil {
			logger.Log.WithFields(map[string]interface{}{
				"user_id": user.ID,
				"error":   err.Error(),
			}).Warn("auth service: не удалось обновить last_login_at")
		}
	}

	tokenPair, _, refreshExp, err := s.tokenManager.GeneratePair(user)
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		UserID:       user.ID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    refreshExp,
	}

	if meta != nil {
		if ua, ok := meta["user_agent"]; ok {
			session.UserAgent = &ua
		}
		if ip, ok := meta["ip"]; ok {
			session.IPAddress = &ip
		}
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	// Загружаем профиль пользователя
	profile, err := s.repo.GetProfile(ctx, user.ID)
	if err != nil {
		// Если профиль не существует, создаём дефолтный
		profile = &models.Profile{
			UserID:          user.ID,
			DisplayName:     user.Username,
			ExperienceLevel: "junior",
			Skills:          []string{},
		}
		if err := s.repo.UpsertProfile(ctx, profile); err != nil {
			// Не критично, если профиль не создался
			profile = nil
		}
	}

	return &AuthResult{
		User:      user,
		Profile:   profile,
		TokenPair: tokenPair,
	}, nil
}

// Refresh выпускает новую пару токенов.
func (s *AuthService) Refresh(ctx context.Context, oldToken string, meta map[string]string) (*TokenPair, error) {
	claims, err := s.tokenManager.ParseRefresh(oldToken)
	if err != nil {
		return nil, fmt.Errorf("auth service: refresh токен невалиден: %w", err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("auth service: некорректный subject: %w", err)
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	tokenPair, _, refreshExp, err := s.tokenManager.GeneratePair(user)
	if err != nil {
		return nil, err
	}

	if err := s.repo.DeleteSession(ctx, oldToken); err != nil {
		return nil, err
	}

	session := &models.Session{
		UserID:       userID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    refreshExp,
	}

	if meta != nil {
		if ua, ok := meta["user_agent"]; ok {
			session.UserAgent = &ua
		}
		if ip, ok := meta["ip"]; ok {
			session.IPAddress = &ip
		}
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return tokenPair, nil
}

// ListSessions возвращает список активных сессий пользователя.
func (s *AuthService) ListSessions(ctx context.Context, userID uuid.UUID) ([]models.Session, error) {
	return s.repo.ListSessions(ctx, userID)
}

// DeleteSession удаляет сессию по идентификатору.
func (s *AuthService) DeleteSession(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) error {
	return s.repo.DeleteSessionByID(ctx, sessionID, userID)
}

// DeleteAllSessionsExcept удаляет все сессии пользователя кроме текущей.
func (s *AuthService) DeleteAllSessionsExcept(ctx context.Context, userID uuid.UUID, currentRefreshToken string) error {
	return s.repo.DeleteAllSessionsExcept(ctx, userID, currentRefreshToken)
}

// deriveUsername формирует красивый username из email.
func deriveUsername(email string) string {
	name := strings.Split(email, "@")[0]
	name = strings.NewReplacer(".", "_", "+", "_").Replace(name)
	name = strings.ToLower(name)
	if len(name) < 3 {
		name = "user_" + uuid.NewString()[:6]
	}
	return name
}
