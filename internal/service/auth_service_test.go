package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

// mockAuthRepository реализует AuthRepository для тестов.
type mockAuthRepository struct {
	usersByEmail map[string]*models.User
	usersByID    map[uuid.UUID]*models.User
	profiles     map[uuid.UUID]*models.Profile
	sessions     map[string]*models.Session
}

func newMockAuthRepository() *mockAuthRepository {
	return &mockAuthRepository{
		usersByEmail: make(map[string]*models.User),
		usersByID:    make(map[uuid.UUID]*models.User),
		profiles:     make(map[uuid.UUID]*models.Profile),
		sessions:     make(map[string]*models.Session),
	}
}

func (m *mockAuthRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.IsActive = true
	m.usersByEmail[user.Email] = user
	m.usersByID[user.ID] = user
	return nil
}

func (m *mockAuthRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if user, ok := m.usersByEmail[email]; ok {
		return user, nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockAuthRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if user, ok := m.usersByID[id]; ok {
		return user, nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockAuthRepository) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	if profile, ok := m.profiles[userID]; ok {
		return profile, nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockAuthRepository) UpsertProfile(ctx context.Context, profile *models.Profile) error {
	m.profiles[profile.UserID] = profile
	return nil
}

func (m *mockAuthRepository) CreateSession(ctx context.Context, session *models.Session) error {
	session.ID = uuid.New()
	session.CreatedAt = time.Now()
	m.sessions[session.RefreshToken] = session
	return nil
}

func (m *mockAuthRepository) DeleteSession(ctx context.Context, refreshToken string) error {
	delete(m.sessions, refreshToken)
	return nil
}

func (m *mockAuthRepository) ListSessions(ctx context.Context, userID uuid.UUID) ([]models.Session, error) {
	var sessions []models.Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			sessions = append(sessions, *s)
		}
	}
	return sessions, nil
}

func (m *mockAuthRepository) DeleteSessionByID(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) error {
	for token, s := range m.sessions {
		if s.ID == sessionID && s.UserID == userID {
			delete(m.sessions, token)
			return nil
		}
	}
	return nil
}

func (m *mockAuthRepository) DeleteAllSessionsExcept(ctx context.Context, userID uuid.UUID, exceptRefreshToken string) error {
	for token, s := range m.sessions {
		if s.UserID == userID && token != exceptRefreshToken {
			delete(m.sessions, token)
		}
	}
	return nil
}

func (m *mockAuthRepository) UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error {
	if user, ok := m.usersByID[userID]; ok {
		now := time.Now()
		user.LastLoginAt = &now
	}
	return nil
}

func TestAuthService_RegisterAndLogin(t *testing.T) {
	repo := newMockAuthRepository()
	tokenManager := NewTokenManager("access", "refresh", time.Minute, time.Hour)
	service := NewAuthService(repo, tokenManager)

	ctx := context.Background()
	res, err := service.Register(ctx, RegisterInput{
		Email:    "test@example.com",
		Password: "password123",
	}, map[string]string{"ip": "127.0.0.1"})
	if err != nil {
		t.Fatalf("register returned error: %v", err)
	}

	if res.User.ID == uuid.Nil {
		t.Fatalf("user ID должен быть установлен")
	}

	if res.Profile == nil || res.Profile.DisplayName == "" {
		t.Fatalf("профиль должен быть создан")
	}

	if len(repo.sessions) != 1 {
		t.Fatalf("ожидалась одна сессия, получили %d", len(repo.sessions))
	}

	loginRes, err := service.Login(ctx, LoginInput{
		Email:    "test@example.com",
		Password: "password123",
	}, nil)
	if err != nil {
		t.Fatalf("login returned error: %v", err)
	}

	if loginRes.TokenPair.AccessToken == "" {
		t.Fatalf("ожидался access токен")
	}
}

func TestAuthService_Refresh(t *testing.T) {
	repo := newMockAuthRepository()
	tokenManager := NewTokenManager("access-secret", "refresh-secret", time.Minute, time.Hour)
	service := NewAuthService(repo, tokenManager)

	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &models.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: string(hash),
		Role:         "freelancer",
	}
	repo.usersByEmail[user.Email] = user
	repo.usersByID[user.ID] = user

	tokenPair, accessExp, refreshExp, err := tokenManager.GeneratePair(user)
	if err != nil {
		t.Fatalf("не удалось сгенерировать токены: %v", err)
	}
	if accessExp.After(refreshExp) {
		t.Fatalf("access должен истекать раньше refresh")
	}

	repo.sessions[tokenPair.RefreshToken] = &models.Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    refreshExp,
	}

	newPair, err := service.Refresh(ctx, tokenPair.RefreshToken, nil)
	if err != nil {
		t.Fatalf("refresh вернул ошибку: %v", err)
	}

	if newPair.RefreshToken == tokenPair.RefreshToken {
		t.Fatalf("ожидался новый refresh токен")
	}
}
