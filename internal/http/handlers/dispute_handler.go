package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

type DisputeHandler struct {
	svc *service.DisputeService
}

func NewDisputeHandler(s *service.DisputeService) *DisputeHandler {
	return &DisputeHandler{svc: s}
}

// CreateDispute POST /orders/:id/dispute
func (h *DisputeHandler) CreateDispute(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "invalid order_id")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	dispute, err := h.svc.CreateDispute(c.Request.Context(), orderID, userID, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dispute)
}

// GetDispute GET /orders/:id/dispute
func (h *DisputeHandler) GetDispute(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "invalid order_id")
		return
	}

	dispute, err := h.svc.GetDispute(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dispute)
}

// ListMyDisputes GET /disputes
func (h *DisputeHandler) ListMyDisputes(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	limit, offset := common.GetPagination(c)
	disputes, err := h.svc.ListUserDisputes(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, disputes)
}
