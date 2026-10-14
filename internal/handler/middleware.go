package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Temirlan2199/file-upload/internal/dto"
)

// Ключ, под которым authMiddleware кладёт id пользователя в gin.Context.
const ctxUserIDKey = "userID"

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

// authMiddleware проверяет заголовок "Authorization: Bearer <token>".
// Если токен валиден — кладёт id пользователя в контекст и пропускает дальше,
// иначе прерывает цепочку с 401 (хендлер даже не вызывается).
func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "missing bearer token"})
			return
		}

		userID, err := h.auth.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "invalid or expired token"})
			return
		}

		c.Set(ctxUserIDKey, userID)
		c.Next()
	}
}

// userIDFromContext достаёт id пользователя, положенный authMiddleware.
// Вызывать только в хендлерах защищённой группы — иначе будет паника,
// и это ошибка программиста, а не пользователя.
func userIDFromContext(c *gin.Context) uuid.UUID {
	return c.MustGet(ctxUserIDKey).(uuid.UUID)
}
