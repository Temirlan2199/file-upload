package handler

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// requestLogger логирует каждый запрос после его обработки.
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next() // передаём управление следующим middleware и хендлеру

		log.Info("http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}
