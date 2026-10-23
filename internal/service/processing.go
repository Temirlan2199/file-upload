package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/Temirlan2199/file-upload/internal/model"
	"github.com/Temirlan2199/file-upload/internal/repository"
)

// ProcessingRepository — то, что нужно от базы для фоновой обработки.
type ProcessingRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.File, error)
	UpdateProcessing(ctx context.Context, id uuid.UUID, status model.FileStatus, checksum string) error
}

// ProcessingService — фоновая обработка файлов. Его вызывает воркер,
// когда получает событие из Kafka (как HTTP-хендлер вызывает FileService).
type ProcessingService struct {
	files   ProcessingRepository
	storage FileStorage
	cache   FileCache
}

func NewProcessingService(files ProcessingRepository, storage FileStorage, cache FileCache) *ProcessingService {
	return &ProcessingService{files: files, storage: storage, cache: cache}
}

// ProcessUploaded обрабатывает только что загруженный файл:
// считает SHA-256 содержимого и ставит статус ready (или failed).
func (s *ProcessingService) ProcessUploaded(ctx context.Context, id uuid.UUID) error {
	f, err := s.files.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil // файл успели удалить — обрабатывать нечего
	}
	if err != nil {
		return err
	}

	status := model.FileStatusReady
	checksum, checksumErr := s.checksum(ctx, f.ID)
	if checksumErr != nil {
		status = model.FileStatusFailed
	}

	err = s.files.UpdateProcessing(ctx, id, status, checksum)
	if errors.Is(err, repository.ErrNotFound) {
		return nil // удалили, пока считали хэш
	}
	if err != nil {
		return err
	}

	// В кэше могла остаться старая версия со статусом pending — удаляем её.
	_ = s.cache.Delete(ctx, id)

	if checksumErr != nil {
		return fmt.Errorf("checksum: %w", checksumErr) // чтобы воркер записал причину в лог
	}
	return nil
}

// checksum читает файл из MinIO потоком и считает его SHA-256.
func (s *ProcessingService) checksum(ctx context.Context, id uuid.UUID) (string, error) {
	body, err := s.storage.Download(ctx, id.String())
	if err != nil {
		return "", err
	}
	defer body.Close()

	// sha256.New() — «хэшер», который принимает данные кусками через Write.
	// io.Copy читает файл порциями и скармливает их хэшеру — весь файл
	// в память не загружается.
	h := sha256.New()
	if _, err := io.Copy(h, body); err != nil {
		return "", err
	}

	// h.Sum(nil) — 32 байта хэша, hex превращает их в строку из 64 символов.
	return hex.EncodeToString(h.Sum(nil)), nil
}

// DeleteObject удаляет объект файла из MinIO (после удаления записи в базе).
// Если объекта уже нет, MinIO не возвращает ошибку — повторное событие безопасно.
func (s *ProcessingService) DeleteObject(ctx context.Context, id uuid.UUID) error {
	return s.storage.Delete(ctx, id.String())
}
