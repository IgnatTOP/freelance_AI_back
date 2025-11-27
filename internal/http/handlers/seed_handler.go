package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ignatzorin/freelance-backend/internal/service"
)

// SeedHandler обрабатывает запросы для генерации фейковых данных.
type SeedHandler struct {
	seedService *service.SeedService
}

// NewSeedHandler создаёт новый seed handler.
func NewSeedHandler(seedService *service.SeedService) *SeedHandler {
	return &SeedHandler{
		seedService: seedService,
	}
}

// SeedRequest представляет запрос на генерацию данных.
type SeedRequest struct {
	NumUsers  int `json:"num_users" form:"num_users"`
	NumOrders int `json:"num_orders" form:"num_orders"`
}

// SeedResponse представляет ответ на запрос генерации данных.
type SeedResponse struct {
	Message   string `json:"message"`
	NumUsers  int    `json:"num_users"`
	NumOrders int    `json:"num_orders"`
}

// Seed генерирует фейковые данные.
// POST /api/seed
func (h *SeedHandler) Seed(c *gin.Context) {
	var req SeedRequest

	// Парсим параметры из query или body
	if c.Request.Method == "GET" {
		numUsersStr := c.DefaultQuery("num_users", "50")
		numOrdersStr := c.DefaultQuery("num_orders", "100")

		var err error
		req.NumUsers, err = strconv.Atoi(numUsersStr)
		if err != nil || req.NumUsers < 1 {
			req.NumUsers = 50
		}

		req.NumOrders, err = strconv.Atoi(numOrdersStr)
		if err != nil || req.NumOrders < 1 {
			req.NumOrders = 100
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Устанавливаем значения по умолчанию
		if req.NumUsers < 1 {
			req.NumUsers = 50
		}
		if req.NumOrders < 1 {
			req.NumOrders = 100
		}
	}

	// Ограничиваем максимальные значения для безопасности
	if req.NumUsers > 1000 {
		req.NumUsers = 1000
	}
	if req.NumOrders > 5000 {
		req.NumOrders = 5000
	}

	// Генерируем данные
	if err := h.seedService.SeedData(c.Request.Context(), req.NumUsers, req.NumOrders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate seed data",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SeedResponse{
		Message:   "Seed data generated successfully",
		NumUsers:  req.NumUsers,
		NumOrders: req.NumOrders,
	})
}


