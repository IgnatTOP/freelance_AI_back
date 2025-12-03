package handlers

import (
	"errors"
	"fmt"
	"net/http"

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

// ProposalOperationsHandler обслуживает маршруты операций с предложениями
type ProposalOperationsHandler struct {
	orders *service.OrderService
	users  *repository.UserRepository
	media  *repository.MediaRepository
	hub    *ws.Hub
}

// NewProposalOperationsHandler создаёт новый хэндлер.
func NewProposalOperationsHandler(orders *service.OrderService, users *repository.UserRepository, media *repository.MediaRepository, hub *ws.Hub) *ProposalOperationsHandler {
	return &ProposalOperationsHandler{orders: orders, users: users, media: media, hub: hub}
}

func (h *ProposalOperationsHandler) CreateProposal(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "только исполнители могут создавать предложения к заказам"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор заказа"})
		return
	}

	var req dto.CreateProposalRequest
	if err := common.BindAndValidate(c, &req); err != nil {
		common.RespondBadRequest(c, err.Error())
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

func (h *ProposalOperationsHandler) ListProposals(c *gin.Context) {
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

func (h *ProposalOperationsHandler) GetMyProposal(c *gin.Context) {
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

func (h *ProposalOperationsHandler) UpdateProposalStatus(c *gin.Context) {
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

	proposalID, err := uuid.Parse(c.Param("proposalId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор отклика"})
		return
	}

	var req dto.UpdateProposalStatusRequest
	if err := common.BindAndValidate(c, &req); err != nil {
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

// MarkOrderAsCompletedByFreelancer обрабатывает POST /orders/:id/complete-by-freelancer.
// Позволяет исполнителю отметить заказ как выполненный.
func (h *ProposalOperationsHandler) MarkOrderAsCompletedByFreelancer(c *gin.Context) {
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
		ClientID:      order.ClientID,    // Используем оригинального клиента
		Title:         order.Title,       // Сохраняем существующий заголовок
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
