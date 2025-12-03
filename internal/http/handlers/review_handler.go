package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

type ReviewHandler struct {
	reviews *service.ReviewService
}

func NewReviewHandler(reviews *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviews: reviews}
}

// CreateReview POST /orders/:id/reviews
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "неверный order_id")
		return
	}

	var req struct {
		Rating  int     `json:"rating" binding:"required,min=1,max=5"`
		Comment *string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, "рейтинг должен быть от 1 до 5")
		return
	}

	review, err := h.reviews.CreateReview(c.Request.Context(), orderID, userID, req.Rating, req.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, review)
}

// ListOrderReviews GET /orders/:id/reviews
func (h *ReviewHandler) ListOrderReviews(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "неверный order_id")
		return
	}

	reviews, err := h.reviews.ListOrderReviews(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reviews": reviews})
}

// ListUserReviews GET /users/:id/reviews
func (h *ReviewHandler) ListUserReviews(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "неверный user_id")
		return
	}

	limit := common.ParseIntQuery(c, "limit", 20)
	offset := common.ParseIntQuery(c, "offset", 0)

	reviews, err := h.reviews.ListUserReviews(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	avg, count, _ := h.reviews.GetUserRating(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"reviews":        reviews,
		"average_rating": avg,
		"total_reviews":  count,
	})
}

// CanLeaveReview GET /orders/:id/can-review
func (h *ReviewHandler) CanLeaveReview(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "неверный order_id")
		return
	}

	canReview, err := h.reviews.CanLeaveReview(c.Request.Context(), orderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"can_review": canReview})
}
