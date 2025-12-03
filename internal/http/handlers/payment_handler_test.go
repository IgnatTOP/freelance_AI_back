package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPaymentHandler_GetBalance_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &PaymentHandler{payments: nil}
	r.GET("/payments/balance", handler.GetBalance)

	req, _ := http.NewRequest("GET", "/payments/balance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPaymentHandler_GetEscrow_InvalidOrderID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &PaymentHandler{payments: nil}
	r.GET("/payments/escrow/:orderId", handler.GetEscrow)

	req, _ := http.NewRequest("GET", "/payments/escrow/invalid-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPaymentHandler_ListTransactions_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &PaymentHandler{payments: nil}
	r.GET("/payments/transactions", handler.ListTransactions)

	req, _ := http.NewRequest("GET", "/payments/transactions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPaymentHandler_Deposit_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &PaymentHandler{payments: nil}
	r.POST("/payments/deposit", handler.Deposit)

	req, _ := http.NewRequest("POST", "/payments/deposit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPaymentHandler_CreateEscrow_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &PaymentHandler{payments: nil}
	r.POST("/payments/escrow", handler.CreateEscrow)

	req, _ := http.NewRequest("POST", "/payments/escrow", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
