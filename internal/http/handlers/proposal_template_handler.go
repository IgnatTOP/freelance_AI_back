package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

type ProposalTemplateHandler struct {
	svc *service.ProposalTemplateService
}

func NewProposalTemplateHandler(s *service.ProposalTemplateService) *ProposalTemplateHandler {
	return &ProposalTemplateHandler{svc: s}
}

// CreateTemplate POST /proposal-templates
func (h *ProposalTemplateHandler) CreateTemplate(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	t, err := h.svc.Create(c.Request.Context(), userID, req.Title, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

// ListTemplates GET /proposal-templates
func (h *ProposalTemplateHandler) ListTemplates(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	templates, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

// UpdateTemplate PUT /proposal-templates/:id
func (h *ProposalTemplateHandler) UpdateTemplate(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "invalid template_id")
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	if err := h.svc.Update(c.Request.Context(), userID, templateID, req.Title, req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DeleteTemplate DELETE /proposal-templates/:id
func (h *ProposalTemplateHandler) DeleteTemplate(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.RespondBadRequest(c, "invalid template_id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, templateID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
