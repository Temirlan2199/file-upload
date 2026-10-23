package handler

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/Temirlan2199/file-upload/internal/dto"
	"github.com/Temirlan2199/file-upload/internal/model"
	"github.com/Temirlan2199/file-upload/internal/service"
)

// Максимальный размер загружаемого файла — 100 МБ.
const maxUploadSize = 100 << 20 // 100 * 1024 * 1024

// uploadFile принимает файл в формате multipart/form-data (поле "file").
// Так файлы отправляют HTML-формы, curl -F и Postman.
func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	// Ограничиваем размер тела: если клиент пришлёт больше — чтение вернёт ошибку.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	// FormFile разбирает multipart-тело и находит поле "file".
	// file — поток с содержимым, header — имя, размер и тип файла.
	// (Файлы до 32 МБ Go держит в памяти, больше — во временном файле на диске.)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required (max 100MB)")
		return
	}
	defer file.Close()
	// Файлы больше 32 МБ FormFile сохраняет во временный файл на диске.
	// net/http удаляет такие файлы сам, но только у исходного запроса,
	// а у нас копия: requireAuth вызывает r.WithContext(...).
	// Поэтому удаляем временные файлы сами, иначе они останутся на диске навсегда.
	defer r.MultipartForm.RemoveAll()

	// Тип файла сообщает клиент. Если не сообщил — «просто байты».
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	userID := userIDFromContext(r.Context())
	f, err := h.files.Upload(r.Context(), userID, header.Filename, contentType, header.Size, file)
	if err != nil {
		h.internalError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.NewFileResponse(f))
}

// listFiles — GET /api/v1/files?limit=20&offset=0
func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	// Читаем параметры из query-строки. Если их нет или они кривые — берём значения по умолчанию.
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	files, err := h.files.List(r.Context(), userIDFromContext(r.Context()), limit, offset)
	if err != nil {
		h.internalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.NewFileListResponse(files))
}

// getFile — GET /api/v1/files/{id} — метаданные файла.
func (h *Handler) getFile(w http.ResponseWriter, r *http.Request) {
	id, ok := fileIDFromPath(w, r)
	if !ok {
		return
	}

	f, err := h.files.Get(r.Context(), userIDFromContext(r.Context()), id)
	if errors.Is(err, service.ErrFileNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.NewFileResponse(f))
}

// downloadFile — GET /api/v1/files/{id}/download — отдаёт сами байты файла.
func (h *Handler) downloadFile(w http.ResponseWriter, r *http.Request) {
	id, ok := fileIDFromPath(w, r)
	if !ok {
		return
	}

	f, body, err := h.files.Download(r.Context(), userIDFromContext(r.Context()), id)
	if errors.Is(err, service.ErrFileNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.writeFile(w, f, body)
}

// writeFile отдаёт файл клиенту: заголовки + байты из потока body.
// Используется и для обычного скачивания, и для скачивания по ссылке.
func (h *Handler) writeFile(w http.ResponseWriter, f *model.File, body io.ReadCloser) {
	defer body.Close() // закрываем поток из MinIO после отправки

	// Заголовки нужно выставить ДО записи тела.
	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))
	// attachment — браузер скачает файл под этим именем, а не откроет его.
	// mime.FormatMediaType правильно экранирует имя (кавычки, кириллица),
	// поэтому не склеиваем заголовок руками.
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Name}))

	// io.Copy перекачивает байты из MinIO клиенту кусками по 32 КБ,
	// не загружая весь файл в память.
	if _, err := io.Copy(w, body); err != nil {
		// Статус уже отправлен, поменять ответ нельзя — только залогировать.
		h.log.Error("download: copy body", "error", err)
	}
}

// deleteFile — DELETE /api/v1/files/{id}
func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	id, ok := fileIDFromPath(w, r)
	if !ok {
		return
	}

	err := h.files.Delete(r.Context(), userIDFromContext(r.Context()), id)
	if errors.Is(err, service.ErrFileNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}

	// 204 No Content — успешно, тела ответа нет.
	w.WriteHeader(http.StatusNoContent)
}

// fileIDFromPath достаёт {id} из пути (/api/v1/files/{id}) и проверяет, что это UUID.
// Если нет — сам отвечает 400 и возвращает ok = false.
func fileIDFromPath(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid file id")
		return uuid.Nil, false
	}
	return id, true
}
