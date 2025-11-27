package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ignatzorin/freelance-backend/internal/logger"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

// ErrorHandler обрабатывает ошибки централизованно.
// Маскирует внутренние ошибки и возвращает понятные сообщения клиенту.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Проверяем, не был ли уже отправлен ответ
		if c.Writer.Written() {
			return
		}

		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Определяем тип ошибки и статус код
			statusCode := http.StatusInternalServerError
			message := "внутренняя ошибка сервера"

			// Логируем ошибку
			if logger.Log != nil {
				logger.Log.WithFields(logrus.Fields{
					"error":  err.Error(),
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
				}).Error("Request error")
			}

			// Обрабатываем известные типы ошибок
			if errors.Is(err.Err, repository.ErrUserNotFound) {
				statusCode = http.StatusNotFound
				message = "пользователь не найден"
			} else if errors.Is(err.Err, repository.ErrOrderNotFound) {
				statusCode = http.StatusNotFound
				message = "заказ не найден"
			} else if errors.Is(err.Err, repository.ErrConversationNotFound) {
				statusCode = http.StatusNotFound
				message = "диалог не найден"
			} else if err.Error() != "" {
				// Если ошибка содержит понятное сообщение, используем его
				// Но только если это не внутренняя ошибка
				errStr := err.Error()
				if !containsInternalKeywords(errStr) {
					message = errStr
					// Для некоторых ошибок меняем статус код
					if contains(errStr, "неверный") || contains(errStr, "невалид") {
						statusCode = http.StatusBadRequest
					} else if contains(errStr, "нет прав") || contains(errStr, "не авторизован") {
						statusCode = http.StatusForbidden
					}
				}
			}

			c.JSON(statusCode, gin.H{"error": message})
		}
	}
}

// containsInternalKeywords проверяет, содержит ли строка ключевые слова внутренних ошибок.
func containsInternalKeywords(s string) bool {
	keywords := []string{
		"sql:",
		"database",
		"connection",
		"timeout",
		"internal",
		"panic",
		"runtime",
	}

	for _, keyword := range keywords {
		if contains(s, keyword) {
			return true
		}
	}
	return false
}

// contains проверяет, содержит ли строка подстроку (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
