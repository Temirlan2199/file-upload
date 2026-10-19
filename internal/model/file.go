package model

import (
	"time"

	"github.com/google/uuid"
)

// File — метаданные файла. Сами байты лежат в MinIO под ключом ID.
type File struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Name        string
	Size        int64 // размер в байтах
	ContentType string
	CreatedAt   time.Time
}
