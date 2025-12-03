package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReviewHandler_CreateReview_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &ReviewHandler{reviews: nil}
	r.POST("/orders/:id/reviews", handler.CreateReview)

	orderID := uuid.New()
	req, _ := http.NewRequest("POST", "/orders/"+orderID.String()+"/reviews", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestReviewHandler_ListOrderReviews_InvalidOrderID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &ReviewHandler{reviews: nil}
	r.GET("/orders/:id/reviews", handler.ListOrderReviews)

	req, _ := http.NewRequest("GET", "/orders/invalid-uuid/reviews", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReviewHandler_ListUserReviews_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &ReviewHandler{reviews: nil}
	r.GET("/users/:id/reviews", handler.ListUserReviews)

	req, _ := http.NewRequest("GET", "/users/invalid-uuid/reviews", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReviewHandler_CanLeaveReview_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := &ReviewHandler{reviews: nil}
	r.GET("/orders/:id/can-review", handler.CanLeaveReview)

	orderID := uuid.New()
	req, _ := http.NewRequest("GET", "/orders/"+orderID.String()+"/can-review", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestReviewHandler_CanLeaveReview_InvalidOrderID_WithAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	userID := uuid.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID) // Используем правильный ключ и тип uuid.UUID
		c.Next()
	})
	handler := &ReviewHandler{reviews: nil}
	r.GET("/orders/:id/can-review", handler.CanLeaveReview)

	// С авторизацией, но невалидный UUID
	req, _ := http.NewRequest("GET", "/orders/invalid-uuid/can-review", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
