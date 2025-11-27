package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimitMiddleware создаёт middleware для ограничения количества запросов.
// По умолчанию: 10 запросов в минуту с одного IP.
func RateLimitMiddleware(limit int64, period time.Duration) gin.HandlerFunc {
	if limit <= 0 {
		limit = 10
	}
	if period <= 0 {
		period = 1 * time.Minute
	}

	rate := limiter.Rate{
		Period: period,
		Limit:  limit,
	}
	store := memory.NewStore()
	instance := limiter.New(store, rate)

	return func(c *gin.Context) {
		key := c.ClientIP()
		context, err := instance.Get(c, key)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", context.Limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", context.Remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", context.Reset))

		if context.Reached {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "слишком много запросов, попробуйте позже",
			})
			return
		}

		c.Next()
	}
}
