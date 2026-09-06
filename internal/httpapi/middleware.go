package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (a *API) requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 128 {
			id = uuid.NewString()
		}
		c.Set("requestId", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func (a *API) accessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		a.logger.InfoContext(c.Request.Context(), "http request",
			slog.String("method", c.Request.Method), slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()), slog.Int64("durationMs", time.Since(started).Milliseconds()),
			slog.String("requestId", c.GetString("requestId")),
		)
	}
}

func bodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func cors(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; origin != "" && ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			if origin == "" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			if _, ok := allowed[origin]; !ok && !strings.Contains(origin, "localhost") {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
