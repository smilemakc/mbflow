package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
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
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, m.maxBodySize)
		c.Next()
	}
}
