package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/interface/http/dto"
	"github.com/ignatzorin/freelance-backend/internal/interface/http/response"
	"github.com/ignatzorin/freelance-backend/internal/usecase/conversation"
)

type ConversationHandler struct {
	getOrCreateConvUC *conversation.GetOrCreateConversationUseCase
	listMyConvsUC     *conversation.ListMyConversationsUseCase
	sendMessageUC     *conversation.SendMessageUseCase
	listMessagesUC    *conversation.ListMessagesUseCase
	updateMessageUC   *conversation.UpdateMessageUseCase
	deleteMessageUC   *conversation.DeleteMessageUseCase
	addReactionUC     *conversation.AddReactionUseCase
	removeReactionUC  *conversation.RemoveReactionUseCase
}

func NewConversationHandler(
	getOrCreateConvUC *conversation.GetOrCreateConversationUseCase,
	listMyConvsUC *conversation.ListMyConversationsUseCase,
	sendMessageUC *conversation.SendMessageUseCase,
	listMessagesUC *conversation.ListMessagesUseCase,
	updateMessageUC *conversation.UpdateMessageUseCase,
	deleteMessageUC *conversation.DeleteMessageUseCase,
	addReactionUC *conversation.AddReactionUseCase,
	removeReactionUC *conversation.RemoveReactionUseCase,
) *ConversationHandler {
	return &ConversationHandler{
		getOrCreateConvUC: getOrCreateConvUC,
		listMyConvsUC:     listMyConvsUC,
		sendMessageUC:     sendMessageUC,
		listMessagesUC:    listMessagesUC,
		updateMessageUC:   updateMessageUC,
		deleteMessageUC:   deleteMessageUC,
		addReactionUC:     addReactionUC,
		removeReactionUC:  removeReactionUC,
	}
}

func (h *ConversationHandler) GetConversation(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "некорректный ID заказа")
		return
	}

	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID участника")
		return
	}

	conv, err := h.getOrCreateConvUC.Execute(c.Request.Context(), orderID, userID, participantID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToConversationResponse(conv))
}

func (h *ConversationHandler) ListMyConversations(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	convs, err := h.listMyConvsUC.Execute(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToConversationResponses(convs))
}

func (h *ConversationHandler) SendMessage(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID беседы")
		return
	}

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "некорректные данные запроса")
		return
	}

	msg, err := h.sendMessageUC.Execute(c.Request.Context(), conversationID, userID, req.Content)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToMessageResponse(msg))
}

func (h *ConversationHandler) ListMessages(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID беседы")
		return
	}

	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)

	messages, err := h.listMessagesUC.Execute(c.Request.Context(), conversationID, userID, limit, offset)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToMessageResponses(messages))
}

func (h *ConversationHandler) UpdateMessage(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID сообщения")
		return
	}

	var req dto.UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "некорректные данные запроса")
		return
	}

	msg, err := h.updateMessageUC.Execute(c.Request.Context(), messageID, userID, req.Content)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToMessageResponse(msg))
}

func (h *ConversationHandler) DeleteMessage(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID сообщения")
		return
	}

	if err := h.deleteMessageUC.Execute(c.Request.Context(), messageID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "сообщение удалено"})
}

func (h *ConversationHandler) AddReaction(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID сообщения")
		return
	}

	var req dto.AddReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "некорректные данные запроса")
		return
	}

	reaction, err := h.addReactionUC.Execute(c.Request.Context(), messageID, userID, req.Emoji)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToReactionResponse(reaction))
}

func (h *ConversationHandler) RemoveReaction(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID сообщения")
		return
	}

	if err := h.removeReactionUC.Execute(c.Request.Context(), messageID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "реакция удалена"})
}
