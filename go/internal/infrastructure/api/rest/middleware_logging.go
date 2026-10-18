package rest

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
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
		if !validRequestID(requestID) {
			requestID = uuid.New().String()
		}
		c.Set(ContextKeyRequestID, requestID)
		c.Header(RequestIDHeader, requestID)

		path := c.Request.URL.Path
		query := redactQuery(c.Request.URL.RawQuery)
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
		)

		c.Next()

		// Auth middleware runs after this one, so the user is known only now.
		userID, _ := GetUserID(c)
		if userID == "" {
			userID = "anonymous"
		}

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

// sensitiveQueryParams are masked in request logs (tokens may be passed in the
// query string, e.g. for WebSocket and SSE connections).
var sensitiveQueryParams = map[string]struct{}{
	"token":         {},
	"access_token":  {},
	"refresh_token": {},
	"api_key":       {},
	"apikey":        {},
	"key":           {},
	"secret":        {},
	"password":      {},
	"code":          {},
}

// redactQuery masks values of sensitive query parameters.
func redactQuery(raw string) string {
	if raw == "" {
		return raw
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "[unparseable]"
	}
	redacted := false
	for k := range values {
		if _, ok := sensitiveQueryParams[strings.ToLower(k)]; ok {
			values[k] = []string{"REDACTED"}
			redacted = true
		}
	}
	if !redacted {
		return raw
	}
	return values.Encode()
}

// validRequestID accepts client-supplied request IDs only if they are short
// and contain safe characters, preventing log injection.
func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.', r == ':':
		default:
			return false
		}
	}
	return true
}
