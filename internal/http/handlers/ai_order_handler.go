package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/service"
	"github.com/ignatzorin/freelance-backend/internal/ws"
)

// AIOrderHandler обслуживает маршруты AI операций с заказами
type AIOrderHandler struct {
	orders *service.OrderService
	users  *repository.UserRepository
	media  *repository.MediaRepository
	hub    *ws.Hub
}

// NewAIOrderHandler создаёт новый хэндлер.
func NewAIOrderHandler(orders *service.OrderService, users *repository.UserRepository, media *repository.MediaRepository, hub *ws.Hub) *AIOrderHandler {
	return &AIOrderHandler{orders: orders, users: users, media: media, hub: hub}
}

func (h *AIOrderHandler) GenerateOrderDescription(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Проверяем роль пользователя из базы данных (не из токена, так как токен может быть устаревшим)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут генерировать описания заказов"})
		return
	}

	var req struct {
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description"`
		Skills      []string `json:"skills"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.orders == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI сервис недоступен"})
		return
	}

	// Получаем AI клиент через рефлексию или добавляем метод в сервис
	// Для простоты, добавим метод в OrderService
	description, err := h.orders.GenerateOrderDescription(c.Request.Context(), req.Title, req.Description, req.Skills)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"description": description})
}

func (h *AIOrderHandler) StreamGenerateOrderDescription(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут генерировать описания заказов"})
		return
	}

	var req struct {
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description"`
		Skills      []string `json:"skills"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamGenerateOrderDescription(
		c.Request.Context(),
		req.Title,
		req.Description,
		req.Skills,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) GenerateProposal(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Проверяем роль пользователя из базы данных (не из токена, так как токен может быть устаревшим)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "freelancer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только исполнители могут генерировать предложения"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	// Опциональные параметры для переопределения данных профиля и портфолио
	var req struct {
		UserSkills     []string `json:"user_skills,omitempty"`
		UserExperience string   `json:"user_experience,omitempty"`
		UserBio        string   `json:"user_bio,omitempty"`
		Portfolio      []struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			AITags      []string `json:"ai_tags"`
		} `json:"portfolio,omitempty"`
	}

	// Если тело запроса пустое, это нормально - будем использовать данные из профиля
	_ = c.ShouldBindJSON(&req)

	// Преобразуем портфолио в нужный тип
	portfolioItems := make([]models.PortfolioItemForAI, len(req.Portfolio))
	for i, item := range req.Portfolio {
		portfolioItems[i] = models.PortfolioItemForAI{
			Title:       item.Title,
			Description: item.Description,
			AITags:      item.AITags,
		}
	}

	proposal, err := h.orders.GenerateProposal(c.Request.Context(), orderID, userID, req.UserSkills, req.UserExperience, req.UserBio, portfolioItems)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"proposal": proposal})
}

func (h *AIOrderHandler) StreamGenerateProposal(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Проверяем роль пользователя из базы данных (не из токена)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "freelancer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только исполнители могут генерировать предложения"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	// Опциональные параметры (как в GenerateProposal)
	var req struct {
		UserSkills     []string `json:"user_skills,omitempty"`
		UserExperience string   `json:"user_experience,omitempty"`
		UserBio        string   `json:"user_bio,omitempty"`
		Portfolio      []struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			AITags      []string `json:"ai_tags"`
		} `json:"portfolio,omitempty"`
	}

	_ = c.ShouldBindJSON(&req)

	portfolioItems := make([]models.PortfolioItemForAI, len(req.Portfolio))
	for i, item := range req.Portfolio {
		portfolioItems[i] = models.PortfolioItemForAI{
			Title:       item.Title,
			Description: item.Description,
			AITags:      item.AITags,
		}
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamGenerateProposal(
		c.Request.Context(),
		orderID,
		userID,
		req.UserSkills,
		req.UserExperience,
		req.UserBio,
		portfolioItems,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) ImproveOrderDescription(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Проверяем роль пользователя из базы данных (не из токена, так как токен может быть устаревшим)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут улучшать описания заказов"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	improved, err := h.orders.ImproveOrderDescription(c.Request.Context(), req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"description": improved})
}

func (h *AIOrderHandler) StreamImproveOrderDescription(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут улучшать описания заказов"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamImproveOrderDescription(
		c.Request.Context(),
		req.Title,
		req.Description,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) RegenerateOrderSummary(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	order, err := h.orders.RegenerateOrderSummary(c.Request.Context(), orderID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *AIOrderHandler) StreamRegenerateOrderSummary(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	order, err := h.orders.StreamRegenerateOrderSummary(
		c.Request.Context(),
		orderID,
		userID,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			_, _ = writeSSEEvent(c.Writer, "error", "заказ не найден")
			flusher.Flush()
			return
		}
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
		return
	}

	// В конце можно отправить событие с информацией об обновлённом заказе (опционально)
	orderJSON, _ := json.Marshal(order)
	_, _ = writeSSEEvent(c.Writer, "done", string(orderJSON))
	flusher.Flush()
}

// GetProposalFeedback обрабатывает GET /ai/orders/:id/proposals/feedback - получает рекомендации по улучшению отклика.
func (h *AIOrderHandler) GetProposalFeedback(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Проверяем роль пользователя из базы данных (не из токена, так как токен может быть устаревшим)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "freelancer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только исполнители могут получать рекомендации по улучшению откликов"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	feedback, err := h.orders.GetProposalFeedback(c.Request.Context(), orderID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrProposalNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "отклик не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}

// StreamProposalFeedback обрабатывает GET /ai/orders/:id/proposals/feedback/stream -
// стриминг рекомендаций по улучшению отклика через SSE.
func (h *AIOrderHandler) StreamProposalFeedback(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Проверяем роль пользователя из базы данных (не из токена, так как токен может быть устаревшим)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "freelancer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только исполнители могут получать рекомендации по улучшению откликов"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	// Настраиваем SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	// Стримим кусочки текста по мере генерации AI
	err = h.orders.StreamProposalFeedback(c.Request.Context(), orderID, userID, func(chunk string) error {
		// Отправляем как простое SSE-событие с data: <chunk>
		if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	})

	if err != nil {
		// В случае ошибки отправляем финальное событие с информацией об ошибке
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) RecommendRelevantOrders(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "freelancer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только фрилансеры могут получать рекомендации по заказам"})
		return
	}

	limit := common.ParseIntQuery(c, "limit", 10)
	// Ограничиваем максимум 10 заказов
	if limit > 10 {
		limit = 10
	}
	recommendedOrders, explanation, err := h.orders.RecommendRelevantOrders(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Преобразуем в формат для фронтенда
	orderIDs := make([]string, 0, len(recommendedOrders))
	ordersWithScores := make([]gin.H, 0, len(recommendedOrders))
	for _, rec := range recommendedOrders {
		orderIDs = append(orderIDs, rec.OrderID.String())
		ordersWithScores = append(ordersWithScores, gin.H{
			"order_id":    rec.OrderID.String(),
			"match_score": rec.MatchScore,
			"explanation": rec.Explanation,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"recommended_order_ids": orderIDs,
		"recommended_orders":    ordersWithScores,
		"explanation":           explanation,
	})
}

func (h *AIOrderHandler) StreamRecommendRelevantOrders(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "freelancer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только фрилансеры могут получать рекомендации по заказам"})
		return
	}

	limit := common.ParseIntQuery(c, "limit", 10)
	// Ограничиваем максимум 10 заказов - показываем только самые подходящие
	if limit > 10 {
		limit = 10
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamRecommendRelevantOrders(
		c.Request.Context(),
		userID,
		limit,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
		func(recommendedOrders []models.RecommendedOrder, generalExplanation string) error {
			// Преобразуем в формат для фронтенда
			orderIDs := make([]string, 0, len(recommendedOrders))
			ordersWithScores := make([]gin.H, 0, len(recommendedOrders))
			for _, rec := range recommendedOrders {
				orderIDs = append(orderIDs, rec.OrderID.String())
				ordersWithScores = append(ordersWithScores, gin.H{
					"order_id":    rec.OrderID.String(),
					"match_score": rec.MatchScore,
					"explanation": rec.Explanation,
				})
			}
			resultJSON, _ := json.Marshal(gin.H{
				"recommended_order_ids": orderIDs,
				"recommended_orders":    ordersWithScores,
				"explanation":           generalExplanation,
			})
			_, _ = writeSSEEvent(c.Writer, "data", string(resultJSON))
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) RecommendPriceAndTimeline(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "freelancer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только фрилансеры могут получать рекомендации по цене и срокам"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	recommendation, err := h.orders.RecommendPriceAndTimeline(c.Request.Context(), orderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, recommendation)
}

func (h *AIOrderHandler) StreamRecommendPriceAndTimeline(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "freelancer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только фрилансеры могут получать рекомендации по цене и срокам"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamRecommendPriceAndTimeline(
		c.Request.Context(),
		orderID,
		userID,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
		func(recommendation *models.PriceTimelineRecommendation) error {
			recJSON, _ := json.Marshal(recommendation)
			_, _ = writeSSEEvent(c.Writer, "data", string(recJSON))
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) EvaluateOrderQuality(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	evaluation, err := h.orders.EvaluateOrderQuality(c.Request.Context(), orderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, evaluation)
}

func (h *AIOrderHandler) StreamEvaluateOrderQuality(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamEvaluateOrderQuality(
		c.Request.Context(),
		orderID,
		userID,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
		func(evaluation *models.OrderQualityEvaluation) error {
			evalJSON, _ := json.Marshal(evaluation)
			_, _ = writeSSEEvent(c.Writer, "data", string(evalJSON))
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) FindSuitableFreelancers(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут искать подходящих исполнителей"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	limit := common.ParseIntQuery(c, "limit", 10)
	freelancers, err := h.orders.FindSuitableFreelancers(c.Request.Context(), orderID, userID, user.Role, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"freelancers": freelancers,
	})
}

func (h *AIOrderHandler) StreamFindSuitableFreelancers(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут искать подходящих исполнителей"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	limit := common.ParseIntQuery(c, "limit", 10)

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamFindSuitableFreelancers(
		c.Request.Context(),
		orderID,
		userID,
		user.Role,
		limit,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
		func(freelancers []models.SuitableFreelancer) error {
			freelancersJSON, _ := json.Marshal(gin.H{"freelancers": freelancers})
			_, _ = writeSSEEvent(c.Writer, "data", string(freelancersJSON))
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) AIChatAssistant(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}

	var req struct {
		Message     string                 `json:"message" binding:"required"`
		ContextData map[string]interface{} `json:"context_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ContextData == nil {
		req.ContextData = make(map[string]interface{})
	}

	response, err := h.orders.AIChatAssistant(c.Request.Context(), userID, req.Message, user.Role, req.ContextData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response": response,
	})
}

func (h *AIOrderHandler) StreamAIChatAssistant(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}

	var req struct {
		Message     string                 `json:"message" binding:"required"`
		ContextData map[string]interface{} `json:"context_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ContextData == nil {
		req.ContextData = make(map[string]interface{})
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamAIChatAssistant(c.Request.Context(), userID, req.Message, user.Role, req.ContextData, func(chunk string) error {
		if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	})

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) ImproveProfile(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		CurrentBio      string   `json:"current_bio" binding:"required"`
		Skills          []string `json:"skills"`
		ExperienceLevel string   `json:"experience_level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем профиль для получения навыков и уровня опыта, если не переданы
	profile, err := h.users.GetProfile(c.Request.Context(), userID)
	if err == nil && profile != nil {
		if len(req.Skills) == 0 {
			req.Skills = profile.Skills
		}
		if req.ExperienceLevel == "" {
			req.ExperienceLevel = profile.ExperienceLevel
		}
	}

	improved, err := h.orders.ImproveProfile(c.Request.Context(), req.CurrentBio, req.Skills, req.ExperienceLevel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"improved_bio": improved,
	})
}

func (h *AIOrderHandler) StreamImproveProfile(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		CurrentBio      string   `json:"current_bio" binding:"required"`
		Skills          []string `json:"skills"`
		ExperienceLevel string   `json:"experience_level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем профиль для получения навыков и уровня опыта, если не переданы
	profile, err := h.users.GetProfile(c.Request.Context(), userID)
	if err == nil && profile != nil {
		if len(req.Skills) == 0 {
			req.Skills = profile.Skills
		}
		if req.ExperienceLevel == "" {
			req.ExperienceLevel = profile.ExperienceLevel
		}
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamImproveProfile(
		c.Request.Context(),
		req.CurrentBio,
		req.Skills,
		req.ExperienceLevel,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) ImprovePortfolioItem(c *gin.Context) {
	_, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description" binding:"required"`
		AITags      []string `json:"ai_tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	improved, err := h.orders.ImprovePortfolioItem(c.Request.Context(), req.Title, req.Description, req.AITags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"improved_description": improved,
	})
}

func (h *AIOrderHandler) StreamImprovePortfolioItem(c *gin.Context) {
	_, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description"`
		AITags      []string `json:"ai_tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Отправляем ошибку валидации через SSE, если это возможно
		// Но сначала проверяем, можем ли мы установить SSE заголовки
		c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")

		flusher, ok := c.Writer.(http.Flusher)
		if ok {
			_, _ = writeSSEEvent(c.Writer, "error", err.Error())
			flusher.Flush()
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	// Если описание пустое, используем значение по умолчанию
	if req.Description == "" {
		req.Description = "Описание проекта"
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamImprovePortfolioItem(
		c.Request.Context(),
		req.Title,
		req.Description,
		req.AITags,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) GenerateOrderSuggestions(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут генерировать предложения для заказов"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.orders == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI сервис недоступен"})
		return
	}

	suggestions, err := h.orders.GenerateOrderSuggestions(c.Request.Context(), req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, suggestions)
}

func (h *AIOrderHandler) StreamGenerateOrderSuggestions(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут генерировать предложения для заказов"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamGenerateOrderSuggestions(
		c.Request.Context(),
		req.Title,
		req.Description,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) GenerateOrderSkills(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут генерировать навыки для заказов"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.orders == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI сервис недоступен"})
		return
	}

	skills, err := h.orders.GenerateOrderSkills(c.Request.Context(), req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"skills": skills})
}

func (h *AIOrderHandler) StreamGenerateOrderSkills(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут генерировать навыки для заказов"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamGenerateOrderSkills(
		c.Request.Context(),
		req.Title,
		req.Description,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) GenerateOrderBudget(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут генерировать бюджет для заказов"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.orders == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI сервис недоступен"})
		return
	}

	budget, err := h.orders.GenerateOrderBudget(c.Request.Context(), req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, budget)
}

func (h *AIOrderHandler) StreamGenerateOrderBudget(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут генерировать бюджет для заказов"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamGenerateOrderBudget(
		c.Request.Context(),
		req.Title,
		req.Description,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}

func (h *AIOrderHandler) GenerateWelcomeMessage(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}

	var req struct {
		UserRole string `json:"user_role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Если роль не передана, используем роль из профиля пользователя
		req.UserRole = user.Role
	}

	if req.UserRole == "" {
		req.UserRole = user.Role
	}

	if h.orders == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI сервис недоступен"})
		return
	}

	message, err := h.orders.GenerateWelcomeMessage(c.Request.Context(), req.UserRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

func (h *AIOrderHandler) StreamGenerateWelcomeMessage(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}

	var req struct {
		UserRole string `json:"user_role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Если роль не передана, используем роль из профиля пользователя
		req.UserRole = user.Role
	}

	if req.UserRole == "" {
		req.UserRole = user.Role
	}

	// SSE заголовки
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamGenerateWelcomeMessage(
		c.Request.Context(),
		req.UserRole,
		func(chunk string) error {
			if _, writeErr := writeSSEData(c.Writer, chunk); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		_, _ = writeSSEEvent(c.Writer, "error", err.Error())
		flusher.Flush()
	}
}
