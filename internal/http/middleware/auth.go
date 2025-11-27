package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/service"
)

// Context ключи для gin.Context.
const (
	ContextUserIDKey = "userID"
	ContextRoleKey   = "role"
)

// AuthMiddleware проверяет JWT access токен.
func AuthMiddleware(tokens *service.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "требуется авторизация"})
			return
		}

		raw := strings.TrimPrefix(auth, "Bearer ")
		userID, role, err := tokens.ParseAccess(raw)
		if err != nil || userID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "токен невалиден"})
			return
		}

		c.Set(ContextUserIDKey, userID)
		c.Set(ContextRoleKey, role)
		c.Next()
	}
}
