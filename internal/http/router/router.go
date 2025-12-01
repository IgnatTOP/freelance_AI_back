package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ignatzorin/freelance-backend/internal/config"
	"github.com/ignatzorin/freelance-backend/internal/http/handlers"
	"github.com/ignatzorin/freelance-backend/internal/http/middleware"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

// SetupRouter формирует корневой роутер Gin.
func SetupRouter(
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	profileHandler *handlers.ProfileHandler,
	orderHandler *handlers.OrderHandler,
	mediaHandler *handlers.MediaHandler,
	wsHandler *handlers.WSHandler,
	statsHandler *handlers.StatsHandler,
	proposalHandler *handlers.ProposalHandler,
	notificationHandler *handlers.NotificationHandler,
	portfolioHandler *handlers.PortfolioHandler,
	healthHandler *handlers.HealthHandler,
	seedHandler *handlers.SeedHandler,
	tokenManager *service.TokenManager,
) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Error handler middleware (должен быть первым)
	r.Use(middleware.ErrorHandler())

	// CORS middleware для всех запросов
	r.Use(middleware.CORSMiddleware(cfg.AllowedOrigins))

	// Health check endpoint (публичный, без префикса /api)
	r.GET("/health", healthHandler.Health)

	// Отдаём статику с фотографиями.
	r.StaticFS("/media", http.Dir(cfg.MediaStoragePath))

	api := r.Group("/api")

	// Seed endpoint для генерации фейковых данных (только для development)
	if seedHandler != nil && cfg.Env == "development" {
		api.POST("/seed", seedHandler.Seed)
		api.GET("/seed", seedHandler.Seed)
	}

	authGroup := api.Group("/auth")
	// Rate limiting для auth endpoints (5 запросов в минуту)
	authRateLimit := middleware.RateLimitMiddleware(5, cfg.RateLimitPeriod)
	authGroup.Use(authRateLimit)
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
	}

	// Управление сессиями (требуют авторизации)
	protectedAuth := api.Group("/auth")
	protectedAuth.Use(middleware.AuthMiddleware(tokenManager))
	{
		protectedAuth.GET("/sessions", authHandler.ListSessions)
		protectedAuth.DELETE("/sessions/:id", authHandler.DeleteSession)
		protectedAuth.DELETE("/sessions", authHandler.DeleteAllSessionsExcept)
	}

	// Публичные маршруты.
	api.GET("/orders", orderHandler.ListOrders)
	api.GET("/orders/:id", middleware.UUIDValidator("id"), orderHandler.GetOrder)
	api.GET("/ws", wsHandler.Handle)
	api.GET("/users/:id", middleware.UUIDValidator("id"), profileHandler.GetUserProfile)
	api.GET("/users/:id/portfolio", middleware.UUIDValidator("id"), portfolioHandler.GetUserPortfolio)

	// Требуют авторизации.
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(tokenManager))
	{
		protected.GET("/profile", profileHandler.GetMe)
		protected.PUT("/profile", profileHandler.UpdateMe)
		protected.PUT("/users/me/role", profileHandler.UpdateRole)

		protected.GET("/stats", statsHandler.GetMyStats)
		protected.GET("/proposals/my", proposalHandler.ListMyProposals)

		protected.GET("/notifications", notificationHandler.ListNotifications)
		protected.GET("/notifications/unread/count", notificationHandler.CountUnread)
		protected.GET("/notifications/:id", notificationHandler.GetNotification)
		protected.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
		protected.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
		protected.DELETE("/notifications/:id", notificationHandler.DeleteNotification)

		protected.POST("/orders", orderHandler.CreateOrder)
		protected.GET("/orders/my", orderHandler.ListMyOrders)
		// Более специфичные роуты должны быть раньше общих
		protected.GET("/orders/:id/my-proposal", middleware.UUIDValidator("id"), orderHandler.GetMyProposal)
		protected.GET("/orders/:id/chat", middleware.UUIDValidator("id"), orderHandler.GetOrderChat)
		protected.POST("/orders/:id/complete-by-freelancer", middleware.UUIDValidator("id"), orderHandler.MarkOrderAsCompletedByFreelancer)
		protected.GET("/orders/:id/conversations/:participantId", middleware.UUIDValidator("id"), middleware.UUIDValidator("participantId"), orderHandler.GetConversation)
		protected.PUT("/orders/:id", middleware.UUIDValidator("id"), orderHandler.UpdateOrder)
		protected.DELETE("/orders/:id", middleware.UUIDValidator("id"), orderHandler.DeleteOrder)
		protected.POST("/orders/:id/proposals", middleware.UUIDValidator("id"), orderHandler.CreateProposal)
		protected.GET("/orders/:id/proposals", middleware.UUIDValidator("id"), orderHandler.ListProposals)
		protected.PUT("/orders/:id/proposals/:proposalId/status", middleware.UUIDValidator("id"), middleware.UUIDValidator("proposalId"), orderHandler.UpdateProposalStatus)
		protected.GET("/conversations/my", orderHandler.ListMyConversations)
		protected.GET("/conversations/:conversationId/messages", middleware.UUIDValidator("conversationId"), orderHandler.ListMessages)
		protected.POST("/conversations/:conversationId/messages", middleware.UUIDValidator("conversationId"), orderHandler.SendMessage)
		protected.PUT("/conversations/:conversationId/messages/:messageId", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), orderHandler.UpdateMessage)
		protected.DELETE("/conversations/:conversationId/messages/:messageId", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), orderHandler.DeleteMessage)

		// AI endpoints
		protected.POST("/ai/orders/description", orderHandler.GenerateOrderDescription)
		protected.POST("/ai/orders/description/stream", orderHandler.StreamGenerateOrderDescription)
		protected.POST("/ai/orders/suggestions", orderHandler.GenerateOrderSuggestions)
		protected.POST("/ai/orders/suggestions/stream", orderHandler.StreamGenerateOrderSuggestions)
		protected.POST("/ai/orders/skills", orderHandler.GenerateOrderSkills)
		protected.POST("/ai/orders/skills/stream", orderHandler.StreamGenerateOrderSkills)
		protected.POST("/ai/orders/budget", orderHandler.GenerateOrderBudget)
		protected.POST("/ai/orders/budget/stream", orderHandler.StreamGenerateOrderBudget)
		protected.POST("/ai/welcome-message", orderHandler.GenerateWelcomeMessage)
		protected.POST("/ai/welcome-message/stream", orderHandler.StreamGenerateWelcomeMessage)
		protected.POST("/ai/orders/:id/proposal", middleware.UUIDValidator("id"), orderHandler.GenerateProposal)
		protected.POST("/ai/orders/:id/proposal/stream", middleware.UUIDValidator("id"), orderHandler.StreamGenerateProposal)
		protected.GET("/ai/orders/:id/proposals/feedback", middleware.UUIDValidator("id"), orderHandler.GetProposalFeedback)
		protected.GET("/ai/orders/:id/proposals/feedback/stream", middleware.UUIDValidator("id"), orderHandler.StreamProposalFeedback)
		protected.POST("/ai/orders/improve", orderHandler.ImproveOrderDescription)
		protected.POST("/ai/orders/improve/stream", orderHandler.StreamImproveOrderDescription)
		protected.POST("/ai/orders/:id/regenerate-summary", middleware.UUIDValidator("id"), orderHandler.RegenerateOrderSummary)
		protected.POST("/ai/orders/:id/regenerate-summary/stream", middleware.UUIDValidator("id"), orderHandler.StreamRegenerateOrderSummary)
		protected.GET("/ai/conversations/:conversationId/summary", middleware.UUIDValidator("conversationId"), orderHandler.SummarizeConversation)
		protected.GET("/ai/conversations/:conversationId/summary/stream", middleware.UUIDValidator("conversationId"), orderHandler.StreamSummarizeConversation)
		protected.GET("/ai/orders/recommended", orderHandler.RecommendRelevantOrders)
		protected.GET("/ai/orders/recommended/stream", orderHandler.StreamRecommendRelevantOrders)
		protected.GET("/ai/orders/:id/price-timeline", middleware.UUIDValidator("id"), orderHandler.RecommendPriceAndTimeline)
		protected.GET("/ai/orders/:id/price-timeline/stream", middleware.UUIDValidator("id"), orderHandler.StreamRecommendPriceAndTimeline)
		protected.GET("/ai/orders/:id/quality", middleware.UUIDValidator("id"), orderHandler.EvaluateOrderQuality)
		protected.GET("/ai/orders/:id/quality/stream", middleware.UUIDValidator("id"), orderHandler.StreamEvaluateOrderQuality)
		protected.GET("/ai/orders/:id/suitable-freelancers", middleware.UUIDValidator("id"), orderHandler.FindSuitableFreelancers)
		protected.GET("/ai/orders/:id/suitable-freelancers/stream", middleware.UUIDValidator("id"), orderHandler.StreamFindSuitableFreelancers)
		protected.POST("/ai/assistant", orderHandler.AIChatAssistant)
		protected.POST("/ai/assistant/stream", orderHandler.StreamAIChatAssistant)
		protected.POST("/ai/profile/improve", orderHandler.ImproveProfile)
		protected.POST("/ai/profile/improve/stream", orderHandler.StreamImproveProfile)
		protected.POST("/ai/portfolio/improve", orderHandler.ImprovePortfolioItem)
		protected.POST("/ai/portfolio/improve/stream", orderHandler.StreamImprovePortfolioItem)

		protected.GET("/portfolio", portfolioHandler.ListPortfolioItems)
		protected.POST("/portfolio", portfolioHandler.CreatePortfolioItem)
		protected.GET("/portfolio/:id", middleware.UUIDValidator("id"), portfolioHandler.GetPortfolioItem)
		protected.PUT("/portfolio/:id", middleware.UUIDValidator("id"), portfolioHandler.UpdatePortfolioItem)
		protected.DELETE("/portfolio/:id", middleware.UUIDValidator("id"), portfolioHandler.DeletePortfolioItem)

		protected.POST("/media/photos", mediaHandler.UploadPhoto)
		protected.DELETE("/media/:id", middleware.UUIDValidator("id"), mediaHandler.DeleteMedia)
	}

	return r
}
