package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ignatzorin/freelance-backend/internal/service"
)

// SeedHandler обрабатывает запросы для генерации фейковых данных.
type SeedHandler struct {
	seedService         *service.SeedService
	extendedSeedService *service.ExtendedSeedService
}

// NewSeedHandler создаёт новый seed handler.
func NewSeedHandler(seedService *service.SeedService, extendedSeedService *service.ExtendedSeedService) *SeedHandler {
	return &SeedHandler{
		seedService:         seedService,
		extendedSeedService: extendedSeedService,
	}
}

// SeedRequest представляет запрос на генерацию данных.
type SeedRequest struct {
	NumUsers  int `json:"num_users" form:"num_users"`
	NumOrders int `json:"num_orders" form:"num_orders"`
}

// SeedAccountInfo представляет информацию об аккаунте.
type SeedAccountInfo struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// SeedResponse представляет ответ на запрос генерации данных.
type SeedResponse struct {
	Message   string            `json:"message"`
	NumUsers  int               `json:"num_users"`
	NumOrders int               `json:"num_orders"`
	Accounts  []SeedAccountInfo `json:"accounts"`
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

		if req.NumUsers < 1 {
			req.NumUsers = 50
		}
		if req.NumOrders < 1 {
			req.NumOrders = 100
		}
	}

	if req.NumUsers > 1000 {
		req.NumUsers = 1000
	}
	if req.NumOrders > 5000 {
		req.NumOrders = 5000
	}

	result, err := h.seedService.SeedData(c.Request.Context(), req.NumUsers, req.NumOrders)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate seed data",
			"details": err.Error(),
		})
		return
	}

	accounts := make([]SeedAccountInfo, len(result.Accounts))
	for i, acc := range result.Accounts {
		accounts[i] = SeedAccountInfo{
			Email:    acc.Email,
			Username: acc.Username,
			Password: acc.Password,
			Role:     acc.Role,
		}
	}

	c.JSON(http.StatusOK, SeedResponse{
		Message:   "Seed data generated successfully",
		NumUsers:  req.NumUsers,
		NumOrders: req.NumOrders,
		Accounts:  accounts,
	})
}

// SeedRealistic генерирует реалистичные данные как от реальных пользователей.
// POST /api/seed/realistic
func (h *SeedHandler) SeedRealistic(c *gin.Context) {
	if h.extendedSeedService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Extended seed service not available"})
		return
	}

	result, err := h.extendedSeedService.SeedRealisticData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate realistic seed data",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Realistic seed data generated successfully",
		"accounts":          result.Accounts,
		"orders_created":    result.OrdersCreated,
		"proposals_created": result.ProposalsCreated,
		"reviews_created":   result.ReviewsCreated,
	})
}
