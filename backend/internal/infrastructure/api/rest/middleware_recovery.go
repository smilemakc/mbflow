package rest

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
)

type RecoveryMiddleware struct {
	logger *logger.Logger
}

func NewRecoveryMiddleware(log *logger.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: log,
	}
}

func (m *RecoveryMiddleware) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()

				requestID := GetRequestID(c)
				userID, _ := GetUserID(c)

				m.logger.Error("panic recovered",
					"request_id", requestID,
					"user_id", userID,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"error", err,
					"stack", string(stack),
				)

				apiErr := NewAPIError(
					"INTERNAL_ERROR",
					fmt.Sprintf("Internal server error (request_id: %s)", requestID),
					http.StatusInternalServerError,
				)

				c.AbortWithStatusJSON(apiErr.HTTPStatus, apiErr)
			}
		}()

		c.Next()
	}
}
