package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

type WithdrawalHandler struct {
	svc *service.WithdrawalService
}

func NewWithdrawalHandler(s *service.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{svc: s}
}

// CreateWithdrawal POST /withdrawals
func (h *WithdrawalHandler) CreateWithdrawal(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	var req struct {
		Amount    float64 `json:"amount" binding:"required,gt=0"`
		CardLast4 string  `json:"card_last4" binding:"required,len=4"`
		BankName  string  `json:"bank_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	w, err := h.svc.CreateWithdrawal(c.Request.Context(), userID, req.Amount, req.CardLast4, req.BankName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, w)
}

// ListWithdrawals GET /withdrawals
func (h *WithdrawalHandler) ListWithdrawals(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	limit, offset := common.GetPagination(c)
	withdrawals, err := h.svc.ListUserWithdrawals(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, withdrawals)
}
