// Package middleware provides HTTP middleware functions.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger returns a middleware that logs HTTP requests.
func Logger(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build log fields
		fields := []interface{}{
			"status", c.Writer.Status(),
			"method", c.Request.Method,
			"path", path,
			"latency", latency,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if raw != "" {
			fields = append(fields, "query", raw)
		}

		if c.Writer.Size() > 0 {
			fields = append(fields, "size", c.Writer.Size())
		}

		// Log errors if any
		if len(c.Errors) > 0 {
			fields = append(fields, "errors", c.Errors.String())
		}

		// Log based on status code
		status := c.Writer.Status()
		if status >= 500 {
			logger.Errorw("HTTP request", fields...)
		} else if status >= 400 {
			logger.Warnw("HTTP request", fields...)
		} else {
			logger.Infow("HTTP request", fields...)
		}
	}
}
