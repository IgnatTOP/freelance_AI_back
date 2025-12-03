package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

// DashboardHandler handles aggregated dashboard data requests.
type DashboardHandler struct {
	orders        *repository.OrderRepository
	users         *repository.UserRepository
	notifications *repository.NotificationRepository
	orderService  *service.OrderService
	cache         *service.CacheService
}

// NewDashboardHandler creates a new instance.
func NewDashboardHandler(
	orders *repository.OrderRepository,
	users *repository.UserRepository,
	notifications *repository.NotificationRepository,
	orderService *service.OrderService,
	cache *service.CacheService,
) *DashboardHandler {
	return &DashboardHandler{
		orders:        orders,
		users:         users,
		notifications: notifications,
		orderService:  orderService,
		cache:         cache,
	}
}

// DashboardData represents aggregated dashboard response.
type DashboardData struct {
	Stats             *StatsData             `json:"stats"`
	Activities        []models.Notification  `json:"activities"`
	RecentOrders      []models.Order         `json:"recent_orders"`
	AIRecommendations *AIRecommendationsData `json:"ai_recommendations,omitempty"`
	Insights          *InsightsData          `json:"insights,omitempty"`
}

// StatsData contains user statistics.
type StatsData struct {
	Orders            OrderStats    `json:"orders"`
	Proposals         ProposalStats `json:"proposals"`
	Balance           float64       `json:"balance"`
	AverageRating     float64       `json:"average_rating"`
	TotalReviews      int           `json:"total_reviews"`
	CompletionRate    float64       `json:"completion_rate"`
	ResponseTimeHours float64       `json:"response_time_hours"`
}

// OrderStats contains order statistics.
type OrderStats struct {
	Total          int `json:"total"`
	Open           int `json:"open"`
	InProgress     int `json:"in_progress"`
	Completed      int `json:"completed"`
	TotalProposals int `json:"total_proposals"`
}

// ProposalStats contains proposal statistics.
type ProposalStats struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

// AIRecommendationsData contains AI recommendations.
type AIRecommendationsData struct {
	// For freelancers
	RecommendedOrders []RecommendedOrder `json:"recommended_orders,omitempty"`
	Explanation       string             `json:"explanation,omitempty"`

	// For clients
	SuitableFreelancers []SuitableFreelancer `json:"suitable_freelancers,omitempty"`
}

// RecommendedOrder represents a recommended order with full details.
type RecommendedOrder struct {
	Order       *models.Order `json:"order"`
	MatchScore  float64       `json:"match_score"`
	Explanation string        `json:"explanation"`
}

// SuitableFreelancer represents a suitable freelancer for a client's order.
type SuitableFreelancer struct {
	FreelancerID uuid.UUID `json:"freelancer_id"`
	OrderID      uuid.UUID `json:"order_id"`
	MatchScore   float64   `json:"match_score"`
	Explanation  string    `json:"explanation"`
}

// InsightsData contains data for generating insights.
type InsightsData struct {
	OrdersWithoutProposals []models.Order `json:"orders_without_proposals,omitempty"`
}

// GetDashboardData returns aggregated dashboard data in a single request.
func (h *DashboardHandler) GetDashboardData(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	userRole, err := common.CurrentUserRole(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	includeAI := c.Query("include_ai") == "true"

	// Try to get from cache first
	cacheKey := service.DashboardCacheKey(userID, includeAI)
	if cached, found := h.cache.Get(cacheKey); found {
		if dashboardData, ok := cached.(*DashboardData); ok {
			c.JSON(http.StatusOK, dashboardData)
			return
		}
	}

	// Prepare response
	dashboardData := &DashboardData{}

	// Fetch stats (parallel execution)
	type statsResult struct {
		orderStats    map[string]int
		proposalStats map[string]int
		userStats     *models.PublicProfileStats
		avgRespTime   float64
		err           error
	}

	statsChan := make(chan statsResult, 1)
	go func() {
		result := statsResult{}

		// Get order stats
		result.orderStats, result.err = h.orders.GetUserOrderStats(ctx, userID)
		if result.err != nil {
			statsChan <- result
			return
		}

		// Get proposal stats
		result.proposalStats, result.err = h.orders.GetUserProposalStats(ctx, userID)
		if result.err != nil {
			statsChan <- result
			return
		}

		// Get user stats
		result.userStats, result.err = h.users.GetUserStats(ctx, userID)
		if result.err != nil {
			// Use defaults if error
			result.userStats = &models.PublicProfileStats{
				TotalOrders:     0,
				CompletedOrders: 0,
				AverageRating:   0.0,
				TotalReviews:    0,
				TotalEarnings:   0.0,
			}
			result.err = nil
		}

		// Get average response time
		result.avgRespTime, _ = h.orders.GetAverageResponseTimeHours(ctx, userID)

		statsChan <- result
	}()

	// Fetch recent activities (notifications)
	activityChan := make(chan []models.Notification, 1)
	go func() {
		notifications, err := h.notifications.List(ctx, userID, 10, 0, false)
		if err != nil {
			activityChan <- []models.Notification{}
			return
		}
		activityChan <- notifications
	}()

	// Fetch recent orders
	ordersChan := make(chan []models.Order, 1)
	go func() {
		var orders []models.Order

		if userRole == "client" {
			clientOrders, _, err := h.orders.ListMyOrders(ctx, userID)
			if err != nil {
				ordersChan <- []models.Order{}
				return
			}
			// Take first 5 orders
			if len(clientOrders) > 5 {
				orders = clientOrders[:5]
			} else {
				orders = clientOrders
			}
		} else {
			// For freelancers, get published orders
			result, err := h.orders.List(ctx, repository.ListFilterParams{
				Status:    "published",
				Limit:     5,
				SortBy:    "created_at",
				SortOrder: "desc",
			})
			if err != nil {
				ordersChan <- []models.Order{}
				return
			}
			orders = result.Orders
		}

		ordersChan <- orders
	}()

	// Wait for results
	statsRes := <-statsChan
	if statsRes.err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}

	activities := <-activityChan
	recentOrders := <-ordersChan

	// Build stats data
	completionRate := 0.0
	if statsRes.userStats.TotalOrders > 0 {
		completionRate = float64(statsRes.userStats.CompletedOrders) / float64(statsRes.userStats.TotalOrders) * 100
	}

	dashboardData.Stats = &StatsData{
		Orders: OrderStats{
			Total:          statsRes.orderStats["total"],
			Open:           statsRes.orderStats["open"],
			InProgress:     statsRes.orderStats["in_progress"],
			Completed:      statsRes.orderStats["completed"],
			TotalProposals: statsRes.orderStats["total_proposals"],
		},
		Proposals: ProposalStats{
			Total:    statsRes.proposalStats["total"],
			Pending:  statsRes.proposalStats["pending"],
			Accepted: statsRes.proposalStats["accepted"],
			Rejected: statsRes.proposalStats["rejected"],
		},
		Balance:           statsRes.userStats.TotalEarnings,
		AverageRating:     statsRes.userStats.AverageRating,
		TotalReviews:      statsRes.userStats.TotalReviews,
		CompletionRate:    completionRate,
		ResponseTimeHours: statsRes.avgRespTime,
	}

	dashboardData.Activities = activities
	dashboardData.RecentOrders = recentOrders

	// Fetch AI recommendations (optional, can be disabled for faster response)
	if includeAI {
		if err := h.fetchAIRecommendations(ctx, userID, userRole, recentOrders, dashboardData); err != nil {
			// Log error but don't fail the request
			fmt.Printf("dashboard: failed to fetch AI recommendations: %v\n", err)
		}
	}

	// Build insights data
	dashboardData.Insights = h.buildInsightsData(recentOrders)

	// Cache the result
	// Cache for 1 hour if no AI, 24 hours if with AI (AI data changes less frequently)
	cacheTTL := 1 * time.Hour
	if includeAI {
		cacheTTL = 24 * time.Hour
	}
	h.cache.Set(cacheKey, dashboardData, cacheTTL)

	c.JSON(http.StatusOK, dashboardData)
}

// fetchAIRecommendations fetches AI recommendations based on user role.
func (h *DashboardHandler) fetchAIRecommendations(
	ctx context.Context,
	userID uuid.UUID,
	userRole string,
	recentOrders []models.Order,
	dashboardData *DashboardData,
) error {
	dashboardData.AIRecommendations = &AIRecommendationsData{}

	if userRole == "freelancer" {
		// Get recommended orders for freelancer with caching
		cacheKey := service.AIRecommendationsCacheKey(userID, userRole)

		var recommendedOrders []models.RecommendedOrder
		var explanation string
		var err error

		// Try cache first
		if cached, found := h.cache.Get(cacheKey); found {
			if cachedData, ok := cached.(struct {
				Orders      []models.RecommendedOrder
				Explanation string
			}); ok {
				recommendedOrders = cachedData.Orders
				explanation = cachedData.Explanation
			}
		}

		// If not in cache, fetch from AI
		if recommendedOrders == nil {
			if h.orderService != nil {
				recommendedOrders, explanation, err = h.orderService.RecommendRelevantOrders(ctx, userID, 10)
				if err != nil {
					// Log but don't fail - AI is optional
					fmt.Printf("dashboard: failed to get recommended orders: %v\n", err)
					return nil
				}

				// Cache for 24 hours
				h.cache.Set(cacheKey, struct {
					Orders      []models.RecommendedOrder
					Explanation string
				}{
					Orders:      recommendedOrders,
					Explanation: explanation,
				}, 24*time.Hour)
			} else {
				return nil
			}
		}

		// Get full order details for recommended orders
		recommendedOrdersWithDetails := make([]RecommendedOrder, 0, len(recommendedOrders))
		for _, rec := range recommendedOrders {
			// Get full order details
			order, err := h.orders.GetByID(ctx, rec.OrderID)
			if err != nil {
				// Skip if order not found
				continue
			}

			recommendedOrdersWithDetails = append(recommendedOrdersWithDetails, RecommendedOrder{
				Order:       order,
				MatchScore:  rec.MatchScore,
				Explanation: rec.Explanation,
			})
		}

		dashboardData.AIRecommendations.RecommendedOrders = recommendedOrdersWithDetails
		dashboardData.AIRecommendations.Explanation = explanation

		return nil
	} else if userRole == "client" {
		// Get suitable freelancers for client's orders with caching and optimization
		suitableFreelancers := make([]SuitableFreelancer, 0)

		// Filter published/in_progress orders
		activeOrders := make([]models.Order, 0)
		for _, order := range recentOrders {
			if order.Status == "published" || order.Status == "in_progress" {
				activeOrders = append(activeOrders, order)
			}
		}

		// Limit to 3 orders to avoid too many AI calls (оптимизировано с 5 до 3)
		if len(activeOrders) > 3 {
			activeOrders = activeOrders[:3]
		}

		// Проверяем кэш для всех заказов сразу
		ordersToFetch := make([]models.Order, 0)
		cachedFreelancersMap := make(map[uuid.UUID][]models.SuitableFreelancer)

		for _, order := range activeOrders {
			cacheKey := service.SuitableFreelancersCacheKey(order.ID)
			if cached, found := h.cache.Get(cacheKey); found {
				if cachedFreelancers, ok := cached.([]models.SuitableFreelancer); ok && len(cachedFreelancers) > 0 {
					cachedFreelancersMap[order.ID] = cachedFreelancers
				} else {
					ordersToFetch = append(ordersToFetch, order)
				}
			} else {
				ordersToFetch = append(ordersToFetch, order)
			}
		}

		// Fetch from AI only for orders not in cache (with rate limiting)
		// Ограничиваем до 2 одновременных AI запросов
		if len(ordersToFetch) > 0 && h.orderService != nil {
			maxConcurrent := 2
			if len(ordersToFetch) < maxConcurrent {
				maxConcurrent = len(ordersToFetch)
			}

			// Используем каналы для ограничения параллельных запросов
			semaphore := make(chan struct{}, maxConcurrent)
			resultsChan := make(chan struct {
				orderID     uuid.UUID
				freelancers []models.SuitableFreelancer
				err         error
			}, len(ordersToFetch))

			// Запускаем запросы с ограничением параллелизма
			for _, order := range ordersToFetch {
				go func(o models.Order) {
					semaphore <- struct{}{}        // Занимаем слот
					defer func() { <-semaphore }() // Освобождаем слот

					cacheKey := service.SuitableFreelancersCacheKey(o.ID)
					freelancers, err := h.orderService.FindSuitableFreelancers(ctx, o.ID, userID, userRole, 5)
					if err != nil {
						// Log but don't fail - use fallback
						fmt.Printf("dashboard: failed to get suitable freelancers for order %s: %v\n", o.ID, err)
						resultsChan <- struct {
							orderID     uuid.UUID
							freelancers []models.SuitableFreelancer
							err         error
						}{orderID: o.ID, freelancers: nil, err: err}
						return
					}

					// Cache for 24 hours
					if len(freelancers) > 0 {
						h.cache.Set(cacheKey, freelancers, 24*time.Hour)
					}

					resultsChan <- struct {
						orderID     uuid.UUID
						freelancers []models.SuitableFreelancer
						err         error
					}{orderID: o.ID, freelancers: freelancers, err: nil}
				}(order)
			}

			// Собираем результаты
			for i := 0; i < len(ordersToFetch); i++ {
				result := <-resultsChan
				if result.err == nil && len(result.freelancers) > 0 {
					cachedFreelancersMap[result.orderID] = result.freelancers
				}
			}
		}

		// Convert to dashboard format from all sources (cache + fresh)
		for _, order := range activeOrders {
			freelancers := cachedFreelancersMap[order.ID]
			if len(freelancers) == 0 {
				// Fallback: если нет рекомендаций, пропускаем этот заказ
				continue
			}

			for _, freelancer := range freelancers {
				suitableFreelancers = append(suitableFreelancers, SuitableFreelancer{
					FreelancerID: freelancer.UserID,
					OrderID:      order.ID,
					MatchScore:   freelancer.MatchScore,
					Explanation:  freelancer.Explanation,
				})
			}
		}

		// Если нет рекомендаций, возвращаем пустой массив (не показываем ошибку)
		if len(suitableFreelancers) == 0 {
			dashboardData.AIRecommendations.SuitableFreelancers = []SuitableFreelancer{}
			return nil
		}

		// Sort by match score and take top 10 unique freelancers
		uniqueFreelancers := make(map[uuid.UUID]SuitableFreelancer)
		for _, sf := range suitableFreelancers {
			if existing, exists := uniqueFreelancers[sf.FreelancerID]; !exists || sf.MatchScore > existing.MatchScore {
				uniqueFreelancers[sf.FreelancerID] = sf
			}
		}

		// Convert to slice and sort
		finalFreelancers := make([]SuitableFreelancer, 0, len(uniqueFreelancers))
		for _, sf := range uniqueFreelancers {
			finalFreelancers = append(finalFreelancers, sf)
		}

		// Sort by match score descending
		for i := 0; i < len(finalFreelancers)-1; i++ {
			for j := i + 1; j < len(finalFreelancers); j++ {
				if finalFreelancers[i].MatchScore < finalFreelancers[j].MatchScore {
					finalFreelancers[i], finalFreelancers[j] = finalFreelancers[j], finalFreelancers[i]
				}
			}
		}

		// Limit to top 10
		if len(finalFreelancers) > 10 {
			finalFreelancers = finalFreelancers[:10]
		}

		dashboardData.AIRecommendations.SuitableFreelancers = finalFreelancers
	}

	return nil
}

// buildInsightsData builds insights data from orders.
func (h *DashboardHandler) buildInsightsData(orders []models.Order) *InsightsData {
	insights := &InsightsData{}

	// Find orders without proposals
	for _, order := range orders {
		if order.Status == "published" && order.ProposalsCount != nil && *order.ProposalsCount == 0 {
			insights.OrdersWithoutProposals = append(insights.OrdersWithoutProposals, order)
		}
	}

	return insights
}

// InvalidateCache manually invalidates dashboard cache for the current user.
func (h *DashboardHandler) InvalidateCache(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.cache.InvalidateUserCache(userID)

	c.JSON(http.StatusOK, gin.H{"message": "cache invalidated"})
}
