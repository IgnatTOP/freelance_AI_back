package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ignatzorin/freelance-backend/internal/config"
	"github.com/ignatzorin/freelance-backend/internal/http/handlers"
	"github.com/ignatzorin/freelance-backend/internal/http/middleware"
	newHandler "github.com/ignatzorin/freelance-backend/internal/interface/http/handler"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

func SetupRouter(
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	profileHandler *handlers.ProfileHandler,
	orderHandler *handlers.OrderHandler,
	conversationHandler *handlers.ConversationHandler,
	proposalOperationsHandler *handlers.ProposalOperationsHandler,
	aiOrderHandler *handlers.AIOrderHandler,
	mediaHandler *handlers.MediaHandler,
	wsHandler *handlers.WSHandler,
	statsHandler *handlers.StatsHandler,
	dashboardHandler *handlers.DashboardHandler,
	proposalHandler *handlers.ProposalHandler,
	notificationHandler *handlers.NotificationHandler,
	portfolioHandler *handlers.PortfolioHandler,
	healthHandler *handlers.HealthHandler,
	seedHandler *handlers.SeedHandler,
	tokenManager *service.TokenManager,
	// Новые handlers (Clean Architecture)
	newOrderHandler *newHandler.OrderHandler,
	newProposalHandler *newHandler.ProposalHandler,
	newConvHandler *newHandler.ConversationHandler,
	// Платежи и отзывы
	paymentHandler *handlers.PaymentHandler,
	reviewHandler *handlers.ReviewHandler,
	// Каталог
	catalogHandler *handlers.CatalogHandler,
	// Дополнительные фичи
	withdrawalHandler *handlers.WithdrawalHandler,
	favoriteHandler *handlers.FavoriteHandler,
	reportHandler *handlers.ReportHandler,
	disputeHandler *handlers.DisputeHandler,
	verificationHandler *handlers.VerificationHandler,
	proposalTemplateHandler *handlers.ProposalTemplateHandler,
	freelancerHandler *handlers.FreelancerHandler,
) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CORSMiddleware(cfg.AllowedOrigins))

	r.GET("/health", healthHandler.Health)
	r.StaticFS("/media", http.Dir(cfg.MediaStoragePath))

	api := r.Group("/api")

	if seedHandler != nil && cfg.Env == "development" {
		api.POST("/seed", seedHandler.Seed)
		api.GET("/seed", seedHandler.Seed)
		api.POST("/seed/realistic", seedHandler.SeedRealistic)
		api.GET("/seed/realistic", seedHandler.SeedRealistic)
	}

	authGroup := api.Group("/auth")
	authRateLimit := middleware.RateLimitMiddleware(5, cfg.RateLimitPeriod)
	authGroup.Use(authRateLimit)
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
	}

	protectedAuth := api.Group("/auth")
	protectedAuth.Use(middleware.AuthMiddleware(tokenManager))
	{
		protectedAuth.GET("/sessions", authHandler.ListSessions)
		protectedAuth.DELETE("/sessions/:id", authHandler.DeleteSession)
		protectedAuth.DELETE("/sessions", authHandler.DeleteAllSessionsExcept)
	}

	// Публичные маршруты
	api.GET("/orders", orderHandler.ListOrders)
	api.GET("/orders/:id", middleware.UUIDValidator("id"), orderHandler.GetOrder)
	api.GET("/ws", wsHandler.Handle)
	api.GET("/users/:id", middleware.UUIDValidator("id"), profileHandler.GetUserProfile)
	api.GET("/users/:id/portfolio", middleware.UUIDValidator("id"), portfolioHandler.GetUserPortfolio)
	if reviewHandler != nil {
		api.GET("/users/:id/reviews", middleware.UUIDValidator("id"), reviewHandler.ListUserReviews)
	}

	// Каталог (публичный)
	if catalogHandler != nil {
		api.GET("/catalog/categories", catalogHandler.ListCategories)
		api.GET("/catalog/categories/:slug", catalogHandler.GetCategory)
		api.GET("/catalog/skills", catalogHandler.ListSkills)
	}

	// Поиск фрилансеров (публичный)
	if freelancerHandler != nil {
		api.GET("/freelancers/search", freelancerHandler.SearchFreelancers)
	}

	// Защищённые маршруты
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(tokenManager))
	{
		protected.GET("/profile", profileHandler.GetMe)
		protected.PUT("/profile", profileHandler.UpdateMe)
		protected.PUT("/users/me/role", profileHandler.UpdateRole)

		protected.GET("/stats", statsHandler.GetMyStats)
		protected.GET("/dashboard/data", dashboardHandler.GetDashboardData)
		protected.POST("/dashboard/cache/invalidate", dashboardHandler.InvalidateCache)
		protected.GET("/proposals/my", proposalHandler.ListMyProposals)

		protected.GET("/notifications", notificationHandler.ListNotifications)
		protected.GET("/notifications/unread/count", notificationHandler.CountUnread)
		protected.GET("/notifications/:id", notificationHandler.GetNotification)
		protected.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
		protected.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
		protected.DELETE("/notifications/:id", notificationHandler.DeleteNotification)

		protected.POST("/orders", orderHandler.CreateOrder)
		protected.GET("/orders/my", orderHandler.ListMyOrders)
		protected.GET("/orders/:id/my-proposal", middleware.UUIDValidator("id"), proposalOperationsHandler.GetMyProposal)
		protected.GET("/orders/:id/chat", middleware.UUIDValidator("id"), conversationHandler.GetOrderChat)
		protected.POST("/orders/:id/complete-by-freelancer", middleware.UUIDValidator("id"), proposalOperationsHandler.MarkOrderAsCompletedByFreelancer)
		protected.GET("/orders/:id/conversations/:participantId", middleware.UUIDValidator("id"), middleware.UUIDValidator("participantId"), conversationHandler.GetConversation)
		protected.PUT("/orders/:id", middleware.UUIDValidator("id"), orderHandler.UpdateOrder)
		protected.DELETE("/orders/:id", middleware.UUIDValidator("id"), orderHandler.DeleteOrder)
		protected.POST("/orders/:id/proposals", middleware.UUIDValidator("id"), proposalOperationsHandler.CreateProposal)
		protected.GET("/orders/:id/proposals", middleware.UUIDValidator("id"), proposalOperationsHandler.ListProposals)
		protected.PUT("/orders/:id/proposals/:proposalId/status", middleware.UUIDValidator("id"), middleware.UUIDValidator("proposalId"), proposalOperationsHandler.UpdateProposalStatus)
		protected.GET("/conversations/my", conversationHandler.ListMyConversations)
		protected.GET("/conversations/:conversationId/messages", middleware.UUIDValidator("conversationId"), conversationHandler.ListMessages)
		protected.POST("/conversations/:conversationId/messages", middleware.UUIDValidator("conversationId"), conversationHandler.SendMessage)
		protected.PUT("/conversations/:conversationId/messages/:messageId", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), conversationHandler.UpdateMessage)
		protected.DELETE("/conversations/:conversationId/messages/:messageId", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), conversationHandler.DeleteMessage)
		protected.POST("/conversations/:conversationId/messages/:messageId/reactions", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), conversationHandler.AddMessageReaction)
		protected.DELETE("/conversations/:conversationId/messages/:messageId/reactions", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), conversationHandler.RemoveMessageReaction)

		// AI endpoints
		protected.POST("/ai/orders/description", aiOrderHandler.GenerateOrderDescription)
		protected.POST("/ai/orders/description/stream", aiOrderHandler.StreamGenerateOrderDescription)
		protected.POST("/ai/orders/suggestions", aiOrderHandler.GenerateOrderSuggestions)
		protected.POST("/ai/orders/suggestions/stream", aiOrderHandler.StreamGenerateOrderSuggestions)
		protected.POST("/ai/orders/skills", aiOrderHandler.GenerateOrderSkills)
		protected.POST("/ai/orders/skills/stream", aiOrderHandler.StreamGenerateOrderSkills)
		protected.POST("/ai/orders/budget", aiOrderHandler.GenerateOrderBudget)
		protected.POST("/ai/orders/budget/stream", aiOrderHandler.StreamGenerateOrderBudget)
		protected.POST("/ai/welcome-message", aiOrderHandler.GenerateWelcomeMessage)
		protected.POST("/ai/welcome-message/stream", aiOrderHandler.StreamGenerateWelcomeMessage)
		protected.POST("/ai/orders/:id/proposal", middleware.UUIDValidator("id"), aiOrderHandler.GenerateProposal)
		protected.POST("/ai/orders/:id/proposal/stream", middleware.UUIDValidator("id"), aiOrderHandler.StreamGenerateProposal)
		protected.GET("/ai/orders/:id/proposals/feedback", middleware.UUIDValidator("id"), aiOrderHandler.GetProposalFeedback)
		protected.GET("/ai/orders/:id/proposals/feedback/stream", middleware.UUIDValidator("id"), aiOrderHandler.StreamProposalFeedback)
		protected.POST("/ai/orders/improve", aiOrderHandler.ImproveOrderDescription)
		protected.POST("/ai/orders/improve/stream", aiOrderHandler.StreamImproveOrderDescription)
		protected.POST("/ai/orders/:id/regenerate-summary", middleware.UUIDValidator("id"), aiOrderHandler.RegenerateOrderSummary)
		protected.POST("/ai/orders/:id/regenerate-summary/stream", middleware.UUIDValidator("id"), aiOrderHandler.StreamRegenerateOrderSummary)
		protected.GET("/ai/conversations/:conversationId/summary", middleware.UUIDValidator("conversationId"), conversationHandler.SummarizeConversation)
		protected.GET("/ai/conversations/:conversationId/summary/stream", middleware.UUIDValidator("conversationId"), conversationHandler.StreamSummarizeConversation)
		protected.GET("/ai/orders/recommended", aiOrderHandler.RecommendRelevantOrders)
		protected.GET("/ai/orders/recommended/stream", aiOrderHandler.StreamRecommendRelevantOrders)
		protected.GET("/ai/orders/:id/price-timeline", middleware.UUIDValidator("id"), aiOrderHandler.RecommendPriceAndTimeline)
		protected.GET("/ai/orders/:id/price-timeline/stream", middleware.UUIDValidator("id"), aiOrderHandler.StreamRecommendPriceAndTimeline)
		protected.GET("/ai/orders/:id/quality", middleware.UUIDValidator("id"), aiOrderHandler.EvaluateOrderQuality)
		protected.GET("/ai/orders/:id/quality/stream", middleware.UUIDValidator("id"), aiOrderHandler.StreamEvaluateOrderQuality)
		protected.GET("/ai/orders/:id/suitable-freelancers", middleware.UUIDValidator("id"), aiOrderHandler.FindSuitableFreelancers)
		protected.GET("/ai/orders/:id/suitable-freelancers/stream", middleware.UUIDValidator("id"), aiOrderHandler.StreamFindSuitableFreelancers)
		protected.POST("/ai/assistant", aiOrderHandler.AIChatAssistant)
		protected.POST("/ai/assistant/stream", aiOrderHandler.StreamAIChatAssistant)
		protected.POST("/ai/profile/improve", aiOrderHandler.ImproveProfile)
		protected.POST("/ai/profile/improve/stream", aiOrderHandler.StreamImproveProfile)
		protected.POST("/ai/portfolio/improve", aiOrderHandler.ImprovePortfolioItem)
		protected.POST("/ai/portfolio/improve/stream", aiOrderHandler.StreamImprovePortfolioItem)

		protected.GET("/portfolio", portfolioHandler.ListPortfolioItems)
		protected.POST("/portfolio", portfolioHandler.CreatePortfolioItem)
		protected.GET("/portfolio/:id", middleware.UUIDValidator("id"), portfolioHandler.GetPortfolioItem)
		protected.PUT("/portfolio/:id", middleware.UUIDValidator("id"), portfolioHandler.UpdatePortfolioItem)
		protected.DELETE("/portfolio/:id", middleware.UUIDValidator("id"), portfolioHandler.DeletePortfolioItem)

		protected.POST("/media/photos", mediaHandler.UploadPhoto)
		protected.DELETE("/media/:id", middleware.UUIDValidator("id"), mediaHandler.DeleteMedia)

		// Платежи и escrow
		if paymentHandler != nil {
			protected.GET("/payments/balance", paymentHandler.GetBalance)
			protected.POST("/payments/deposit", paymentHandler.Deposit)
			protected.POST("/payments/escrow", paymentHandler.CreateEscrow)
			protected.GET("/payments/escrow/:orderId", middleware.UUIDValidator("orderId"), paymentHandler.GetEscrow)
			protected.GET("/payments/transactions", paymentHandler.ListTransactions)
		}

		// Вывод средств
		if withdrawalHandler != nil {
			protected.POST("/withdrawals", withdrawalHandler.CreateWithdrawal)
			protected.GET("/withdrawals", withdrawalHandler.ListWithdrawals)
		}

		// Отзывы
		if reviewHandler != nil {
			protected.POST("/orders/:id/reviews", middleware.UUIDValidator("id"), reviewHandler.CreateReview)
			protected.GET("/orders/:id/reviews", middleware.UUIDValidator("id"), reviewHandler.ListOrderReviews)
			protected.GET("/orders/:id/can-review", middleware.UUIDValidator("id"), reviewHandler.CanLeaveReview)
		}

		// Споры
		if disputeHandler != nil {
			protected.POST("/orders/:id/dispute", middleware.UUIDValidator("id"), disputeHandler.CreateDispute)
			protected.GET("/orders/:id/dispute", middleware.UUIDValidator("id"), disputeHandler.GetDispute)
			protected.GET("/disputes", disputeHandler.ListMyDisputes)
		}

		// Избранное
		if favoriteHandler != nil {
			protected.POST("/favorites", favoriteHandler.AddFavorite)
			protected.GET("/favorites", favoriteHandler.ListFavorites)
			protected.GET("/favorites/:type/:id", favoriteHandler.CheckFavorite)
			protected.DELETE("/favorites/:type/:id", favoriteHandler.RemoveFavorite)
		}

		// Жалобы
		if reportHandler != nil {
			protected.POST("/reports", reportHandler.CreateReport)
			protected.GET("/reports", reportHandler.ListMyReports)
		}

		// Верификация
		if verificationHandler != nil {
			protected.POST("/verification/email/send", verificationHandler.SendEmailCode)
			protected.POST("/verification/phone/send", verificationHandler.SendPhoneCode)
			protected.POST("/verification/verify", verificationHandler.VerifyCode)
			protected.GET("/verification/status", verificationHandler.GetStatus)
		}

		// Шаблоны откликов
		if proposalTemplateHandler != nil {
			protected.POST("/proposal-templates", proposalTemplateHandler.CreateTemplate)
			protected.GET("/proposal-templates", proposalTemplateHandler.ListTemplates)
			protected.PUT("/proposal-templates/:id", middleware.UUIDValidator("id"), proposalTemplateHandler.UpdateTemplate)
			protected.DELETE("/proposal-templates/:id", middleware.UUIDValidator("id"), proposalTemplateHandler.DeleteTemplate)
		}
	}

	// === НОВЫЕ ENDPOINTS (Clean Architecture) ===
	v2 := api.Group("/v2")
	v2.Use(middleware.AuthMiddleware(tokenManager))
	{
		// Orders
		v2.POST("/orders", newOrderHandler.CreateOrder)
		v2.GET("/orders", newOrderHandler.ListOrders)
		v2.GET("/orders/:id", middleware.UUIDValidator("id"), newOrderHandler.GetOrder)
		v2.PUT("/orders/:id", middleware.UUIDValidator("id"), newOrderHandler.UpdateOrder)
		v2.DELETE("/orders/:id", middleware.UUIDValidator("id"), newOrderHandler.DeleteOrder)

		// Proposals
		v2.POST("/orders/:id/proposals", middleware.UUIDValidator("id"), newProposalHandler.CreateProposal)
		v2.GET("/orders/:id/proposals", middleware.UUIDValidator("id"), newProposalHandler.ListProposals)
		v2.GET("/orders/:id/my-proposal", middleware.UUIDValidator("id"), newProposalHandler.GetMyProposalForOrder)
		v2.GET("/proposals/:proposalId", middleware.UUIDValidator("proposalId"), newProposalHandler.GetProposal)
		v2.PUT("/proposals/:proposalId/status", middleware.UUIDValidator("proposalId"), newProposalHandler.UpdateProposalStatus)
		v2.GET("/proposals/my", newProposalHandler.ListMyProposals)

		// Conversations
		v2.GET("/orders/:id/conversations/:participantId", middleware.UUIDValidator("id"), middleware.UUIDValidator("participantId"), newConvHandler.GetConversation)
		v2.GET("/conversations/my", newConvHandler.ListMyConversations)
		v2.GET("/conversations/:conversationId/messages", middleware.UUIDValidator("conversationId"), newConvHandler.ListMessages)
		v2.POST("/conversations/:conversationId/messages", middleware.UUIDValidator("conversationId"), newConvHandler.SendMessage)
		v2.PUT("/conversations/:conversationId/messages/:messageId", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), newConvHandler.UpdateMessage)
		v2.DELETE("/conversations/:conversationId/messages/:messageId", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), newConvHandler.DeleteMessage)
		v2.POST("/conversations/:conversationId/messages/:messageId/reactions", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), newConvHandler.AddReaction)
		v2.DELETE("/conversations/:conversationId/messages/:messageId/reactions", middleware.UUIDValidator("conversationId"), middleware.UUIDValidator("messageId"), newConvHandler.RemoveReaction)
	}

	return r
}
