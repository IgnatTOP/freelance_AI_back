package common

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/dto"
	"github.com/ignatzorin/freelance-backend/internal/http/middleware"
)

var (
	// ErrUserNotFound is returned when user is not found in context
	ErrUserNotFound = errors.New("пользователь не найден в контексте")

	// ErrInvalidUUID is returned when UUID parsing fails
	ErrInvalidUUID = errors.New("неверный формат UUID")
)

// CurrentUserID extracts user ID from Gin context
// Consolidates the duplicate currentUserID functions across handlers
func CurrentUserID(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		return uuid.Nil, ErrUserNotFound
	}

	userID, ok := raw.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrUserNotFound
	}

	return userID, nil
}

// CurrentUserRole extracts user role from Gin context
// Consolidates the duplicate currentUserRole functions across handlers
func CurrentUserRole(c *gin.Context) (string, error) {
	raw, exists := c.Get(middleware.ContextRoleKey)
	if !exists {
		return "", ErrUserNotFound
	}

	role, ok := raw.(string)
	if !ok {
		return "", ErrUserNotFound
	}

	return role, nil
}

// ParseUUIDParam parses UUID from URL parameter
// Consolidates UUID parsing logic across handlers
func ParseUUIDParam(c *gin.Context, paramName string) (uuid.UUID, error) {
	param := c.Param(paramName)
	if param == "" {
		return uuid.Nil, fmt.Errorf("параметр %s отсутствует", paramName)
	}

	parsed, err := uuid.Parse(param)
	if err != nil {
		return uuid.Nil, ErrInvalidUUID
	}

	return parsed, nil
}

// BindAndValidate binds JSON request and returns properly formatted error
func BindAndValidate(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		return fmt.Errorf("ошибка валидации запроса: %w", err)
	}
	return nil
}

// RespondError sends a standardized error response
func RespondError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, dto.ErrorResponse{Error: message})
}

// RespondSuccess sends a standardized success response
func RespondSuccess(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, dto.SuccessResponse{
		Message: message,
		Data:    data,
	})
}

// RespondJSON sends a JSON response with the given status code and data
func RespondJSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// RespondUnauthorized sends a 401 Unauthorized response
func RespondUnauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "требуется авторизация"
	}
	RespondError(c, http.StatusUnauthorized, message)
}

// RespondForbidden sends a 403 Forbidden response
func RespondForbidden(c *gin.Context, message string) {
	if message == "" {
		message = "доступ запрещён"
	}
	RespondError(c, http.StatusForbidden, message)
}

// RespondNotFound sends a 404 Not Found response
func RespondNotFound(c *gin.Context, message string) {
	if message == "" {
		message = "ресурс не найден"
	}
	RespondError(c, http.StatusNotFound, message)
}

// RespondBadRequest sends a 400 Bad Request response
func RespondBadRequest(c *gin.Context, message string) {
	if message == "" {
		message = "некорректный запрос"
	}
	RespondError(c, http.StatusBadRequest, message)
}

// RespondInternalError sends a 500 Internal Server Error response
func RespondInternalError(c *gin.Context, message string) {
	if message == "" {
		message = "внутренняя ошибка сервера"
	}
	RespondError(c, http.StatusInternalServerError, message)
}

// Contains checks if a string contains a substring (case-sensitive)
// Helper function used in handlers for error message checking
func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ParseIntQuery safely reads an integer query parameter with a fallback value
func ParseIntQuery(c *gin.Context, key string, fallback int) int {
	if v := c.Query(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

// GetPagination extracts limit and offset from query parameters with defaults
func GetPagination(c *gin.Context) (limit, offset int) {
	limit = ParseIntQuery(c, "limit", 20)
	offset = ParseIntQuery(c, "offset", 0)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return
}
