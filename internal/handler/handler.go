// Package handler — HTTP-слой: роутинг, хендлеры, middleware.
// Хендлер только разбирает запрос (DTO), вызывает сервис и формирует ответ.
// Бизнес-логики здесь быть не должно.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Temirlan2199/file-upload/internal/model"
)

// AuthService — то, что хендлерам нужно от сервиса авторизации.
type AuthService interface {
	Register(ctx context.Context, email, password string) (*model.User, error)
	Login(ctx context.Context, email, password string) (token string, expiresAt time.Time, err error)
	ParseToken(token string) (uuid.UUID, error)
	GetUser(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type Handler struct {
	log  *slog.Logger
	auth AuthService
}

func New(log *slog.Logger, auth AuthService) *Handler {
	return &Handler{log: log, auth: auth}
}

// Router собирает gin.Engine со всеми маршрутами и middleware.
func (h *Handler) Router() *gin.Engine {
	registerValidatorJSONNames()

	// gin.New() вместо gin.Default(): middleware подключаем явно,
	// чтобы логи шли через наш slog, а не через встроенный логгер gin.
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(h.log))

	r.GET("/health", health)

	api := r.Group("/api/v1")

	auth := api.Group("/auth")
	auth.POST("/register", h.register)
	auth.POST("/login", h.login)

	// Всё, что в этой группе, требует валидный JWT.
	protected := api.Group("", h.authMiddleware())
	protected.GET("/users/me", h.me)

	return r
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
