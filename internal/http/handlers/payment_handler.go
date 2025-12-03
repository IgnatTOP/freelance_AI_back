package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

type PaymentHandler struct {
	payments *service.PaymentService
}

func NewPaymentHandler(payments *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

// GetBalance GET /payments/balance
func (h *PaymentHandler) GetBalance(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	balance, err := h.payments.GetBalance(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}

// Deposit POST /payments/deposit
func (h *PaymentHandler) Deposit(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, "сумма должна быть положительной")
		return
	}

	transaction, err := h.payments.Deposit(c.Request.Context(), userID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// CreateEscrow POST /payments/escrow
func (h *PaymentHandler) CreateEscrow(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	var req struct {
		OrderID      string  `json:"order_id" binding:"required"`
		FreelancerID string  `json:"freelancer_id" binding:"required"`
		Amount       float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		common.RespondBadRequest(c, "неверный order_id")
		return
	}
	freelancerID, err := uuid.Parse(req.FreelancerID)
	if err != nil {
		common.RespondBadRequest(c, "неверный freelancer_id")
		return
	}

	escrow, err := h.payments.CreateEscrow(c.Request.Context(), orderID, userID, freelancerID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, escrow)
}

// GetEscrow GET /payments/escrow/:orderId
func (h *PaymentHandler) GetEscrow(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		common.RespondBadRequest(c, "неверный order_id")
		return
	}

	escrow, err := h.payments.GetEscrow(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "escrow не найден"})
		return
	}

	c.JSON(http.StatusOK, escrow)
}

// ListTransactions GET /payments/transactions
func (h *PaymentHandler) ListTransactions(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	limit := common.ParseIntQuery(c, "limit", 20)
	offset := common.ParseIntQuery(c, "offset", 0)

	transactions, err := h.payments.ListTransactions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}
