package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user_id не найден в контексте")
	}
	
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("некорректный формат user_id")
	}
	
	return userID, nil
}

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

func parseFloatQuery(c *gin.Context, key string) *float64 {
	valueStr := c.Query(key)
	if valueStr == "" {
		return nil
	}
	
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil
	}
	
	return &value
}
