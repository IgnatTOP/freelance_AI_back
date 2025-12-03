package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/dto"
	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/service"
	"github.com/ignatzorin/freelance-backend/internal/validation"
)

// PortfolioHandler обслуживает маршруты портфолио.
type PortfolioHandler struct {
	portfolio *service.PortfolioService
}

// NewPortfolioHandler создаёт новый хэндлер.
func NewPortfolioHandler(portfolio *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{portfolio: portfolio}
}

// ListPortfolioItems обрабатывает GET /portfolio.
func (h *PortfolioHandler) ListPortfolioItems(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	items, err := h.portfolio.ListPortfolioItems(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем медиа для каждой работы
	type portfolioItemWithMedia struct {
		*models.PortfolioItem
		Media []models.MediaFile `json:"media"`
	}

	result := make([]portfolioItemWithMedia, 0, len(items))
	for _, item := range items {
		media, err := h.portfolio.ListPortfolioMedia(c.Request.Context(), item.ID)
		if err != nil {
			media = []models.MediaFile{}
		}
		result = append(result, portfolioItemWithMedia{
			PortfolioItem: &item,
			Media:         media,
		})
	}

	c.JSON(http.StatusOK, result)
}

// CreatePortfolioItem обрабатывает POST /portfolio.
func (h *PortfolioHandler) CreatePortfolioItem(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Title        string   `json:"title" binding:"required"`
		Description  *string  `json:"description"`
		CoverMediaID *string  `json:"cover_media_id"`
		AITags       []string `json:"ai_tags"`
		ExternalLink *string  `json:"external_link"`
		MediaIDs     []string `json:"media_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация заголовка
	if err := validation.ValidatePortfolioTitle(req.Title); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация описания
	if err := validation.ValidatePortfolioDescription(req.Description); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация внешней ссылки
	if err := validation.ValidateExternalLink(req.ExternalLink); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация тегов (как навыков)
	if req.AITags != nil && len(req.AITags) > 0 {
		if err := validation.ValidateSkills(req.AITags); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	var coverMediaID *uuid.UUID
	if req.CoverMediaID != nil && *req.CoverMediaID != "" {
		id, err := uuid.Parse(*req.CoverMediaID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный cover_media_id"})
			return
		}
		coverMediaID = &id
	}

	var mediaIDs []uuid.UUID
	for _, raw := range req.MediaIDs {
		if raw == "" {
			continue
		}
		mediaID, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "media_ids содержит некорректный UUID"})
			return
		}
		mediaIDs = append(mediaIDs, mediaID)
	}

	item, err := h.portfolio.CreatePortfolioItem(
		c.Request.Context(),
		userID,
		req.Title,
		req.Description,
		coverMediaID,
		req.AITags,
		req.ExternalLink,
		mediaIDs,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем медиа для работы
	media, err := h.portfolio.ListPortfolioMedia(c.Request.Context(), item.ID)
	if err != nil {
		// Не критично, если медиа не загрузились
		media = []models.MediaFile{}
	}

	c.JSON(http.StatusCreated, dto.NewPortfolioItemResponse(item, media))
}

// GetPortfolioItem обрабатывает GET /portfolio/:id.
func (h *PortfolioHandler) GetPortfolioItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор работы"})
		return
	}

	item, err := h.portfolio.GetPortfolioItem(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPortfolioItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "работа не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем медиа для работы
	media, err := h.portfolio.ListPortfolioMedia(c.Request.Context(), id)
	if err != nil {
		// Не критично, если медиа не загрузились
		media = []models.MediaFile{}
	}

	c.JSON(http.StatusOK, dto.NewPortfolioItemResponse(item, media))
}

// UpdatePortfolioItem обрабатывает PUT /portfolio/:id.
func (h *PortfolioHandler) UpdatePortfolioItem(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор работы"})
		return
	}

	var req struct {
		Title        string   `json:"title" binding:"required"`
		Description  *string  `json:"description"`
		CoverMediaID *string  `json:"cover_media_id"`
		AITags       []string `json:"ai_tags"`
		ExternalLink *string  `json:"external_link"`
		MediaIDs     []string `json:"media_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация заголовка
	if err := validation.ValidatePortfolioTitle(req.Title); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация описания
	if err := validation.ValidatePortfolioDescription(req.Description); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация внешней ссылки
	if err := validation.ValidateExternalLink(req.ExternalLink); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация тегов (как навыков)
	if req.AITags != nil && len(req.AITags) > 0 {
		if err := validation.ValidateSkills(req.AITags); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	var coverMediaID *uuid.UUID
	if req.CoverMediaID != nil && *req.CoverMediaID != "" {
		mediaID, err := uuid.Parse(*req.CoverMediaID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный cover_media_id"})
			return
		}
		coverMediaID = &mediaID
	}

	var mediaIDs []uuid.UUID
	for _, raw := range req.MediaIDs {
		if raw == "" {
			continue
		}
		mediaID, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "media_ids содержит некорректный UUID"})
			return
		}
		mediaIDs = append(mediaIDs, mediaID)
	}

	item, err := h.portfolio.UpdatePortfolioItem(
		c.Request.Context(),
		id,
		userID,
		req.Title,
		req.Description,
		coverMediaID,
		req.AITags,
		req.ExternalLink,
		mediaIDs,
	)
	if err != nil {
		if errors.Is(err, repository.ErrPortfolioItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "работа не найдена"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем медиа для работы
	media, err := h.portfolio.ListPortfolioMedia(c.Request.Context(), id)
	if err != nil {
		media = []models.MediaFile{}
	}

	c.JSON(http.StatusOK, dto.NewPortfolioItemResponse(item, media))
}

// DeletePortfolioItem обрабатывает DELETE /portfolio/:id.
func (h *PortfolioHandler) DeletePortfolioItem(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор работы"})
		return
	}

	if err := h.portfolio.DeletePortfolioItem(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, repository.ErrPortfolioItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "работа не найдена"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "работа успешно удалена"})
}

// GetUserPortfolio обрабатывает GET /users/:id/portfolio - публичное портфолио пользователя.
func (h *PortfolioHandler) GetUserPortfolio(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор пользователя"})
		return
	}

	items, err := h.portfolio.ListPortfolioItems(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем медиа для каждой работы
	type portfolioItemWithMedia struct {
		*models.PortfolioItem
		Media []models.MediaFile `json:"media"`
	}

	result := make([]portfolioItemWithMedia, 0, len(items))
	for _, item := range items {
		media, err := h.portfolio.ListPortfolioMedia(c.Request.Context(), item.ID)
		if err != nil {
			media = []models.MediaFile{}
		}
		result = append(result, portfolioItemWithMedia{
			PortfolioItem: &item,
			Media:         media,
		})
	}

	c.JSON(http.StatusOK, result)
}
