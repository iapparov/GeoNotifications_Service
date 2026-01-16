package middlewares

import (
	"geoNotifications/internal/logger"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func LoggerMiddleware(l *logger.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		l.Log(
			zapcore.InfoLevel,
			"HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", status),
			zap.Duration("duration", duration),
		)
	}
}

func AuthMiddleware(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-API-Key")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Api Key"})
			return
		}

		if token != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Api Key is invalid"})
			return
		}

		c.Next()
	}
}
