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

// ConversationHandler обслуживает маршруты чатов и сообщений
type ConversationHandler struct {
	orders *service.OrderService
	users  *repository.UserRepository
	media  *repository.MediaRepository
	hub    *ws.Hub
}

// NewConversationHandler создаёт новый хэндлер.
func NewConversationHandler(orders *service.OrderService, users *repository.UserRepository, media *repository.MediaRepository, hub *ws.Hub) *ConversationHandler {
	return &ConversationHandler{orders: orders, users: users, media: media, hub: hub}
}

func (h *ConversationHandler) GetOrderChat(c *gin.Context) {
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
	limit := common.ParseIntQuery(c, "limit", 50)
	offset := common.ParseIntQuery(c, "offset", 0)
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

func (h *ConversationHandler) GetConversation(c *gin.Context) {
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

	limit := common.ParseIntQuery(c, "limit", 50)
	offset := common.ParseIntQuery(c, "offset", 0)

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

func (h *ConversationHandler) ListMessages(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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

	limit := common.ParseIntQuery(c, "limit", 50)
	offset := common.ParseIntQuery(c, "offset", 0)

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

func (h *ConversationHandler) SendMessage(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор чата"})
		return
	}

	var req dto.SendMessageRequest
	if err := common.BindAndValidate(c, &req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	content := req.Content
	
	// Парсим parent message ID используя метод из DTO
	parentMessageID, err := req.ParseParentMessageID()
	if err != nil {
		common.RespondBadRequest(c, fmt.Sprintf("parent_message_id содержит некорректный UUID: %v", err))
		return
	}

	// Парсим attachment IDs используя метод из DTO
	attachmentMediaIDs, err := req.ParseAttachmentIDs()
	if err != nil {
		common.RespondBadRequest(c, fmt.Sprintf("attachment_ids содержит некорректный UUID: %v", err))
		return
	}

	// Валидация: сообщение должно содержать текст или вложения
	if content == "" && len(attachmentMediaIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "сообщение должно содержать текст или вложения"})
		return
	}

	// Валидация содержимого сообщения (если есть текст)
	if content != "" {
		if err := validation.ValidateMessageContent(content); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	message, conversation, err := h.orders.SendMessage(c.Request.Context(), conversationID, userID, content, parentMessageID, attachmentMediaIDs)
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

func (h *ConversationHandler) UpdateMessage(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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

	var req dto.UpdateMessageRequest
	if err := common.BindAndValidate(c, &req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	// Валидация содержимого сообщения
	if err := validation.ValidateMessageContent(req.Content); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	updatedMessage, err := h.orders.UpdateMessage(c.Request.Context(), messageID, userID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedMessage)
}

func (h *ConversationHandler) DeleteMessage(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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

func (h *ConversationHandler) AddMessageReaction(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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

	var req dto.AddMessageReactionRequest
	if err := common.BindAndValidate(c, &req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	reaction, err := h.orders.AddMessageReaction(c.Request.Context(), messageID, userID, req.Emoji)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// WebSocket уведомление о новой реакции
	if h.hub != nil {
		payload := gin.H{
			"conversation_id": conversationID,
			"message_id":      messageID,
			"reaction":        reaction,
		}
		// Уведомляем обоих участников чата
		_ = h.hub.BroadcastToUser(conversation.ClientID, "message.reaction.added", payload)
		if conversation.FreelancerID != conversation.ClientID {
			_ = h.hub.BroadcastToUser(conversation.FreelancerID, "message.reaction.added", payload)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"reaction": reaction})
}

func (h *ConversationHandler) RemoveMessageReaction(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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

	if err := h.orders.RemoveMessageReaction(c.Request.Context(), messageID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// WebSocket уведомление об удалении реакции
	if h.hub != nil {
		payload := gin.H{
			"conversation_id": conversationID,
			"message_id":      messageID,
			"user_id":         userID,
		}
		_ = h.hub.BroadcastToUser(conversation.ClientID, "message.reaction.removed", payload)
		if conversation.FreelancerID != conversation.ClientID {
			_ = h.hub.BroadcastToUser(conversation.FreelancerID, "message.reaction.removed", payload)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "реакция успешно удалена"})
}

func (h *ConversationHandler) ListMyConversations(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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

func (h *ConversationHandler) SummarizeConversation(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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

func (h *ConversationHandler) StreamSummarizeConversation(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
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
