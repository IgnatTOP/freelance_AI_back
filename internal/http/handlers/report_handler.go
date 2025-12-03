package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

type ReportHandler struct {
	svc *service.ReportService
}

func NewReportHandler(s *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: s}
}

// CreateReport POST /reports
func (h *ReportHandler) CreateReport(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	var req struct {
		TargetType  string  `json:"target_type" binding:"required"`
		TargetID    string  `json:"target_id" binding:"required,uuid"`
		Reason      string  `json:"reason" binding:"required"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondBadRequest(c, err.Error())
		return
	}

	targetID, _ := uuid.Parse(req.TargetID)
	report, err := h.svc.CreateReport(c.Request.Context(), userID, req.TargetType, targetID, req.Reason, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, report)
}

// ListMyReports GET /reports
func (h *ReportHandler) ListMyReports(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		common.RespondUnauthorized(c, err.Error())
		return
	}

	limit, offset := common.GetPagination(c)
	reports, err := h.svc.ListMyReports(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, reports)
}
