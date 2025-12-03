package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/interface/http/dto"
	"github.com/ignatzorin/freelance-backend/internal/interface/http/response"
	"github.com/ignatzorin/freelance-backend/internal/usecase/order"
)

type OrderHandler struct {
	createOrderUC   *order.CreateOrderUseCase
	updateOrderUC   *order.UpdateOrderUseCase
	getOrderUC      *order.GetOrderUseCase
	listOrdersUC    *order.ListOrdersUseCase
	deleteOrderUC   *order.DeleteOrderUseCase
	publishOrderUC  *order.PublishOrderUseCase
	cancelOrderUC   *order.CancelOrderUseCase
	completeOrderUC *order.CompleteOrderUseCase
	listMyOrdersUC  *order.ListMyOrdersUseCase
}

func NewOrderHandler(
	createOrderUC *order.CreateOrderUseCase,
	updateOrderUC *order.UpdateOrderUseCase,
	getOrderUC *order.GetOrderUseCase,
	listOrdersUC *order.ListOrdersUseCase,
	deleteOrderUC *order.DeleteOrderUseCase,
) *OrderHandler {
	return &OrderHandler{
		createOrderUC: createOrderUC,
		updateOrderUC: updateOrderUC,
		getOrderUC:    getOrderUC,
		listOrdersUC:  listOrdersUC,
		deleteOrderUC: deleteOrderUC,
	}
}

func NewOrderHandlerFull(
	createOrderUC *order.CreateOrderUseCase,
	updateOrderUC *order.UpdateOrderUseCase,
	getOrderUC *order.GetOrderUseCase,
	listOrdersUC *order.ListOrdersUseCase,
	deleteOrderUC *order.DeleteOrderUseCase,
	publishOrderUC *order.PublishOrderUseCase,
	cancelOrderUC *order.CancelOrderUseCase,
	completeOrderUC *order.CompleteOrderUseCase,
	listMyOrdersUC *order.ListMyOrdersUseCase,
) *OrderHandler {
	return &OrderHandler{
		createOrderUC:   createOrderUC,
		updateOrderUC:   updateOrderUC,
		getOrderUC:      getOrderUC,
		listOrdersUC:    listOrdersUC,
		deleteOrderUC:   deleteOrderUC,
		publishOrderUC:  publishOrderUC,
		cancelOrderUC:   cancelOrderUC,
		completeOrderUC: completeOrderUC,
		listMyOrdersUC:  listMyOrdersUC,
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "некорректные данные запроса")
		return
	}

	deadline, err := dto.ParseDeadline(req.DeadlineAt)
	if err != nil {
		response.BadRequest(c, "некорректный формат дедлайна")
		return
	}

	attachmentIDs, err := dto.ParseUUIDs(req.AttachmentIDs)
	if err != nil {
		response.BadRequest(c, "некорректный формат ID вложений")
		return
	}

	requirements := make([]order.RequirementInput, 0, len(req.Requirements))
	for _, r := range req.Requirements {
		requirements = append(requirements, order.RequirementInput{
			Skill: r.Skill,
			Level: r.Level,
		})
	}

	createdOrder, err := h.createOrderUC.Execute(c.Request.Context(), order.CreateOrderInput{
		ClientID:      userID,
		Title:         req.Title,
		Description:   req.Description,
		BudgetMin:     req.BudgetMin,
		BudgetMax:     req.BudgetMax,
		DeadlineAt:    deadline,
		Requirements:  requirements,
		AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToOrderResponse(createdOrder))
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "некорректный ID заказа")
		return
	}

	o, err := h.getOrderUC.Execute(c.Request.Context(), orderID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToOrderResponse(o))
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	filter := repository.OrderFilter{
		Status:    c.Query("status"),
		Search:    c.Query("search"),
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
		Limit:     parseIntQuery(c, "limit", 20),
		Offset:    parseIntQuery(c, "offset", 0),
	}

	if budgetMin := parseFloatQuery(c, "budget_min"); budgetMin != nil {
		filter.BudgetMin = budgetMin
	}
	if budgetMax := parseFloatQuery(c, "budget_max"); budgetMax != nil {
		filter.BudgetMax = budgetMax
	}

	orders, total, err := h.listOrdersUC.Execute(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginated(c, dto.ToOrderResponses(orders), total, filter.Limit, filter.Offset)
}

func (h *OrderHandler) UpdateOrder(c *gin.Context) {
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

	var req dto.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "некорректные данные запроса")
		return
	}

	deadline, err := dto.ParseDeadline(req.DeadlineAt)
	if err != nil {
		response.BadRequest(c, "некорректный формат дедлайна")
		return
	}

	attachmentIDs, err := dto.ParseUUIDs(req.AttachmentIDs)
	if err != nil {
		response.BadRequest(c, "некорректный формат ID вложений")
		return
	}

	requirements := make([]order.RequirementInput, 0, len(req.Requirements))
	for _, r := range req.Requirements {
		requirements = append(requirements, order.RequirementInput{
			Skill: r.Skill,
			Level: r.Level,
		})
	}

	updatedOrder, err := h.updateOrderUC.Execute(c.Request.Context(), order.UpdateOrderInput{
		OrderID:       orderID,
		ClientID:      userID,
		Title:         req.Title,
		Description:   req.Description,
		BudgetMin:     req.BudgetMin,
		BudgetMax:     req.BudgetMax,
		DeadlineAt:    deadline,
		Requirements:  requirements,
		AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToOrderResponse(updatedOrder))
}

func (h *OrderHandler) DeleteOrder(c *gin.Context) {
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

	if err := h.deleteOrderUC.Execute(c.Request.Context(), orderID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "заказ успешно удалён"})
}

func (h *OrderHandler) PublishOrder(c *gin.Context) {
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

	if h.publishOrderUC == nil {
		response.BadRequest(c, "функция недоступна")
		return
	}

	o, err := h.publishOrderUC.Execute(c.Request.Context(), orderID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToOrderResponse(o))
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
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

	if h.cancelOrderUC == nil {
		response.BadRequest(c, "функция недоступна")
		return
	}

	o, err := h.cancelOrderUC.Execute(c.Request.Context(), orderID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToOrderResponse(o))
}

func (h *OrderHandler) CompleteOrder(c *gin.Context) {
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

	if h.completeOrderUC == nil {
		response.BadRequest(c, "функция недоступна")
		return
	}

	o, err := h.completeOrderUC.Execute(c.Request.Context(), orderID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToOrderResponse(o))
}

func (h *OrderHandler) ListMyOrders(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "требуется авторизация")
		return
	}

	if h.listMyOrdersUC == nil {
		response.BadRequest(c, "функция недоступна")
		return
	}

	orders, err := h.listMyOrdersUC.Execute(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToOrderResponses(orders))
}
