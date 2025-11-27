package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/service"
	"github.com/ignatzorin/freelance-backend/internal/validation"
	"github.com/ignatzorin/freelance-backend/internal/ws"
)

// OrderHandler обслуживает маршруты заказов и откликов.
type OrderHandler struct {
	orders *service.OrderService
	users  *repository.UserRepository
	media  *repository.MediaRepository
	hub    *ws.Hub
}

// NewOrderHandler создаёт новый хэндлер.
func NewOrderHandler(orders *service.OrderService, users *repository.UserRepository, media *repository.MediaRepository, hub *ws.Hub) *OrderHandler {
	return &OrderHandler{orders: orders, users: users, media: media, hub: hub}
}

// CreateOrder обрабатывает POST /orders.
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, err := currentUserID(c)
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
	// Разрешаем создавать заказы клиентам и админам (админы считаются клиентами)
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут создавать заказы"})
		return
	}

	var req struct {
		Title        string   `json:"title" binding:"required"`
		Description  string   `json:"description" binding:"required"`
		BudgetMin    *float64 `json:"budget_min"`
		BudgetMax    *float64 `json:"budget_max"`
		DeadlineAt   *string  `json:"deadline_at"`
		Requirements []struct {
			Skill string `json:"skill" binding:"required"`
			Level string `json:"level"`
		} `json:"requirements"`
		Attachments []string `json:"attachment_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация заголовка
	if err := validation.ValidateOrderTitle(req.Title); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация описания
	if err := validation.ValidateOrderDescription(req.Description); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация бюджета
	if err := validation.ValidateBudget(req.BudgetMin, req.BudgetMax); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация требований
	if req.Requirements != nil {
		for _, r := range req.Requirements {
			if err := validation.ValidateRequirementSkill(r.Skill); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	var deadline *time.Time
	if req.DeadlineAt != nil && *req.DeadlineAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.DeadlineAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "deadline_at должен быть в формате RFC3339"})
			return
		}
		deadline = &parsed
	}

	var requirements []models.OrderRequirement
	if req.Requirements != nil {
		for _, r := range req.Requirements {
			level := r.Level
			if level == "" {
				level = "middle"
			}
			requirements = append(requirements, models.OrderRequirement{
				Skill: r.Skill,
				Level: level,
			})
		}
	}

	var attachmentIDs []uuid.UUID
	if req.Attachments != nil {
		for _, raw := range req.Attachments {
			if raw == "" {
				continue
			}
			mediaID, err := uuid.Parse(raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("attachment_ids содержит некорректный UUID: %s", raw)})
				return
			}
			attachmentIDs = append(attachmentIDs, mediaID)
		}
	}

	order, err := h.orders.CreateOrder(c.Request.Context(), service.CreateOrderInput{
		ClientID:      userID,
		Title:         req.Title,
		Description:   req.Description,
		BudgetMin:     req.BudgetMin,
		BudgetMax:     req.BudgetMax,
		DeadlineAt:    deadline,
		Requirements:  requirements,
		AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		// Определяем тип ошибки для правильного HTTP статуса
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		// Ошибки валидации возвращаем как BadRequest
		if contains(err.Error(), "не может быть") || contains(err.Error(), "некорректный") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем requirements и attachments для полного ответа
	requirements, errReq := h.orders.ListRequirements(c.Request.Context(), order.ID)
	if errReq != nil {
		requirements = []models.OrderRequirement{}
	}

	attachments, errAtt := h.orders.ListAttachments(c.Request.Context(), order.ID)
	if errAtt != nil {
		attachments = []models.OrderAttachment{}
	}

	// WebSocket уведомление создателю заказа
	if h.hub != nil {
		_ = h.hub.BroadcastToUser(userID, "orders.new", gin.H{
			"order":   order,
			"message": "Заказ успешно создан",
		})
	}

	type orderResponse struct {
		*models.Order
		Requirements []models.OrderRequirement `json:"requirements"`
		Attachments  []models.OrderAttachment  `json:"attachments"`
	}

	c.JSON(http.StatusCreated, orderResponse{
		Order:        order,
		Requirements: requirements,
		Attachments:  attachments,
	})
}

// ListOrders обрабатывает GET /orders.
func (h *OrderHandler) ListOrders(c *gin.Context) {
	// Маппинг статуса "open" на "published" для совместимости с API документацией
	status := c.Query("status")
	if status == "open" {
		status = "published"
	}

	params := repository.ListFilterParams{
		Status:    status,
		Search:    c.Query("search"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Limit:     parseIntQuery(c, "limit", 20),
		Offset:    parseIntQuery(c, "offset", 0),
	}

	// Парсим навыки из query параметра (может быть несколько)
	if skillsParam := c.Query("skills"); skillsParam != "" {
		params.Skills = strings.Split(skillsParam, ",")
		for i := range params.Skills {
			params.Skills[i] = strings.TrimSpace(params.Skills[i])
		}
	}

	// Парсим бюджет
	if budgetMinStr := c.Query("budget_min"); budgetMinStr != "" {
		if budgetMin, err := strconv.ParseFloat(budgetMinStr, 64); err == nil {
			params.BudgetMin = &budgetMin
		}
	}
	if budgetMaxStr := c.Query("budget_max"); budgetMaxStr != "" {
		if budgetMax, err := strconv.ParseFloat(budgetMaxStr, 64); err == nil {
			params.BudgetMax = &budgetMax
		}
	}

	result, err := h.orders.ListOrders(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result.Orders,
		"pagination": gin.H{
			"total":    result.Total,
			"limit":    result.Limit,
			"offset":   result.Offset,
			"has_more": result.HasMore,
		},
	})
}

// CreateProposal обрабатывает POST /orders/:id/proposals.
func (h *OrderHandler) CreateProposal(c *gin.Context) {
	userID, err := currentUserID(c)
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
		c.JSON(http.StatusForbidden, gin.H{"error": "только исполнители могут создавать предложения к заказам"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	var req struct {
		CoverLetter string   `json:"cover_letter" binding:"required"`
		Amount      *float64 `json:"amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация сопроводительного письма
	if err := validation.ValidateProposalCoverLetter(req.CoverLetter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация суммы предложения
	if req.Amount != nil {
		if *req.Amount < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "сумма предложения не может быть отрицательной"})
			return
		}
		if *req.Amount > validation.MaxBudget {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("сумма предложения не может превышать %.0f", validation.MaxBudget)})
			return
		}
	}

	proposal, err := h.orders.CreateProposal(c.Request.Context(), service.ProposalInput{
		OrderID:      orderID,
		FreelancerID: userID,
		CoverLetter:  req.CoverLetter,
		Amount:       req.Amount,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// WebSocket уведомления
	if h.hub != nil {
		order, err := h.orders.GetOrder(c.Request.Context(), orderID)
		if err == nil {
			// Уведомление клиенту о новом предложении
			_ = h.hub.BroadcastToUser(order.ClientID, "proposals.new", gin.H{
				"order": gin.H{
					"id":    order.ID,
					"title": order.Title,
				},
				"proposal": proposal,
				"message":  "Получено новое предложение",
			})
			// Уведомление фрилансеру о успешной отправке
			_ = h.hub.BroadcastToUser(userID, "proposals.sent", gin.H{
				"order": gin.H{
					"id":    order.ID,
					"title": order.Title,
				},
				"proposal": proposal,
				"message":  "Предложение успешно отправлено",
			})
		}
	}

	c.JSON(http.StatusCreated, proposal)
}

// GetOrder обрабатывает GET /orders/:id.
// Использует оптимизированный метод для избежания N+1 запросов.
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	order, requirements, attachments, err := h.orders.GetOrderWithDetails(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type orderResponse struct {
		*models.Order
		Requirements []models.OrderRequirement `json:"requirements"`
		Attachments  []models.OrderAttachment  `json:"attachments"`
	}

	c.JSON(http.StatusOK, orderResponse{
		Order:        order,
		Requirements: requirements,
		Attachments:  attachments,
	})
}

// UpdateOrder обрабатывает PUT /orders/:id.
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	existing, err := h.orders.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if existing.ClientID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "недостаточно прав для редактирования заказа"})
		return
	}

	var req struct {
		Title        string   `json:"title" binding:"required"`
		Description  string   `json:"description" binding:"required"`
		BudgetMin    *float64 `json:"budget_min"`
		BudgetMax    *float64 `json:"budget_max"`
		DeadlineAt   *string  `json:"deadline_at"`
		Status       string   `json:"status"`
		Requirements []struct {
			Skill string `json:"skill" binding:"required"`
			Level string `json:"level"`
		} `json:"requirements"`
		Attachments []string `json:"attachment_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация заголовка
	if err := validation.ValidateOrderTitle(req.Title); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация описания
	if err := validation.ValidateOrderDescription(req.Description); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация бюджета
	if err := validation.ValidateBudget(req.BudgetMin, req.BudgetMax); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация требований
	if req.Requirements != nil {
		for _, r := range req.Requirements {
			if err := validation.ValidateRequirementSkill(r.Skill); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	status := req.Status
	if status == "" {
		status = existing.Status
	}
	if _, ok := models.ValidOrderStatuses[status]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный статус заказа"})
		return
	}

	var deadline *time.Time
	if req.DeadlineAt != nil && *req.DeadlineAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, *req.DeadlineAt)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "deadline_at должен быть в формате RFC3339"})
			return
		}
		deadline = &parsed
	}

	requirements := make([]models.OrderRequirement, 0)
	if req.Requirements != nil {
		for _, item := range req.Requirements {
			level := item.Level
			if level == "" {
				level = "middle"
			}
			requirements = append(requirements, models.OrderRequirement{
				Skill: item.Skill,
				Level: level,
			})
		}
	}

	var attachmentIDs []uuid.UUID
	if req.Attachments != nil {
		for _, raw := range req.Attachments {
			if raw == "" {
				continue
			}
			mediaID, parseErr := uuid.Parse(raw)
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("attachment_ids содержит некорректный UUID: %s", raw)})
				return
			}
			attachmentIDs = append(attachmentIDs, mediaID)
		}
	}

	updated, err := h.orders.UpdateOrder(c.Request.Context(), service.UpdateOrderInput{
		OrderID:       orderID,
		ClientID:      userID,
		Title:         req.Title,
		Description:   req.Description,
		BudgetMin:     req.BudgetMin,
		BudgetMax:     req.BudgetMax,
		Status:        status,
		DeadlineAt:    deadline,
		Requirements:  requirements,
		AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	requirements, err = h.orders.ListRequirements(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	attachments, err := h.orders.ListAttachments(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var wsPayload = gin.H{
		"order":        updated,
		"requirements": requirements,
		"attachments":  attachments,
	}

	// WebSocket уведомления всем заинтересованным пользователям
	if h.hub != nil {
		// Уведомление владельцу заказа
		_ = h.hub.BroadcastToUser(userID, "orders.updated", wsPayload)

		// Уведомление всем фрилансерам, которые отправили предложения
		var clientID *uuid.UUID
		proposalsResult, err := h.orders.ListProposals(c.Request.Context(), orderID, clientID)
		if err == nil && proposalsResult != nil {
			proposals := proposalsResult.Proposals
			seen := make(map[uuid.UUID]struct{})
			for _, proposal := range proposals {
				if proposal.FreelancerID == userID {
					continue
				}
				if _, exists := seen[proposal.FreelancerID]; exists {
					continue
				}
				seen[proposal.FreelancerID] = struct{}{}
				_ = h.hub.BroadcastToUser(proposal.FreelancerID, "orders.updated", wsPayload)
			}
		}
	}

	type orderResponse struct {
		*models.Order
		Requirements []models.OrderRequirement `json:"requirements"`
		Attachments  []models.OrderAttachment  `json:"attachments"`
	}

	updated.Attachments = attachments

	c.JSON(http.StatusOK, orderResponse{
		Order:        updated,
		Requirements: requirements,
		Attachments:  attachments,
	})
}

// ListProposals обрабатывает GET /orders/:id/proposals.
func (h *OrderHandler) ListProposals(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	// Проверяем права доступа: должны видеть только владелец заказа и авторы предложений
	order, err := h.orders.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, является ли пользователь владельцем заказа
	isOwner := order.ClientID == userID

	// Если не владелец, проверяем, есть ли у пользователя предложение к этому заказу
	if !isOwner {
		var clientID *uuid.UUID
		proposalsResult, err := h.orders.ListProposals(c.Request.Context(), orderID, clientID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		hasProposal := false
		for _, proposal := range proposalsResult.Proposals {
			if proposal.FreelancerID == userID {
				hasProposal = true
				break
			}
		}

		if !hasProposal {
			c.JSON(http.StatusForbidden, gin.H{"error": "у вас нет доступа к предложениям этого заказа"})
			return
		}
	}

	// Передаём clientID только если пользователь - владелец заказа
	var clientID *uuid.UUID
	if isOwner {
		clientID = &userID
	}
	result, err := h.orders.ListProposals(c.Request.Context(), orderID, clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Если это не заказчик, возвращаем только список откликов (для обратной совместимости)
	if !isOwner {
		c.JSON(http.StatusOK, result.Proposals)
		return
	}

	// Для заказчика возвращаем полную структуру с рекомендацией
	c.JSON(http.StatusOK, result)
}

// GetMyProposal обрабатывает GET /orders/:id/my-proposal.
func (h *OrderHandler) GetMyProposal(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	proposal, err := h.orders.GetMyProposalForOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrProposalNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "предложение не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, proposal)
}

// UpdateProposalStatus обрабатывает PUT /orders/:orderId/proposals/:proposalId/status.
func (h *OrderHandler) UpdateProposalStatus(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	proposalID, err := uuid.Parse(c.Param("proposalId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор отклика"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, conversation, err := h.orders.UpdateProposalStatus(c.Request.Context(), userID, proposalID, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProposalNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "отклик не найден"})
		case errors.Is(err, repository.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	if updated.OrderID != orderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "отклик не относится к указанному заказу"})
		return
	}

	response := gin.H{
		"proposal":     updated,
		"conversation": conversation,
	}

	if orderData, err := h.orders.GetOrder(c.Request.Context(), orderID); err == nil {
		response["order"] = gin.H{
			"id":    orderData.ID,
			"title": orderData.Title,
		}
	}

	// WebSocket уведомления
	if h.hub != nil {
		payload := gin.H{
			"proposal":     updated,
			"conversation": conversation,
		}

		if ord, ok := response["order"]; ok {
			payload["order"] = ord
		}

		// Уведомление клиенту
		var clientMessage string
		switch updated.Status {
		case models.ProposalStatusAccepted:
			clientMessage = "Предложение принято"
		case models.ProposalStatusRejected:
			clientMessage = "Предложение отклонено"
		case models.ProposalStatusShortlisted:
			clientMessage = "Предложение добавлено в шортлист"
		default:
			clientMessage = "Статус предложения изменён"
		}
		payload["message"] = clientMessage
		_ = h.hub.BroadcastToUser(userID, "proposals.updated", payload)

		// Уведомление фрилансеру
		var freelancerMessage string
		switch updated.Status {
		case models.ProposalStatusAccepted:
			freelancerMessage = "Ваше предложение принято! Начните работу над заказом."
		case models.ProposalStatusRejected:
			freelancerMessage = "Ваше предложение отклонено"
		case models.ProposalStatusShortlisted:
			freelancerMessage = "Ваше предложение добавлено в шортлист"
		default:
			freelancerMessage = "Статус вашего предложения изменён"
		}
		freelancerPayload := gin.H{}
		for k, v := range payload {
			freelancerPayload[k] = v
		}
		freelancerPayload["message"] = freelancerMessage
		_ = h.hub.BroadcastToUser(updated.FreelancerID, "proposals.updated", freelancerPayload)
	}

	c.JSON(http.StatusOK, response)
}

// GetOrderChat обрабатывает GET /orders/:id/chat - возвращает чат для заказа (только если есть accepted proposal).
func (h *OrderHandler) GetOrderChat(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	conversation, acceptedProposal, err := h.orders.GetOrderChat(c.Request.Context(), orderID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		if errors.Is(err, repository.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "чат не найден"})
			return
		}
		// Проверяем, нет ли принятого исполнителя
		if err.Error() == "order service: для этого заказа нет принятого исполнителя" {
			c.JSON(http.StatusNotFound, gin.H{"error": "для этого заказа нет принятого исполнителя"})
			return
		}
		if err.Error() == "order service: у вас нет доступа к этому чату" {
			c.JSON(http.StatusForbidden, gin.H{"error": "у вас нет доступа к этому чату"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем информацию о заказе
	order, err := h.orders.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Определяем ID собеседника и получаем его информацию
	var otherUserID uuid.UUID
	if order.ClientID == userID {
		otherUserID = acceptedProposal.FreelancerID
	} else {
		otherUserID = order.ClientID
	}

	// Получаем информацию о собеседнике (исполнителе)
	freelancerProfile, err := h.users.GetProfile(c.Request.Context(), acceptedProposal.FreelancerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось получить профиль исполнителя"})
		return
	}

	var freelancerPhotoIDStr *string
	var freelancerPhotoURL *string
	if freelancerProfile.PhotoID != nil {
		photoIDStrVal := freelancerProfile.PhotoID.String()
		freelancerPhotoIDStr = &photoIDStrVal
		// Получаем file_path из media
		media, err := h.media.GetByID(c.Request.Context(), *freelancerProfile.PhotoID)
		if err == nil && media != nil {
			photoURLVal := media.FilePath
			freelancerPhotoURL = &photoURLVal
		}
	}

	// Получаем информацию о собеседнике (для other_user)
	otherUserProfile, err := h.users.GetProfile(c.Request.Context(), otherUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось получить профиль собеседника"})
		return
	}

	var otherUserPhotoIDStr *string
	var otherUserPhotoURL *string
	if otherUserProfile.PhotoID != nil {
		photoIDStrVal := otherUserProfile.PhotoID.String()
		otherUserPhotoIDStr = &photoIDStrVal
		// Получаем file_path из media
		media, err := h.media.GetByID(c.Request.Context(), *otherUserProfile.PhotoID)
		if err == nil && media != nil {
			photoURLVal := media.FilePath
			otherUserPhotoURL = &photoURLVal
		}
	}

	// Получаем сообщения
	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)
	messages, err := h.orders.ListMessages(c.Request.Context(), conversation.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"conversation": conversation,
		"messages":     messages,
		"order": gin.H{
			"id":          order.ID,
			"title":       order.Title,
			"description": order.Description,
			"status":      order.Status,
		},
		"freelancer": gin.H{
			"id":           acceptedProposal.FreelancerID,
			"display_name": freelancerProfile.DisplayName,
			"photo_id":     freelancerPhotoIDStr,
			"photo_url":    freelancerPhotoURL,
			"proposal": gin.H{
				"id":              acceptedProposal.ID,
				"cover_letter":    acceptedProposal.CoverLetter,
				"proposed_amount": acceptedProposal.ProposedAmount,
			},
		},
		"other_user": gin.H{
			"id":           otherUserID,
			"display_name": otherUserProfile.DisplayName,
			"photo_id":     otherUserPhotoIDStr,
			"photo_url":    otherUserPhotoURL,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetConversation обрабатывает GET /orders/:id/conversations/:participantId.
func (h *OrderHandler) GetConversation(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор участника"})
		return
	}

	order, err := h.orders.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var clientID, freelancerID uuid.UUID

	switch {
	case order.ClientID == userID && participantID != userID:
		clientID = order.ClientID
		freelancerID = participantID
	case order.ClientID == participantID && userID != participantID:
		clientID = order.ClientID
		freelancerID = userID
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "у вас нет доступа к этому чату"})
		return
	}

	conversation, err := h.orders.GetConversation(c.Request.Context(), orderID, clientID, freelancerID)
	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "чат не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)

	messages, err := h.orders.ListMessages(c.Request.Context(), conversation.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation": conversation,
		"messages":     messages,
	})
}

// ListMessages обрабатывает GET /conversations/:id/messages.
// Возвращает расширенную информацию: название заказа, информацию о собеседнике.
func (h *OrderHandler) ListMessages(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор чата"})
		return
	}

	conversation, err := h.orders.GetConversationByID(c.Request.Context(), conversationID)
	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "чат не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем доступ
	if conversation.ClientID != userID && conversation.FreelancerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "у вас нет доступа к этому чату"})
		return
	}

	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)

	messages, err := h.orders.ListMessages(c.Request.Context(), conversationID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Обогащаем ответ информацией о заказе и собеседнике
	response := gin.H{
		"conversation": conversation,
		"messages":     messages,
	}

	// Получаем полную информацию о заказе
	if conversation.OrderID != nil {
		order, err := h.orders.GetOrder(c.Request.Context(), *conversation.OrderID)
		if err == nil && order != nil {
			requirements, _ := h.orders.ListRequirements(c.Request.Context(), *conversation.OrderID)
			attachments, _ := h.orders.ListAttachments(c.Request.Context(), *conversation.OrderID)

			// Получаем принятое предложение
			var acceptedProposal *models.Proposal
			proposalsResult, err := h.orders.ListProposals(c.Request.Context(), *conversation.OrderID, nil)
			if err == nil && proposalsResult != nil {
				for _, p := range proposalsResult.Proposals {
					if p.Status == models.ProposalStatusAccepted {
						acceptedProposal = &p
						break
					}
				}
			}

			orderData := gin.H{
				"id":          order.ID,
				"title":       order.Title,
				"description": order.Description,
				"status":      order.Status,
				"budget_min":  order.BudgetMin,
				"budget_max":  order.BudgetMax,
				"deadline_at": order.DeadlineAt,
				"created_at":  order.CreatedAt,
				"updated_at":  order.UpdatedAt,
				"client_id":   order.ClientID,
			}

			if len(requirements) > 0 {
				orderData["requirements"] = requirements
			}
			if len(attachments) > 0 {
				orderData["attachments"] = attachments
			}
			if acceptedProposal != nil {
				orderData["accepted_proposal"] = gin.H{
					"id":              acceptedProposal.ID,
					"proposed_amount": acceptedProposal.ProposedAmount,
					"cover_letter":    acceptedProposal.CoverLetter,
				}
			}

			response["order"] = orderData
		}
	}

	// Определяем ID собеседника
	var otherUserID uuid.UUID
	if conversation.ClientID == userID {
		otherUserID = conversation.FreelancerID
	} else {
		otherUserID = conversation.ClientID
	}

	// Получаем информацию о собеседнике
	otherUser, err := h.users.GetByID(c.Request.Context(), otherUserID)
	if err == nil && otherUser != nil {
		profile, err := h.users.GetProfile(c.Request.Context(), otherUserID)
		if err == nil && profile != nil {
			var photoIDStr *string
			if profile.PhotoID != nil {
				photoIDStrVal := profile.PhotoID.String()
				photoIDStr = &photoIDStrVal
			}
			response["other_user"] = gin.H{
				"id":           otherUserID,
				"display_name": profile.DisplayName,
				"photo_id":     photoIDStr,
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// SendMessage обрабатывает POST /conversations/:id/messages.
func (h *OrderHandler) SendMessage(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор чата"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация содержимого сообщения
	if err := validation.ValidateMessageContent(req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, conversation, err := h.orders.SendMessage(c.Request.Context(), conversationID, userID, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "чат не найден"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	var orderSummary gin.H
	if conversation.OrderID != nil {
		if orderData, err := h.orders.GetOrder(c.Request.Context(), *conversation.OrderID); err == nil {
			orderSummary = gin.H{"id": orderData.ID, "title": orderData.Title}
		}
	}

	// WebSocket уведомления о новом сообщении
	if h.hub != nil {
		payload := gin.H{
			"message":      message,
			"conversation": conversation,
		}
		if orderSummary != nil {
			payload["order"] = orderSummary
		}

		// Уведомление получателю (не отправителю)
		if conversation.ClientID == userID {
			_ = h.hub.BroadcastToUser(conversation.FreelancerID, "chat.message", payload)
		} else {
			_ = h.hub.BroadcastToUser(conversation.ClientID, "chat.message", payload)
		}
	}

	response := gin.H{"message": message}
	if orderSummary != nil {
		response["order"] = orderSummary
	}
	c.JSON(http.StatusCreated, response)
}

// UpdateMessage обрабатывает PUT /conversations/:conversationId/messages/:messageId.
func (h *OrderHandler) UpdateMessage(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор чата"})
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор сообщения"})
		return
	}

	// Проверяем доступ к conversation
	conversation, err := h.orders.GetConversationByID(c.Request.Context(), conversationID)
	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "чат не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if conversation.ClientID != userID && conversation.FreelancerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "у вас нет доступа к этому чату"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация содержимого сообщения
	if err := validation.ValidateMessageContent(req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedMessage, err := h.orders.UpdateMessage(c.Request.Context(), messageID, userID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedMessage)
}

// DeleteMessage обрабатывает DELETE /conversations/:conversationId/messages/:messageId.
func (h *OrderHandler) DeleteMessage(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор чата"})
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор сообщения"})
		return
	}

	// Проверяем доступ к conversation
	conversation, err := h.orders.GetConversationByID(c.Request.Context(), conversationID)
	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "чат не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if conversation.ClientID != userID && conversation.FreelancerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "у вас нет доступа к этому чату"})
		return
	}

	if err := h.orders.DeleteMessage(c.Request.Context(), messageID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "сообщение успешно удалено"})
}

// ListMyOrders обрабатывает GET /orders/my - возвращает заказы текущего пользователя.
func (h *OrderHandler) ListMyOrders(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	clientOrders, freelancerOrders, err := h.orders.ListMyOrders(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"as_client":     clientOrders,
		"as_freelancer": freelancerOrders,
	})
}

// DeleteOrder обрабатывает DELETE /orders/:id.
func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	if err := h.orders.DeleteOrder(c.Request.Context(), orderID, userID); err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "заказ успешно удалён"})
}

// ListMyConversations обрабатывает GET /conversations/my - возвращает все чаты текущего пользователя.
// Возвращает расширенную информацию: название заказа, информацию о собеседнике, последнее сообщение.
func (h *OrderHandler) ListMyConversations(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conversations, err := h.orders.ListMyConversations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Обогащаем данные информацией о заказах и собеседниках
	type ConversationResponse struct {
		models.Conversation
		OrderTitle *string `json:"order_title,omitempty"`
		OtherUser  *struct {
			ID          uuid.UUID `json:"id"`
			DisplayName string    `json:"display_name"`
			PhotoID     *string   `json:"photo_id,omitempty"`
			PhotoURL    *string   `json:"photo_url,omitempty"`
		} `json:"other_user,omitempty"`
		LastMessage *models.Message `json:"last_message,omitempty"`
	}

	response := make([]ConversationResponse, 0, len(conversations))
	for _, conv := range conversations {
		resp := ConversationResponse{
			Conversation: conv,
		}

		// Получаем информацию о заказе
		if conv.OrderID != nil {
			order, err := h.orders.GetOrder(c.Request.Context(), *conv.OrderID)
			if err == nil && order != nil {
				resp.OrderTitle = &order.Title
			}
		}

		// Определяем ID собеседника
		var otherUserID uuid.UUID
		if conv.ClientID == userID {
			otherUserID = conv.FreelancerID
		} else {
			otherUserID = conv.ClientID
		}

		// Получаем информацию о собеседнике
		otherUser, err := h.users.GetByID(c.Request.Context(), otherUserID)
		if err == nil && otherUser != nil {
			profile, err := h.users.GetProfile(c.Request.Context(), otherUserID)
			if err == nil && profile != nil {
				var photoIDStr *string
				var photoURL *string
				if profile.PhotoID != nil {
					photoIDStrVal := profile.PhotoID.String()
					photoIDStr = &photoIDStrVal
					// Получаем file_path из media
					media, err := h.media.GetByID(c.Request.Context(), *profile.PhotoID)
					if err == nil && media != nil {
						photoURLVal := media.FilePath
						photoURL = &photoURLVal
					}
				}
				resp.OtherUser = &struct {
					ID          uuid.UUID `json:"id"`
					DisplayName string    `json:"display_name"`
					PhotoID     *string   `json:"photo_id,omitempty"`
					PhotoURL    *string   `json:"photo_url,omitempty"`
				}{
					ID:          otherUserID,
					DisplayName: profile.DisplayName,
					PhotoID:     photoIDStr,
					PhotoURL:    photoURL,
				}
			}
		}

		// Получаем последнее сообщение
		lastMessage, err := h.orders.GetLastMessageForConversation(c.Request.Context(), conv.ID)
		if err == nil && lastMessage != nil {
			resp.LastMessage = lastMessage
		}

		response = append(response, resp)
	}

	c.JSON(http.StatusOK, response)
}

// GenerateOrderDescription обрабатывает POST /ai/orders/description - генерирует описание заказа.
func (h *OrderHandler) GenerateOrderDescription(c *gin.Context) {
	userID, err := currentUserID(c)
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

// StreamGenerateOrderDescription обрабатывает POST /ai/orders/description/stream - генерирует описание заказа потоково.
func (h *OrderHandler) StreamGenerateOrderDescription(c *gin.Context) {
	userID, err := currentUserID(c)
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

// GenerateProposal обрабатывает POST /ai/orders/:id/proposal - генерирует отклик на заказ.
func (h *OrderHandler) GenerateProposal(c *gin.Context) {
	userID, err := currentUserID(c)
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

// StreamGenerateProposal обрабатывает GET /ai/orders/:id/proposal/stream -
// генерирует отклик на заказ потоково через SSE.
func (h *OrderHandler) StreamGenerateProposal(c *gin.Context) {
	userID, err := currentUserID(c)
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

// ImproveOrderDescription обрабатывает POST /ai/orders/improve - улучшает описание заказа.
func (h *OrderHandler) ImproveOrderDescription(c *gin.Context) {
	userID, err := currentUserID(c)
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

// StreamImproveOrderDescription обрабатывает POST /ai/orders/improve/stream - улучшает описание заказа потоково.
func (h *OrderHandler) StreamImproveOrderDescription(c *gin.Context) {
	userID, err := currentUserID(c)
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

// RegenerateOrderSummary обрабатывает POST /ai/orders/:id/regenerate-summary - регенерирует AI summary заказа.
func (h *OrderHandler) RegenerateOrderSummary(c *gin.Context) {
	userID, err := currentUserID(c)
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

// StreamRegenerateOrderSummary обрабатывает POST /ai/orders/:id/regenerate-summary/stream -
// регенерирует AI summary заказа потоково через SSE.
func (h *OrderHandler) StreamRegenerateOrderSummary(c *gin.Context) {
	userID, err := currentUserID(c)
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

// MarkOrderAsCompletedByFreelancer обрабатывает POST /orders/:id/complete-by-freelancer.
// Позволяет исполнителю отметить заказ как выполненный.
func (h *OrderHandler) MarkOrderAsCompletedByFreelancer(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	// Получаем заказ
	order, err := h.orders.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что заказ в статусе in_progress
	if order.Status != models.OrderStatusInProgress {
		c.JSON(http.StatusBadRequest, gin.H{"error": "заказ должен быть в статусе 'in_progress'"})
		return
	}

	// Проверяем, что пользователь является исполнителем заказа
	proposalsResult, err := h.orders.ListProposals(c.Request.Context(), orderID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var isAcceptedFreelancer bool
	if proposalsResult != nil {
		for _, proposal := range proposalsResult.Proposals {
			if proposal.FreelancerID == userID && proposal.Status == models.ProposalStatusAccepted {
				isAcceptedFreelancer = true
				break
			}
		}
	}

	if !isAcceptedFreelancer {
		c.JSON(http.StatusForbidden, gin.H{"error": "только принятый исполнитель может отметить заказ как выполненный"})
		return
	}

	// Получаем требования и вложения заказа, чтобы не потерять их при обновлении
	requirements, err := h.orders.ListRequirements(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	attachments, err := h.orders.ListAttachments(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем ID вложений
	attachmentIDs := make([]uuid.UUID, 0, len(attachments))
	for _, att := range attachments {
		attachmentIDs = append(attachmentIDs, att.MediaID)
	}

	// Обновляем статус заказа на completed, сохраняя все существующие данные
	updated, err := h.orders.UpdateOrder(c.Request.Context(), service.UpdateOrderInput{
		OrderID:       orderID,
		ClientID:      order.ClientID, // Используем оригинального клиента
		Title:         order.Title,    // Сохраняем существующий заголовок
		Description:   order.Description, // Сохраняем существующее описание
		BudgetMin:     order.BudgetMin,
		BudgetMax:     order.BudgetMax,
		DeadlineAt:    order.DeadlineAt,
		Status:        models.OrderStatusCompleted,
		Requirements:  requirements,
		AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// WebSocket уведомления
	if h.hub != nil {
		payload := gin.H{
			"order":   updated,
			"message": "Исполнитель отметил заказ как выполненный",
		}
		_ = h.hub.BroadcastToUser(order.ClientID, "orders.updated", payload)
		_ = h.hub.BroadcastToUser(userID, "orders.updated", payload)
	}

	c.JSON(http.StatusOK, gin.H{
		"order":   updated,
		"message": "Заказ успешно отмечен как выполненный",
	})
}

// GetProposalFeedback обрабатывает GET /ai/orders/:id/proposals/feedback - получает рекомендации по улучшению отклика.
func (h *OrderHandler) GetProposalFeedback(c *gin.Context) {
	userID, err := currentUserID(c)
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
func (h *OrderHandler) StreamProposalFeedback(c *gin.Context) {
	userID, err := currentUserID(c)
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

// SummarizeConversation создаёт резюме переписки в чате.
func (h *OrderHandler) SummarizeConversation(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор чата"})
		return
	}

	summary, err := h.orders.SummarizeConversation(c.Request.Context(), conversationID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// StreamSummarizeConversation создаёт резюме переписки потоково.
func (h *OrderHandler) StreamSummarizeConversation(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор чата"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "стриминг не поддерживается"})
		return
	}

	err = h.orders.StreamSummarizeConversation(c.Request.Context(), conversationID, userID, func(chunk string) error {
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

// RecommendRelevantOrders рекомендует подходящие заказы для фрилансера.
func (h *OrderHandler) RecommendRelevantOrders(c *gin.Context) {
	userID, err := currentUserID(c)
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

	limit := parseIntQuery(c, "limit", 10)
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

// StreamRecommendRelevantOrders рекомендует подходящие заказы для фрилансера потоково.
func (h *OrderHandler) StreamRecommendRelevantOrders(c *gin.Context) {
	userID, err := currentUserID(c)
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

	limit := parseIntQuery(c, "limit", 10)
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
				"explanation":            generalExplanation,
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

// RecommendPriceAndTimeline рекомендует цену и сроки для отклика.
func (h *OrderHandler) RecommendPriceAndTimeline(c *gin.Context) {
	userID, err := currentUserID(c)
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

// StreamRecommendPriceAndTimeline рекомендует цену и сроки для отклика потоково.
func (h *OrderHandler) StreamRecommendPriceAndTimeline(c *gin.Context) {
	userID, err := currentUserID(c)
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

// EvaluateOrderQuality оценивает качество заказа.
func (h *OrderHandler) EvaluateOrderQuality(c *gin.Context) {
	userID, err := currentUserID(c)
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

// StreamEvaluateOrderQuality оценивает качество заказа потоково.
func (h *OrderHandler) StreamEvaluateOrderQuality(c *gin.Context) {
	userID, err := currentUserID(c)
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

// FindSuitableFreelancers находит подходящих фрилансеров для заказа.
func (h *OrderHandler) FindSuitableFreelancers(c *gin.Context) {
	userID, err := currentUserID(c)
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

	limit := parseIntQuery(c, "limit", 10)
	freelancers, err := h.orders.FindSuitableFreelancers(c.Request.Context(), orderID, userID, user.Role, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"freelancers": freelancers,
	})
}

// StreamFindSuitableFreelancers находит подходящих фрилансеров для заказа потоково.
func (h *OrderHandler) StreamFindSuitableFreelancers(c *gin.Context) {
	userID, err := currentUserID(c)
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

	limit := parseIntQuery(c, "limit", 10)

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

// AIChatAssistant обрабатывает запросы к AI помощнику.
func (h *OrderHandler) AIChatAssistant(c *gin.Context) {
	userID, err := currentUserID(c)
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

// StreamAIChatAssistant обрабатывает запросы к AI помощнику потоково.
func (h *OrderHandler) StreamAIChatAssistant(c *gin.Context) {
	userID, err := currentUserID(c)
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

// ImproveProfile улучшает описание профиля с помощью AI.
func (h *OrderHandler) ImproveProfile(c *gin.Context) {
	userID, err := currentUserID(c)
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

// StreamImproveProfile обрабатывает POST /ai/profile/improve/stream - улучшает описание профиля потоково.
func (h *OrderHandler) StreamImproveProfile(c *gin.Context) {
	userID, err := currentUserID(c)
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

// ImprovePortfolioItem улучшает описание работы в портфолио с помощью AI.
func (h *OrderHandler) ImprovePortfolioItem(c *gin.Context) {
	_, err := currentUserID(c)
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

// StreamImprovePortfolioItem обрабатывает POST /ai/portfolio/improve/stream - улучшает описание работы в портфолио потоково.
func (h *OrderHandler) StreamImprovePortfolioItem(c *gin.Context) {
	_, err := currentUserID(c)
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

// parseIntQuery безопасно читает query параметр.
func parseIntQuery(c *gin.Context, key string, fallback int) int {
	if v := c.Query(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

// contains проверяет, содержит ли строка подстроку (для проверки типа ошибки).
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
