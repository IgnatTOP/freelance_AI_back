package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/ai"
	"github.com/ignatzorin/freelance-backend/internal/config"
	"github.com/ignatzorin/freelance-backend/internal/db"
	httpHandlers "github.com/ignatzorin/freelance-backend/internal/http/handlers"
	httpRouter "github.com/ignatzorin/freelance-backend/internal/http/router"
	"github.com/ignatzorin/freelance-backend/internal/infrastructure/persistence"
	newHandler "github.com/ignatzorin/freelance-backend/internal/interface/http/handler"
	"github.com/ignatzorin/freelance-backend/internal/logger"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/service"
	"github.com/ignatzorin/freelance-backend/internal/storage"
	convUC "github.com/ignatzorin/freelance-backend/internal/usecase/conversation"
	orderUC "github.com/ignatzorin/freelance-backend/internal/usecase/order"
	proposalUC "github.com/ignatzorin/freelance-backend/internal/usecase/proposal"
	"github.com/ignatzorin/freelance-backend/internal/ws"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("main: ошибка загрузки конфигурации: %v", err)
	}

	logLevel := "info"
	if cfg.Env == "development" {
		logLevel = "debug"
		logger.Init(logLevel)
		logger.SetTextFormatter()
	} else {
		logger.Init(logLevel)
	}

	dbConn, err := db.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("main: ошибка подключения к базе: %v", err)
	}
	defer safeClose(dbConn)

	if err := db.RunMigrations(ctx, dbConn, cfg.MigrationsPath); err != nil {
		log.Fatalf("main: ошибка миграций: %v", err)
	}

	tokenManager := service.NewTokenManager(cfg.JWTSecret, cfg.RefreshSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	cacheService := service.NewCacheService()

	photoStorage, err := storage.NewPhotoStorage(cfg.MediaStoragePath, cfg.MaxUploadSizeMB)
	if err != nil {
		log.Fatalf("main: не удалось подготовить файловое хранилище: %v", err)
	}

	// === СТАРЫЕ РЕПОЗИТОРИИ (для совместимости) ===
	userRepo := repository.NewUserRepository(dbConn)
	orderRepo := repository.NewOrderRepository(dbConn)
	mediaRepo := repository.NewMediaRepository(dbConn)
	notificationRepo := repository.NewNotificationRepository(dbConn)
	portfolioRepo := repository.NewPortfolioRepository(dbConn)
	paymentRepo := repository.NewPaymentRepository(dbConn)
	reviewRepo := repository.NewReviewRepository(dbConn)
	catalogRepo := repository.NewCatalogRepository(dbConn)
	// Новые репозитории
	withdrawalRepo := repository.NewWithdrawalRepository(dbConn)
	favoriteRepo := repository.NewFavoriteRepository(dbConn)
	reportRepo := repository.NewReportRepository(dbConn)
	disputeRepo := repository.NewDisputeRepository(dbConn)
	verificationRepo := repository.NewVerificationRepository(dbConn)
	proposalTemplateRepo := repository.NewProposalTemplateRepository(dbConn)

	// === НОВЫЕ РЕПОЗИТОРИИ (Clean Architecture) ===
	newOrderRepo := persistence.NewOrderRepositoryAdapter(dbConn)
	newProposalRepo := persistence.NewProposalRepositoryAdapter(dbConn)
	newConvRepo := persistence.NewConversationRepositoryAdapter(dbConn)
	newMsgRepo := persistence.NewMessageRepositoryAdapter(dbConn)

	// === USE CASES ===
	// Order
	createOrderUC := orderUC.NewCreateOrderUseCase(newOrderRepo)
	updateOrderUC := orderUC.NewUpdateOrderUseCase(newOrderRepo)
	getOrderUC := orderUC.NewGetOrderUseCase(newOrderRepo)
	listOrdersUC := orderUC.NewListOrdersUseCase(newOrderRepo)
	deleteOrderUC := orderUC.NewDeleteOrderUseCase(newOrderRepo)

	// Proposal
	createProposalUC := proposalUC.NewCreateProposalUseCase(newProposalRepo, newOrderRepo)
	updateProposalStatusUC := proposalUC.NewUpdateProposalStatusUseCase(newProposalRepo, newOrderRepo)
	getProposalUC := proposalUC.NewGetProposalUseCase(newProposalRepo)
	listProposalsUC := proposalUC.NewListProposalsUseCase(newProposalRepo)
	listMyProposalsUC := proposalUC.NewListMyProposalsUseCase(newProposalRepo)
	getMyProposalForOrderUC := proposalUC.NewGetMyProposalForOrderUseCase(newProposalRepo)

	// Conversation
	getOrCreateConvUC := convUC.NewGetOrCreateConversationUseCase(newConvRepo, newOrderRepo)
	listMyConvsUC := convUC.NewListMyConversationsUseCase(newConvRepo)
	sendMessageUC := convUC.NewSendMessageUseCase(newConvRepo, newMsgRepo)
	listMessagesUC := convUC.NewListMessagesUseCase(newConvRepo, newMsgRepo)
	updateMessageUC := convUC.NewUpdateMessageUseCase(newMsgRepo)
	deleteMessageUC := convUC.NewDeleteMessageUseCase(newMsgRepo)
	addReactionUC := convUC.NewAddReactionUseCase(newMsgRepo)
	removeReactionUC := convUC.NewRemoveReactionUseCase(newMsgRepo)

	// === НОВЫЕ HANDLERS ===
	newOrderHandler := newHandler.NewOrderHandler(createOrderUC, updateOrderUC, getOrderUC, listOrdersUC, deleteOrderUC)
	newProposalHandler := newHandler.NewProposalHandler(createProposalUC, updateProposalStatusUC, getProposalUC, listProposalsUC, listMyProposalsUC, getMyProposalForOrderUC)
	newConvHandler := newHandler.NewConversationHandler(getOrCreateConvUC, listMyConvsUC, sendMessageUC, listMessagesUC, updateMessageUC, deleteMessageUC, addReactionUC, removeReactionUC)

	// === СТАРЫЕ СЕРВИСЫ (для совместимости) ===
	authService := service.NewAuthService(userRepo, tokenManager)
	notificationService := service.NewNotificationService(notificationRepo)
	portfolioService := service.NewPortfolioService(portfolioRepo)
	seedService := service.NewSeedService(userRepo, orderRepo)
	extendedSeedService := service.NewExtendedSeedService(userRepo, orderRepo, paymentRepo, reviewRepo, favoriteRepo, proposalTemplateRepo)
	paymentService := service.NewPaymentService(paymentRepo)
	reviewService := service.NewReviewService(reviewRepo, orderRepo)
	// Новые сервисы
	withdrawalService := service.NewWithdrawalService(withdrawalRepo)
	favoriteService := service.NewFavoriteService(favoriteRepo)
	reportService := service.NewReportService(reportRepo)
	disputeService := service.NewDisputeService(disputeRepo, paymentRepo)
	verificationService := service.NewVerificationService(verificationRepo)
	proposalTemplateService := service.NewProposalTemplateService(proposalTemplateRepo)

	var orderService *service.OrderService
	if cfg.AIBaseURL != "" && cfg.AIModel != "" {
		orderService = service.NewOrderService(orderRepo, userRepo, portfolioRepo, userRepo, ai.NewClient(cfg.AIBaseURL, cfg.AIModel))
	} else {
		orderService = service.NewOrderService(orderRepo, userRepo, portfolioRepo, userRepo, nil)
	}
	orderService.SetPaymentRepository(paymentRepo)

	hub := ws.NewHub(ctx)
	hub.SetNotificationSaver(ws.NewNotificationServiceAdapter(notificationService))
	go hub.Run()

	orderService.SetHub(hub)

	// === СТАРЫЕ HANDLERS (для совместимости) ===
	authHandler := httpHandlers.NewAuthHandler(authService)
	profileHandler := httpHandlers.NewProfileHandler(userRepo, hub)
	orderHandler := httpHandlers.NewOrderHandler(orderService, userRepo, mediaRepo, hub, cacheService)
	conversationHandler := httpHandlers.NewConversationHandler(orderService, userRepo, mediaRepo, hub)
	proposalOperationsHandler := httpHandlers.NewProposalOperationsHandler(orderService, userRepo, mediaRepo, hub)
	aiOrderHandler := httpHandlers.NewAIOrderHandler(orderService, userRepo, mediaRepo, hub)
	mediaHandler := httpHandlers.NewMediaHandler(mediaRepo, photoStorage)
	wsHandler := httpHandlers.NewWSHandler(hub, tokenManager)
	statsHandler := httpHandlers.NewStatsHandler(orderRepo, userRepo)
	dashboardHandler := httpHandlers.NewDashboardHandler(orderRepo, userRepo, notificationRepo, orderService, cacheService)
	proposalHandler := httpHandlers.NewProposalHandler(orderRepo)
	notificationHandler := httpHandlers.NewNotificationHandler(notificationService)
	portfolioHandler := httpHandlers.NewPortfolioHandler(portfolioService)
	healthHandler := httpHandlers.NewHealthHandler(dbConn)
	seedHandler := httpHandlers.NewSeedHandler(seedService, extendedSeedService)
	paymentHandler := httpHandlers.NewPaymentHandler(paymentService)
	reviewHandler := httpHandlers.NewReviewHandler(reviewService)
	catalogHandler := httpHandlers.NewCatalogHandler(catalogRepo)
	// Новые handlers
	withdrawalHandler := httpHandlers.NewWithdrawalHandler(withdrawalService)
	favoriteHandler := httpHandlers.NewFavoriteHandler(favoriteService)
	reportHandler := httpHandlers.NewReportHandler(reportService)
	disputeHandler := httpHandlers.NewDisputeHandler(disputeService)
	verificationHandler := httpHandlers.NewVerificationHandler(verificationService)
	proposalTemplateHandler := httpHandlers.NewProposalTemplateHandler(proposalTemplateService)
	freelancerHandler := httpHandlers.NewFreelancerHandler(userRepo)

	// Роутер с новыми и старыми handlers
	engine := httpRouter.SetupRouter(
		cfg,
		authHandler,
		profileHandler,
		orderHandler,
		conversationHandler,
		proposalOperationsHandler,
		aiOrderHandler,
		mediaHandler,
		wsHandler,
		statsHandler,
		dashboardHandler,
		proposalHandler,
		notificationHandler,
		portfolioHandler,
		healthHandler,
		seedHandler,
		tokenManager,
		// Новые handlers
		newOrderHandler,
		newProposalHandler,
		newConvHandler,
		// Платежи и отзывы
		paymentHandler,
		reviewHandler,
		// Каталог
		catalogHandler,
		// Дополнительные фичи
		withdrawalHandler,
		favoriteHandler,
		reportHandler,
		disputeHandler,
		verificationHandler,
		proposalTemplateHandler,
		freelancerHandler,
	)

	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: engine,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("main: ошибка остановки http сервера: %v", err)
		}
	}()

	log.Printf("main: HTTP сервер запущен на порту %s", cfg.HTTPPort)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("main: сервер завершился с ошибкой: %v", err)
	}
}

func safeClose(db *sqlx.DB) {
	if err := db.Close(); err != nil {
		log.Printf("main: ошибка закрытия базы: %v", err)
	}
}
