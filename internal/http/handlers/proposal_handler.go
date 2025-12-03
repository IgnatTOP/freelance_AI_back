package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

// ProposalHandler отвечает за работу с предложениями.
type ProposalHandler struct {
	orders *repository.OrderRepository
}

// NewProposalHandler создаёт экземпляр.
func NewProposalHandler(orders *repository.OrderRepository) *ProposalHandler {
	return &ProposalHandler{orders: orders}
}

// ListMyProposals возвращает все предложения текущего пользователя.
func (h *ProposalHandler) ListMyProposals(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	proposals, err := h.orders.ListMyProposals(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось получить предложения"})
		return
	}

	// Собираем уникальные order_id
	orderIDs := make(map[uuid.UUID]struct{})
	for _, p := range proposals {
		orderIDs[p.OrderID] = struct{}{}
	}

	// Загружаем заказы по идентификаторам
	ordersByID := make(map[uuid.UUID]*models.Order)
	for id := range orderIDs {
		order, err := h.orders.GetByID(c.Request.Context(), id)
		if err == nil && order != nil {
			ordersByID[id] = order
		}
	}

	// Формируем ответ с вложенными заказами (минимальный срез полей)
	type OrderShort struct {
		ID     uuid.UUID `json:"id"`
		Title  string    `json:"title"`
		Status string    `json:"status"`
	}

	type ProposalWithOrder struct {
		models.Proposal
		Order *OrderShort `json:"order,omitempty"`
	}

	resp := make([]ProposalWithOrder, len(proposals))
	for i, p := range proposals {
		var orderShort *OrderShort
		if o, ok := ordersByID[p.OrderID]; ok {
			orderShort = &OrderShort{
				ID:     o.ID,
				Title:  o.Title,
				Status: string(o.Status),
			}
		}
		resp[i] = ProposalWithOrder{
			Proposal: p,
			Order:    orderShort,
		}
	}

	c.JSON(http.StatusOK, resp)
}

