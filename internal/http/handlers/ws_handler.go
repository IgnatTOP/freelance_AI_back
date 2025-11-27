package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/ignatzorin/freelance-backend/internal/service"
	"github.com/ignatzorin/freelance-backend/internal/ws"
)

// WSHandler отвечает за установку WebSocket соединений.
type WSHandler struct {
	hub          *ws.Hub
	tokenManager *service.TokenManager
	upgrader     websocket.Upgrader
}

// NewWSHandler создаёт новый хэндлер.
func NewWSHandler(hub *ws.Hub, tokens *service.TokenManager) *WSHandler {
	return &WSHandler{
		hub:          hub,
		tokenManager: tokens,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// TokenManager возвращает менеджер токенов (используется в middleware).
func (h *WSHandler) TokenManager() *service.TokenManager {
	return h.tokenManager
}

// Handle обслуживает GET /api/ws?token=...
func (h *WSHandler) Handle(c *gin.Context) {
	rawToken := c.Query("token")
	if rawToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access токен обязателен"})
		return
	}

	userID, _, err := h.tokenManager.ParseAccess(rawToken)
	if err != nil || userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "невалидный access токен"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	client := ws.NewClient(conn, h.hub, userID)
	h.hub.Register(client)

	client.Run(c.Request.Context())
}
