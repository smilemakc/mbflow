package rest

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
)

const (
	RequestIDHeader     = "X-Request-ID"
	ContextKeyRequestID = "request_id"
)

type LoggingMiddleware struct {
	logger *logger.Logger
}

func NewLoggingMiddleware(log *logger.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: log,
	}
}

func (m *LoggingMiddleware) RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(ContextKeyRequestID, requestID)
		c.Header(RequestIDHeader, requestID)

		userID, _ := GetUserID(c)
		if userID == "" {
			userID = "anonymous"
		}

		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		m.logger.Info("request started",
			"request_id", requestID,
			"method", method,
			"path", path,
			"query", query,
			"client_ip", clientIP,
			"user_agent", userAgent,
			"user_id", userID,
		)

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()
		responseSize := c.Writer.Size()

		level := "info"
		if statusCode >= 500 {
			level = "error"
		} else if statusCode >= 400 {
			level = "warn"
		}

		logArgs := []any{
			"request_id", requestID,
			"method", method,
			"path", path,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
			"response_size", responseSize,
			"client_ip", clientIP,
			"user_id", userID,
		}

		if len(c.Errors) > 0 {
			logArgs = append(logArgs, "errors", c.Errors.String())
		}

		switch level {
		case "error":
			m.logger.Error("request completed", logArgs...)
		case "warn":
			m.logger.Warn("request completed", logArgs...)
		default:
			m.logger.Info("request completed", logArgs...)
		}
	}
}

func GetRequestID(c *gin.Context) string {
	requestID, exists := c.Get(ContextKeyRequestID)
	if !exists {
		return ""
	}
	return requestID.(string)
}
