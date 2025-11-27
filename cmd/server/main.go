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
	"github.com/ignatzorin/freelance-backend/internal/logger"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/service"
	"github.com/ignatzorin/freelance-backend/internal/storage"
	"github.com/ignatzorin/freelance-backend/internal/ws"
)

func main() {
	// Готовим контекст для graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("main: ошибка загрузки конфигурации: %v", err)
	}

	// Инициализация логгера
	logLevel := "info"
	if cfg.Env == "development" {
		logLevel = "debug"
		logger.Init(logLevel)
		logger.SetTextFormatter()
	} else {
		logger.Init(logLevel)
	}

	// Подключение к базе и миграции.
	dbConn, err := db.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("main: ошибка подключения к базе: %v", err)
	}
	defer safeClose(dbConn)

	if err := db.RunMigrations(ctx, dbConn, cfg.MigrationsPath); err != nil {
		log.Fatalf("main: ошибка миграций: %v", err)
	}

	// Инициализируем вспомогательные сервисы.
	tokenManager := service.NewTokenManager(cfg.JWTSecret, cfg.RefreshSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	photoStorage, err := storage.NewPhotoStorage(cfg.MediaStoragePath, cfg.MaxUploadSizeMB)
	if err != nil {
		log.Fatalf("main: не удалось подготовить файловое хранилище: %v", err)
	}

	// Репозитории.
	userRepo := repository.NewUserRepository(dbConn)
	orderRepo := repository.NewOrderRepository(dbConn)
	mediaRepo := repository.NewMediaRepository(dbConn)
	notificationRepo := repository.NewNotificationRepository(dbConn)
	portfolioRepo := repository.NewPortfolioRepository(dbConn)

	// Сервисы.
	authService := service.NewAuthService(userRepo, tokenManager)
	notificationService := service.NewNotificationService(notificationRepo)
	portfolioService := service.NewPortfolioService(portfolioRepo)
	seedService := service.NewSeedService(userRepo, orderRepo)
	
	var orderService *service.OrderService
	if cfg.AIBaseURL != "" && cfg.AIModel != "" {
		orderService = service.NewOrderService(orderRepo, userRepo, portfolioRepo, userRepo, ai.NewClient(cfg.AIBaseURL, cfg.AIModel))
	} else {
		orderService = service.NewOrderService(orderRepo, userRepo, portfolioRepo, userRepo, nil)
	}

	// Вебсокеты.
	hub := ws.NewHub(ctx)
	hub.SetNotificationSaver(ws.NewNotificationServiceAdapter(notificationService))
	go hub.Run()
	
	// Устанавливаем hub для отправки уведомлений о готовности AI анализа
	orderService.SetHub(hub)

	// HTTP хэндлеры.
	authHandler := httpHandlers.NewAuthHandler(authService)
	profileHandler := httpHandlers.NewProfileHandler(userRepo, hub)
	orderHandler := httpHandlers.NewOrderHandler(orderService, userRepo, mediaRepo, hub)
	mediaHandler := httpHandlers.NewMediaHandler(mediaRepo, photoStorage)
	wsHandler := httpHandlers.NewWSHandler(hub, tokenManager)
	statsHandler := httpHandlers.NewStatsHandler(orderRepo, userRepo)
	proposalHandler := httpHandlers.NewProposalHandler(orderRepo)
	notificationHandler := httpHandlers.NewNotificationHandler(notificationService)
	portfolioHandler := httpHandlers.NewPortfolioHandler(portfolioService)
	healthHandler := httpHandlers.NewHealthHandler(dbConn)
	seedHandler := httpHandlers.NewSeedHandler(seedService)

	// Роутер.
	engine := httpRouter.SetupRouter(cfg, authHandler, profileHandler, orderHandler, mediaHandler, wsHandler, statsHandler, proposalHandler, notificationHandler, portfolioHandler, healthHandler, seedHandler, tokenManager)

	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: engine,
	}

	// Завершаем сервер при получении сигнала.
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

// safeClose закрывает соединение с базой.
func safeClose(db *sqlx.DB) {
	if err := db.Close(); err != nil {
		log.Printf("main: ошибка закрытия базы: %v", err)
	}
}
