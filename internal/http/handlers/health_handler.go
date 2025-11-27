package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// HealthHandler предоставляет endpoint для проверки здоровья сервиса.
type HealthHandler struct {
	db *sqlx.DB
}

// NewHealthHandler создаёт новый health handler.
func NewHealthHandler(db *sqlx.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthResponse представляет ответ health check.
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// Health обрабатывает GET /health.
func (h *HealthHandler) Health(c *gin.Context) {
	checks := make(map[string]string)
	status := "healthy"

	// Проверка подключения к БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		status = "unhealthy"
	} else {
		checks["database"] = "healthy"
	}

	// Проверка статистики пула соединений
	stats := h.db.Stats()
	if stats.OpenConnections > stats.MaxOpenConnections {
		checks["connection_pool"] = "warning: too many connections"
	} else {
		checks["connection_pool"] = "healthy"
	}

	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
	})
}

