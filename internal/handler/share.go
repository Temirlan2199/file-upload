package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Temirlan2199/file-upload/internal/dto"
	"github.com/Temirlan2199/file-upload/internal/service"
)

// shareFile — POST /api/v1/files/{id}/share — создать публичную ссылку на файл.
// Тело: {"ttl_minutes": 60}
func (h *Handler) shareFile(w http.ResponseWriter, r *http.Request) {
	id, ok := fileIDFromPath(w, r)
	if !ok {
		return
	}

	var req dto.ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ttl := time.Duration(req.TTLMinutes) * time.Minute
	token, expiresAt, err := h.files.Share(r.Context(), userIDFromContext(r.Context()), id, ttl)
	if errors.Is(err, service.ErrFileNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}

	// Сервис вернул только токен — полный адрес собираем здесь,
	// потому что URL — это забота HTTP-слоя, а не бизнес-логики.
	writeJSON(w, http.StatusCreated, dto.ShareResponse{
		URL:       h.publicURL + "/s/" + token,
		ExpiresAt: expiresAt,
	})
}

// downloadShared — GET /s/{token} — скачать файл по публичной ссылке.
// Авторизации нет: доступ даёт сам токен в ссылке.
func (h *Handler) downloadShared(w http.ResponseWriter, r *http.Request) {
	f, body, err := h.files.DownloadShared(r.Context(), r.PathValue("token"))
	// Ссылка истекла или файл уже удалён — для клиента это одно и то же: 404.
	if errors.Is(err, service.ErrShareNotFound) || errors.Is(err, service.ErrFileNotFound) {
		writeError(w, http.StatusNotFound, "link not found or expired")
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.writeFile(w, f, body)
}
