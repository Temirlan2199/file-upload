// Package handler — HTTP-слой: роутинг, хендлеры, middleware.
// Хендлер только разбирает запрос (DTO), вызывает сервис и формирует ответ.
// Бизнес-логики здесь быть не должно.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter собирает gin.Engine со всеми маршрутами и middleware.
func NewRouter(log *slog.Logger) *gin.Engine {
	// gin.New() вместо gin.Default(): middleware подключаем явно,
	// чтобы логи шли через наш slog, а не через встроенный логгер gin.
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(log))

	r.GET("/health", health)

	return r
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
