package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/validation"
	"github.com/ignatzorin/freelance-backend/internal/ws"
)

// ProfileHandler отвечает за работу с профилем.
type ProfileHandler struct {
	users *repository.UserRepository
	hub   *ws.Hub
}

// NewProfileHandler создаёт экземпляр.
func NewProfileHandler(users *repository.UserRepository, hub *ws.Hub) *ProfileHandler {
	return &ProfileHandler{users: users, hub: hub}
}

// GetMe возвращает профиль текущего пользователя.
func (h *ProfileHandler) GetMe(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.users.GetProfile(c.Request.Context(), userID)
	if err != nil {
		// Если профиль не найден, создаём дефолтный
		user, userErr := h.users.GetByID(c.Request.Context(), userID)
		if userErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "профиль не найден"})
			return
		}

		// Создаём дефолтный профиль
		profile = &models.Profile{
			UserID:          userID,
			DisplayName:     user.Username,
			ExperienceLevel: models.ExperienceLevelJunior,
			Skills:          []string{},
		}

		if err := h.users.UpsertProfile(c.Request.Context(), profile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось создать профиль"})
			return
		}
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateMe обновляет профиль текущего пользователя.
func (h *ProfileHandler) UpdateMe(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		DisplayName     string   `json:"display_name" binding:"required"`
		Bio             *string  `json:"bio"`
		HourlyRate      *float64 `json:"hourly_rate"`
		ExperienceLevel string   `json:"experience_level"`
		Skills          []string `json:"skills"`
		Location        *string  `json:"location"`
		PhotoID         *string  `json:"photo_id"`
		AISummary       *string  `json:"ai_summary"`
		Phone           *string  `json:"phone"`
		Telegram        *string  `json:"telegram"`
		Website         *string  `json:"website"`
		CompanyName     *string  `json:"company_name"`
		INN             *string  `json:"inn"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация отображаемого имени
	if err := validation.ValidateDisplayName(req.DisplayName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация биографии
	if err := validation.ValidateBio(req.Bio); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация местоположения
	if err := validation.ValidateLocation(req.Location); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация навыков
	if req.Skills != nil && len(req.Skills) > 0 {
		if err := validation.ValidateSkills(req.Skills); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Валидация почасовой ставки
	if err := validation.ValidateHourlyRate(req.HourlyRate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var photoUUID *uuid.UUID
	if req.PhotoID != nil && *req.PhotoID != "" {
		id, err := uuid.Parse(*req.PhotoID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "photo_id некорректен"})
			return
		}
		photoUUID = &id
	}

	profile := &models.Profile{
		UserID:          userID,
		DisplayName:     req.DisplayName,
		Bio:             req.Bio,
		HourlyRate:      req.HourlyRate,
		ExperienceLevel: req.ExperienceLevel,
		Skills:          req.Skills,
		Location:        req.Location,
		PhotoID:         photoUUID,
		AISummary:       req.AISummary,
		Phone:           req.Phone,
		Telegram:        req.Telegram,
		Website:         req.Website,
		CompanyName:     req.CompanyName,
		INN:             req.INN,
	}

	// Валидация уровня опыта
	if profile.ExperienceLevel == "" {
		profile.ExperienceLevel = models.ExperienceLevelJunior
	} else {
		if _, ok := models.ValidExperienceLevels[profile.ExperienceLevel]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный уровень опыта"})
			return
		}
	}

	// Валидация почасовой ставки
	if profile.HourlyRate != nil && *profile.HourlyRate < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "почасовая ставка не может быть отрицательной"})
		return
	}

	if profile.Skills == nil {
		profile.Skills = []string{}
	}

	if err := h.users.UpsertProfile(c.Request.Context(), profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// UpsertProfile теперь возвращает все поля через RETURNING *, поэтому profile уже содержит актуальные данные
	// WebSocket уведомление об обновлении профиля
	if h.hub != nil {
		_ = h.hub.BroadcastToUser(userID, "profile.updated", gin.H{
			"profile": profile,
			"message": "Профиль успешно обновлён",
		})
	}

	c.JSON(http.StatusOK, profile)
}

// GetUserProfile возвращает публичный профиль пользователя по ID.
func (h *ProfileHandler) GetUserProfile(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный ID пользователя"})
		return
	}

	// Получаем пользователя
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "пользователь не найден"})
		return
	}

	// Получаем профиль
	profile, err := h.users.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "профиль не найден"})
		return
	}

	// Получаем статистику
	stats, err := h.users.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		// Если статистика не найдена, создаём пустую
		stats = &models.PublicProfileStats{}
	}

	// Получаем отзывы (первые 10)
	reviews, err := h.users.GetReviewsForUser(c.Request.Context(), userID, 10, 0)
	if err != nil {
		reviews = []models.Review{}
	}

	// Получаем завершённые заказы (первые 10)
	completedOrders, err := h.users.GetCompletedOrdersForUser(c.Request.Context(), userID, 10, 0)
	if err != nil {
		completedOrders = []models.Order{}
	}

	// Возвращаем публичную информацию (без email и других приватных данных)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
		"profile":          profile,
		"stats":            stats,
		"reviews":          reviews,
		"completed_orders": completedOrders,
	})
}

// UpdateRole обрабатывает PUT /users/me/role - изменение роли пользователя.
func (h *ProfileHandler) UpdateRole(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=client freelancer"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Обновляем роль
	if err := h.users.UpdateRole(c.Request.Context(), userID, req.Role); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "пользователь не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем обновленного пользователя
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось получить обновленного пользователя"})
		return
	}

	c.JSON(http.StatusOK, user)
}
