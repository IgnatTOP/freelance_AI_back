package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/dto"
	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/service"
	"github.com/ignatzorin/freelance-backend/internal/validation"
	"github.com/ignatzorin/freelance-backend/internal/ws"
)

type OrderHandler struct {
	orders *service.OrderService
	users  *repository.UserRepository
	media  *repository.MediaRepository
	hub    *ws.Hub
	cache  *service.CacheService
}

// NewOrderHandler создаёт новый хэндлер.
func NewOrderHandler(orders *service.OrderService, users *repository.UserRepository, media *repository.MediaRepository, hub *ws.Hub, cache *service.CacheService) *OrderHandler {
	return &OrderHandler{orders: orders, users: users, media: media, hub: hub, cache: cache}
}

// CreateOrder обрабатывает POST /orders.
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не найден"})
		return
	}
	if user.Role != "client" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "только заказчики могут создавать заказы"})
		return
	}

	var req dto.CreateOrderRequest
	if err := common.BindAndValidate(c, &req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if err := validation.ValidateOrderTitle(req.Title); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if err := validation.ValidateOrderDescription(req.Description); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if err := validation.ValidateBudget(req.BudgetMin, req.BudgetMax); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if req.Requirements != nil {
		for _, r := range req.Requirements {
			if err := validation.ValidateRequirementSkill(r.Skill); err != nil {
				common.RespondBadRequest(c, err.Error())
				return
			}
		}
	}

	deadline, err := req.ParseDeadline()
	if err != nil {
		common.RespondBadRequest(c, "deadline_at должен быть в формате RFC3339")
		return
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

	attachmentIDs, err := req.ParseAttachmentIDs()
	if err != nil {
		common.RespondBadRequest(c, fmt.Sprintf("attachment_ids содержит некорректный UUID: %v", err))
		return
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
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if contains(err.Error(), "не может быть") || contains(err.Error(), "некорректный") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	requirements, errReq := h.orders.ListRequirements(c.Request.Context(), order.ID)
	if errReq != nil {
		requirements = []models.OrderRequirement{}
	}

	attachments, errAtt := h.orders.ListAttachments(c.Request.Context(), order.ID)
	if errAtt != nil {
		attachments = []models.OrderAttachment{}
	}

	if h.hub != nil {
		_ = h.hub.BroadcastToUser(userID, "orders.new", gin.H{
			"order":   order,
			"message": "Заказ успешно создан",
		})
	}

	// Invalidate cache
	if h.cache != nil {
		h.cache.InvalidateUserCache(userID)
		h.cache.InvalidateOrderCache(order.ID)
	}

	c.JSON(http.StatusCreated, dto.NewOrderResponse(order, requirements, attachments))
}

// ListOrders обрабатывает GET /orders.
func (h *OrderHandler) ListOrders(c *gin.Context) {
	status := c.Query("status")
	if status == "open" {
		status = "published"
	}

	params := repository.ListFilterParams{
		Status:    status,
		Search:    c.Query("search"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Limit:     common.ParseIntQuery(c, "limit", 20),
		Offset:    common.ParseIntQuery(c, "offset", 0),
	}

	if skillsParam := c.Query("skills"); skillsParam != "" {
		params.Skills = strings.Split(skillsParam, ",")
		for i := range params.Skills {
			params.Skills[i] = strings.TrimSpace(params.Skills[i])
		}
	}

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

	c.JSON(http.StatusOK, dto.NewOrderResponse(order, requirements, attachments))
}

// UpdateOrder обрабатывает PUT /orders/:id.
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
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

	var req dto.UpdateOrderRequest
	if err := common.BindAndValidate(c, &req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if err := validation.ValidateOrderTitle(req.Title); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if err := validation.ValidateOrderDescription(req.Description); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if err := validation.ValidateBudget(req.BudgetMin, req.BudgetMax); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if req.Requirements != nil {
		for _, r := range req.Requirements {
			if err := validation.ValidateRequirementSkill(r.Skill); err != nil {
				common.RespondBadRequest(c, err.Error())
				return
			}
		}
	}

	status := req.Status
	if status == "" {
		status = existing.Status
	}
	if _, ok := models.ValidOrderStatuses[status]; !ok {
		common.RespondBadRequest(c, "некорректный статус заказа")
		return
	}

	deadline, err := req.ParseDeadline()
	if err != nil {
		common.RespondBadRequest(c, "deadline_at должен быть в формате RFC3339")
		return
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

	attachmentIDs, err := req.ParseAttachmentIDs()
	if err != nil {
		common.RespondBadRequest(c, fmt.Sprintf("attachment_ids содержит некорректный UUID: %v", err))
		return
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

	if h.hub != nil {
		_ = h.hub.BroadcastToUser(userID, "orders.updated", wsPayload)

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

	updated.Attachments = attachments

	// Invalidate cache
	if h.cache != nil {
		h.cache.InvalidateUserCache(userID)
		h.cache.InvalidateOrderCache(orderID)
	}

	c.JSON(http.StatusOK, dto.NewOrderResponse(updated, requirements, attachments))
}

// ListMyOrders обрабатывает GET /orders/my - возвращает заказы текущего пользователя.
func (h *OrderHandler) ListMyOrders(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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

	if err := h.orders.DeleteOrder(c.Request.Context(), orderID, userID); err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заказ не найден"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Invalidate cache
	if h.cache != nil {
		h.cache.InvalidateUserCache(userID)
		h.cache.InvalidateOrderCache(orderID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "заказ успешно удалён"})
}

// contains проверяет, содержит ли строка подстроку (для проверки типа ошибки).
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
