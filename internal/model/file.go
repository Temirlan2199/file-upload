package model

import (
	"time"

	"github.com/google/uuid"
)

// FileStatus — статус обработки файла воркером.
type FileStatus string

const (
	FileStatusPending FileStatus = "pending" // загружен, ждёт обработки
	FileStatusReady   FileStatus = "ready"   // воркер обработал
	FileStatusFailed  FileStatus = "failed"  // обработка не удалась
)

// File — метаданные файла. Сами байты лежат в MinIO под ключом ID.
type File struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Name        string
	Size        int64 // размер в байтах
	ContentType string
	Status      FileStatus
	Checksum    string // SHA-256 содержимого (hex), считает воркер
	CreatedAt   time.Time
}
