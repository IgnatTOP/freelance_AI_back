package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/interface/http/dto"
	"github.com/ignatzorin/freelance-backend/internal/interface/http/response"
	"github.com/ignatzorin/freelance-backend/internal/usecase/proposal"
)

type ProposalHandler struct {
	createProposalUC       *proposal.CreateProposalUseCase
	updateStatusUC         *proposal.UpdateProposalStatusUseCase
	getProposalUC          *proposal.GetProposalUseCase
	listProposalsUC        *proposal.ListProposalsUseCase
	listMyProposalsUC      *proposal.ListMyProposalsUseCase
	getMyProposalForOrderUC *proposal.GetMyProposalForOrderUseCase
}

func NewProposalHandler(
	createProposalUC *proposal.CreateProposalUseCase,
	updateStatusUC *proposal.UpdateProposalStatusUseCase,
	getProposalUC *proposal.GetProposalUseCase,
	listProposalsUC *proposal.ListProposalsUseCase,
	listMyProposalsUC *proposal.ListMyProposalsUseCase,
	getMyProposalForOrderUC *proposal.GetMyProposalForOrderUseCase,
) *ProposalHandler {
	return &ProposalHandler{
		createProposalUC:       createProposalUC,
		updateStatusUC:         updateStatusUC,
		getProposalUC:          getProposalUC,
		listProposalsUC:        listProposalsUC,
		listMyProposalsUC:      listMyProposalsUC,
		getMyProposalForOrderUC: getMyProposalForOrderUC,
	}
}

func (h *ProposalHandler) CreateProposal(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "некорректный ID заказа")
		return
	}

	var req dto.CreateProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "некорректные данные запроса")
		return
	}

	deadline, err := dto.ParseDeadline(req.ProposedDeadline)
	if err != nil {
		response.BadRequest(c, "некорректный формат дедлайна")
		return
	}

	created, err := h.createProposalUC.Execute(c.Request.Context(), proposal.CreateProposalInput{
		OrderID:          orderID,
		FreelancerID:     userID,
		CoverLetter:      req.CoverLetter,
		ProposedBudget:   req.ProposedBudget,
		ProposedDeadline: deadline,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToProposalResponse(created))
}

func (h *ProposalHandler) UpdateProposalStatus(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	proposalID, err := uuid.Parse(c.Param("proposalId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID предложения")
		return
	}

	var req dto.UpdateProposalStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "некорректные данные запроса")
		return
	}

	updated, err := h.updateStatusUC.Execute(c.Request.Context(), proposalID, userID, req.Status)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToProposalResponse(updated))
}

func (h *ProposalHandler) GetProposal(c *gin.Context) {
	proposalID, err := uuid.Parse(c.Param("proposalId"))
	if err != nil {
		response.BadRequest(c, "некорректный ID предложения")
		return
	}

	p, err := h.getProposalUC.Execute(c.Request.Context(), proposalID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToProposalResponse(p))
}

func (h *ProposalHandler) ListProposals(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "некорректный ID заказа")
		return
	}

	proposals, err := h.listProposalsUC.Execute(c.Request.Context(), orderID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToProposalResponses(proposals))
}

func (h *ProposalHandler) ListMyProposals(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	proposals, err := h.listMyProposalsUC.Execute(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToProposalResponses(proposals))
}

func (h *ProposalHandler) GetMyProposalForOrder(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "некорректный ID заказа")
		return
	}

	p, err := h.getMyProposalForOrderUC.Execute(c.Request.Context(), orderID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	if p == nil {
		response.NotFound(c, "предложение не найдено")
		return
	}

	response.Success(c, dto.ToProposalResponse(p))
}
