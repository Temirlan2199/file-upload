package dto

import (
	"time"

	"github.com/Temirlan2199/file-upload/internal/model"
)

type FileResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	Status      string    `json:"status"`             // pending | ready | failed
	Checksum    string    `json:"checksum,omitempty"` // omitempty — не выводить, пока пустой
	CreatedAt   time.Time `json:"created_at"`
}

func NewFileResponse(f *model.File) FileResponse {
	return FileResponse{
		ID:          f.ID.String(),
		Name:        f.Name,
		Size:        f.Size,
		ContentType: f.ContentType,
		Status:      string(f.Status),
		Checksum:    f.Checksum,
		CreatedAt:   f.CreatedAt,
	}
}

// NewFileListResponse переводит срез моделей в срез ответов.
func NewFileListResponse(files []model.File) []FileResponse {
	resp := make([]FileResponse, 0, len(files))
	for i := range files {
		resp = append(resp, NewFileResponse(&files[i]))
	}
	return resp
}
