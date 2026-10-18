package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
)

type BodySizeMiddleware struct {
	logger      *logger.Logger
	maxBodySize int64
}

func NewBodySizeMiddleware(log *logger.Logger, maxBodySize int64) *BodySizeMiddleware {
	return &BodySizeMiddleware{
		logger:      log,
		maxBodySize: maxBodySize,
	}
}

func (m *BodySizeMiddleware) LimitBodySize() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Reject declared oversized bodies up front; chunked bodies are cut
		// off by MaxBytesReader and reported by bindJSON.
		if c.Request.ContentLength > m.maxBodySize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, ErrPayloadTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, m.maxBodySize)
		c.Next()
	}
}
