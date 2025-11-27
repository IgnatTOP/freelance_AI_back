package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UUIDValidator проверяет, что параметр с указанным именем является валидным UUID.
// Использование: router.GET("/orders/:id", UUIDValidator("id"), handler.GetOrder)
func UUIDValidator(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param(paramName)
		if idStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "параметр " + paramName + " обязателен",
			})
			c.Abort()
			return
		}

		_, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "параметр " + paramName + " должен быть валидным UUID",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
