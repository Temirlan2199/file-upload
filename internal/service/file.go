package service

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"

	"github.com/Temirlan2199/file-upload/internal/model"
	"github.com/Temirlan2199/file-upload/internal/repository"
)

var ErrFileNotFound = errors.New("file not found")

// FileRepository — метаданные файлов в Postgres.
type FileRepository interface {
	Create(ctx context.Context, f *model.File) error
	GetByID(ctx context.Context, id, ownerID uuid.UUID) (*model.File, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.File, error)
	Delete(ctx context.Context, id, ownerID uuid.UUID) error
}

// FileStorage — сами байты файлов (MinIO).
type FileStorage interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type FileService struct {
	files   FileRepository
	storage FileStorage
}

func NewFileService(files FileRepository, storage FileStorage) *FileService {
	return &FileService{files: files, storage: storage}
}

// Upload сохраняет файл: байты — в MinIO, метаданные — в Postgres.
func (s *FileService) Upload(ctx context.Context, ownerID uuid.UUID, name, contentType string, size int64, r io.Reader) (*model.File, error) {
	f := &model.File{
		ID:          uuid.New(), // генерируем id сами — он же ключ объекта в MinIO
		OwnerID:     ownerID,
		Name:        name,
		Size:        size,
		ContentType: contentType,
	}

	// 1. Сначала загружаем байты в MinIO.
	if err := s.storage.Upload(ctx, f.ID.String(), r, size, contentType); err != nil {
		return nil, err
	}

	// 2. Потом записываем метаданные в базу.
	// Если база упала — удаляем уже загруженный объект, чтобы в MinIO
	// не остался «мусор», о котором никто не знает.
	if err := s.files.Create(ctx, f); err != nil {
		_ = s.storage.Delete(ctx, f.ID.String())
		return nil, err
	}

	return f, nil
}

// Get возвращает метаданные файла владельца.
func (s *FileService) Get(ctx context.Context, ownerID, id uuid.UUID) (*model.File, error) {
	f, err := s.files.GetByID(ctx, id, ownerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFileNotFound
	}
	return f, err
}

func (s *FileService) List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.File, error) {
	return s.files.ListByOwner(ctx, ownerID, limit, offset)
}

// Download возвращает метаданные (нужны для заголовков ответа) и поток с байтами.
func (s *FileService) Download(ctx context.Context, ownerID, id uuid.UUID) (*model.File, io.ReadCloser, error) {
	// Сначала проверяем по базе, что файл существует и принадлежит пользователю.
	f, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return nil, nil, err
	}

	body, err := s.storage.Download(ctx, f.ID.String())
	if err != nil {
		return nil, nil, err
	}
	return f, body, nil
}

// Delete удаляет файл: сначала запись в базе, потом объект в MinIO.
// Порядок важен: если упадёт удаление из MinIO, останется лишний объект
// (его не видно пользователю), а не запись, указывающая на пустоту.
func (s *FileService) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	err := s.files.Delete(ctx, id, ownerID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrFileNotFound
	}
	if err != nil {
		return err
	}

	return s.storage.Delete(ctx, id.String())
}
