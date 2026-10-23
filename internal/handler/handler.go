// Package handler — HTTP-слой: маршруты, хендлеры, middleware.
// Хендлер разбирает запрос, вызывает сервис и пишет ответ. Бизнес-логики здесь нет.
package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

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

// FileService — методы сервиса файлов, которые нужны хендлерам.
type FileService interface {
	Upload(ctx context.Context, ownerID uuid.UUID, name, contentType string, size int64, r io.Reader) (*model.File, error)
	Get(ctx context.Context, ownerID, id uuid.UUID) (*model.File, error)
	List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.File, error)
	Download(ctx context.Context, ownerID, id uuid.UUID) (*model.File, io.ReadCloser, error)
	Delete(ctx context.Context, ownerID, id uuid.UUID) error
	Share(ctx context.Context, ownerID, id uuid.UUID, ttl time.Duration) (string, time.Time, error)
	DownloadShared(ctx context.Context, token string) (*model.File, io.ReadCloser, error)
}

// RateLimiter — ограничитель частоты запросов.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type Handler struct {
	log       *slog.Logger
	auth      AuthService
	files     FileService
	limiter   RateLimiter
	publicURL string // внешний адрес сервиса для ссылок «поделиться»
}

func New(log *slog.Logger, auth AuthService, files FileService, limiter RateLimiter, publicURL string) *Handler {
	return &Handler{log: log, auth: auth, files: files, limiter: limiter, publicURL: publicURL}
}

// Routes регистрирует все маршруты.
// С Go 1.22 в шаблоне можно указать метод: "POST /path".
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	// rateLimit — не больше N попыток в минуту с одного IP.
	mux.HandleFunc("POST /api/v1/auth/register", h.rateLimit(h.register))
	mux.HandleFunc("POST /api/v1/auth/login", h.rateLimit(h.login))

	// requireAuth пропустит запрос дальше, только если токен валиден.
	mux.HandleFunc("GET /api/v1/users/me", h.requireAuth(h.me))

	// Файлы. {id} — параметр пути, достаётся через r.PathValue("id").
	mux.HandleFunc("POST /api/v1/files", h.requireAuth(h.uploadFile))
	mux.HandleFunc("GET /api/v1/files", h.requireAuth(h.listFiles))
	mux.HandleFunc("GET /api/v1/files/{id}", h.requireAuth(h.getFile))
	mux.HandleFunc("GET /api/v1/files/{id}/download", h.requireAuth(h.downloadFile))
	mux.HandleFunc("DELETE /api/v1/files/{id}", h.requireAuth(h.deleteFile))

	// Ссылки «поделиться»: создать — с токеном, скачать — без (доступ даёт сама ссылка).
	mux.HandleFunc("POST /api/v1/files/{id}/share", h.requireAuth(h.shareFile))
	mux.HandleFunc("GET /s/{token}", h.downloadShared)

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
