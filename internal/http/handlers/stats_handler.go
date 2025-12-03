package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

// StatsHandler отвечает за статистику пользователя.
type StatsHandler struct {
	orders *repository.OrderRepository
	users  *repository.UserRepository
}

// NewStatsHandler создаёт экземпляр.
func NewStatsHandler(orders *repository.OrderRepository, users *repository.UserRepository) *StatsHandler {
	return &StatsHandler{
		orders: orders,
		users:  users,
	}
}

// GetMyStats возвращает статистику текущего пользователя.
func (h *StatsHandler) GetMyStats(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Получаем статистику по заказам
	orderStats, err := h.orders.GetUserOrderStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось получить статистику заказов"})
		return
	}

	// Получаем статистику по предложениям
	proposalStats, err := h.orders.GetUserProposalStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось получить статистику предложений"})
		return
	}

	// Получаем статистику пользователя (рейтинг, отзывы, completion rate)
	userStats, err := h.users.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		// Если ошибка, используем значения по умолчанию
		userStats = &models.PublicProfileStats{
			TotalOrders:     0,
			CompletedOrders: 0,
			AverageRating:   0.0,
			TotalReviews:    0,
			TotalEarnings:   0.0,
		}
	}

	// Вычисляем процент выполнения (completion rate)
	completionRate := 0.0
	if userStats.TotalOrders > 0 {
		completionRate = float64(userStats.CompletedOrders) / float64(userStats.TotalOrders) * 100
	}

	// Вычисляем среднее время ответа (в часах)
	avgResponseTimeHours, err := h.orders.GetAverageResponseTimeHours(c.Request.Context(), userID)
	if err != nil {
		// Если ошибка, используем значение по умолчанию
		avgResponseTimeHours = 0.0
	}

	// Объединяем статистику
	stats := gin.H{
		"orders": gin.H{
			"total":           orderStats["total"],
			"open":            orderStats["open"],
			"in_progress":     orderStats["in_progress"],
			"completed":       orderStats["completed"],
			"total_proposals": orderStats["total_proposals"],
		},
		"proposals": gin.H{
			"total":    proposalStats["total"],
			"pending":  proposalStats["pending"],
			"accepted": proposalStats["accepted"],
			"rejected": proposalStats["rejected"],
		},
		"balance":          userStats.TotalEarnings,
		"average_rating":   userStats.AverageRating,
		"total_reviews":    userStats.TotalReviews,
		"completion_rate":  completionRate,
		"response_time_hours": avgResponseTimeHours,
	}

	c.JSON(http.StatusOK, stats)
}

