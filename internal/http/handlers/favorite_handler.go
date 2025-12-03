package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

type FavoriteHandler struct {
	svc *service.FavoriteService
}

func NewFavoriteHandler(s *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{svc: s}
}

// AddFavorite POST /favorites
func (h *FavoriteHandler) AddFavorite(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	var req struct {
		TargetType string `json:"target_type" binding:"required"`
		TargetID   string `json:"target_id" binding:"required,uuid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	targetID, _ := uuid.Parse(req.TargetID)
	fav, err := h.svc.AddFavorite(c.Request.Context(), userID, req.TargetType, targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, fav)
}

// RemoveFavorite DELETE /favorites/:type/:id
func (h *FavoriteHandler) RemoveFavorite(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	targetType := c.Param("type")
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "invalid target_id")
		return
	}

	if err := h.svc.RemoveFavorite(c.Request.Context(), userID, targetType, targetID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}

// ListFavorites GET /favorites
func (h *FavoriteHandler) ListFavorites(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	targetType := c.Query("type")
	limit, offset := common.GetPagination(c)

	favorites, err := h.svc.ListFavorites(c.Request.Context(), userID, targetType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, favorites)
}

// CheckFavorite GET /favorites/:type/:id
func (h *FavoriteHandler) CheckFavorite(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	targetType := c.Param("type")
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "invalid target_id")
		return
	}

	isFav, err := h.svc.IsFavorite(c.Request.Context(), userID, targetType, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_favorite": isFav})
}
