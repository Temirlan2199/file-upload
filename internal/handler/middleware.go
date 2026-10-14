package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Middleware — функция, которая принимает обработчик и возвращает новый,
// добавляя что-то до и/или после вызова исходного.

// logRequests логирует каждый запрос.
func (h *Handler) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r) // вызываем следующий обработчик
		h.log.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

// Свой тип ключа, чтобы наш ключ в context не совпал с чужим.
type ctxKey string

const userIDKey ctxKey = "userID"

// requireAuth проверяет заголовок "Authorization: Bearer <token>".
// Если токен валиден — кладёт id пользователя в context запроса и вызывает next.
// Если нет — отвечает 401, и next не вызывается.
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

		userID, err := h.auth.ParseToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// userIDFromContext достаёт id пользователя, который положил requireAuth.
func userIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey).(uuid.UUID)
	return id
}
