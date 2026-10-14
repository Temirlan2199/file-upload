// Package handler — HTTP-слой: маршруты, хендлеры, middleware.
// Хендлер разбирает запрос, вызывает сервис и пишет ответ. Бизнес-логики здесь нет.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/Temirlan2199/file-upload/internal/dto"
	"github.com/Temirlan2199/file-upload/internal/model"
)

// AuthService — методы сервиса, которые нужны хендлерам.
type AuthService interface {
	Register(ctx context.Context, email, password string) (*model.User, error)
	Login(ctx context.Context, email, password string) (string, error)
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

// Routes регистрирует все маршруты.
// С Go 1.22 в шаблоне можно указать метод: "POST /path".
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)

	// requireAuth пропустит запрос дальше, только если токен валиден.
	mux.HandleFunc("GET /api/v1/users/me", h.requireAuth(h.me))

	// logRequests оборачивает весь mux — логируется каждый запрос.
	return h.logRequests(mux)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// writeJSON пишет ответ в формате JSON с нужным статусом.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, dto.ErrorResponse{Error: msg})
}

// internalError — для неожиданных ошибок: подробности пишем в лог,
// а клиенту отдаём общий текст, чтобы не раскрывать внутренности.
func (h *Handler) internalError(w http.ResponseWriter, err error) {
	h.log.Error("internal error", "error", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}
