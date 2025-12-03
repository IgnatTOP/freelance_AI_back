package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/service"
	"github.com/ignatzorin/freelance-backend/internal/validation"
)

// AuthHandler предоставляет HTTP слой для регистрации и логина.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler создаёт хэндлер.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register обрабатывает POST /auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required"`
		Username    string `json:"username"`
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация email
	if err := validation.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация пароля
	if err := validation.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация username
	if req.Username != "" {
		if err := validation.ValidateUsername(req.Username); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Валидация display_name
	if req.DisplayName != "" {
		if err := validation.ValidateDisplayName(req.DisplayName); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Валидация роли
	if req.Role != "" && req.Role != "client" && req.Role != "freelancer" && req.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "роль должна быть client, freelancer или admin"})
		return
	}

	meta := map[string]string{
		"user_agent": c.GetHeader("User-Agent"),
		"ip":         c.ClientIP(),
	}

	result, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		Username:    req.Username,
		Role:        req.Role,
		DisplayName: req.DisplayName,
	}, meta)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":    result.User,
		"profile": result.Profile,
		"tokens":  result.TokenPair,
	})
}

// Login обрабатывает POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация email
	if err := validation.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация пароля (не пустой)
	if strings.TrimSpace(req.Password) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "пароль обязателен"})
		return
	}

	meta := map[string]string{
		"user_agent": c.GetHeader("User-Agent"),
		"ip":         c.ClientIP(),
	}

	result, err := h.auth.Login(c.Request.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}, meta)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    result.User,
		"profile": result.Profile,
		"tokens":  result.TokenPair,
	})
}

// Refresh обрабатывает POST /auth/refresh.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	meta := map[string]string{
		"user_agent": c.GetHeader("User-Agent"),
		"ip":         c.ClientIP(),
	}

	tokenPair, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken, meta)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokenPair})
}

// ListSessions обрабатывает GET /auth/sessions - список активных сессий.
func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	sessions, err := h.auth.ListSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// DeleteSession обрабатывает DELETE /auth/sessions/:id - удаление конкретной сессии.
func (h *AuthHandler) DeleteSession(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор сессии"})
		return
	}

	if err := h.auth.DeleteSession(c.Request.Context(), sessionID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "сессия успешно удалена"})
}

// DeleteAllSessionsExcept обрабатывает DELETE /auth/sessions - удаление всех сессий кроме текущей.
func (h *AuthHandler) DeleteAllSessionsExcept(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Получаем refresh токен из заголовка или body
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Пытаемся получить из заголовка
		req.RefreshToken = c.GetHeader("X-Refresh-Token")
		if req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token обязателен"})
			return
		}
	}

	if err := h.auth.DeleteAllSessionsExcept(c.Request.Context(), userID, req.RefreshToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "все сессии кроме текущей успешно удалены"})
}
