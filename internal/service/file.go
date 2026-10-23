package service

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Temirlan2199/file-upload/internal/model"
	"github.com/Temirlan2199/file-upload/internal/repository"
)

var ErrFileNotFound = errors.New("file not found")

// FileRepository — метаданные файлов в Postgres.
type FileRepository interface {
	Create(ctx context.Context, f *model.File) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.File, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.File, error)
	Delete(ctx context.Context, id, ownerID uuid.UUID) error
}

// FileStorage — сами байты файлов (MinIO).
type FileStorage interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// FileCache — кэш метаданных файлов (Redis).
type FileCache interface {
	Get(ctx context.Context, id uuid.UUID) (*model.File, error)
	Set(ctx context.Context, f *model.File) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// EventPublisher — отправка событий о файлах (Kafka).
type EventPublisher interface {
	PublishFileUploaded(ctx context.Context, fileID uuid.UUID) error
	PublishFileDeleted(ctx context.Context, fileID uuid.UUID) error
}

type FileService struct {
	files   FileRepository
	storage FileStorage
	cache   FileCache
	events  EventPublisher
	log     *slog.Logger
}

func NewFileService(files FileRepository, storage FileStorage, cache FileCache, events EventPublisher, log *slog.Logger) *FileService {
	return &FileService{files: files, storage: storage, cache: cache, events: events, log: log}
}

// Upload сохраняет файл: байты — в MinIO, метаданные — в Postgres.
func (s *FileService) Upload(ctx context.Context, ownerID uuid.UUID, name, contentType string, size int64, r io.Reader) (*model.File, error) {
	f := &model.File{
		ID:          uuid.New(), // генерируем id сами — он же ключ объекта в MinIO
		OwnerID:     ownerID,
		Name:        name,
		Size:        size,
		ContentType: contentType,
		Status:      model.FileStatusPending, // ready станет после обработки воркером
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

	// 3. Сообщаем в Kafka: «файл загружен». Обработает его воркер — позже,
	// в фоне. Пользователь не ждёт обработки и сразу получает ответ.
	// Если Kafka недоступна, файл всё равно сохранён — просто останется
	// в статусе pending (надёжно решается паттерном Transactional Outbox).
	// Ошибку не возвращаем клиенту, но обязательно пишем в лог —
	// иначе потерю события никто не заметит.
	if err := s.events.PublishFileUploaded(ctx, f.ID); err != nil {
		s.log.Error("publish file.uploaded", "file_id", f.ID, "error", err)
	}

	return f, nil
}

// findFile находит файл по id — любого владельца.
//
// Паттерн cache-aside («кэш сбоку»):
//  1. ищем в кэше (Redis) — если нашли, в базу не ходим;
//  2. не нашли — берём из базы (Postgres);
//  3. кладём результат в кэш, чтобы следующий запрос был быстрым.
func (s *FileService) findFile(ctx context.Context, id uuid.UUID) (*model.File, error) {
	// 1. Кэш. Любая ошибка (нет ключа или Redis недоступен) — не страшно,
	// просто идём в базу: кэш только ускоряет, без него всё работает.
	if f, err := s.cache.Get(ctx, id); err == nil {
		return f, nil
	}

	// 2. База.
	f, err := s.files.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}

	// 3. Запоминаем в кэше. Не получилось — не беда, ошибку игнорируем.
	_ = s.cache.Set(ctx, f)
	return f, nil
}

// Get возвращает метаданные файла, только если он принадлежит ownerID.
func (s *FileService) Get(ctx context.Context, ownerID, id uuid.UUID) (*model.File, error) {
	f, err := s.findFile(ctx, id)
	if err != nil {
		return nil, err
	}

	// Проверка доступа: чужой файл — для пользователя «не найден»
	// (404, а не 403 — чтобы не раскрывать, что такой файл вообще существует).
	if f.OwnerID != ownerID {
		return nil, ErrFileNotFound
	}
	return f, nil
}

func (s *FileService) List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.File, error) {
	return s.files.ListByOwner(ctx, ownerID, limit, offset)
}

// Download — скачивание своего файла. Возвращает метаданные (нужны для
// заголовков ответа) и поток с байтами.
func (s *FileService) Download(ctx context.Context, ownerID, id uuid.UUID) (*model.File, io.ReadCloser, error) {
	// Сначала проверяем, что файл существует и принадлежит пользователю.
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

// DownloadPublic — скачивание по публичной ссылке /s/{id}, без проверки владельца.
// Любой, кто знает id файла, может его скачать.
func (s *FileService) DownloadPublic(ctx context.Context, id uuid.UUID) (*model.File, io.ReadCloser, error) {
	f, err := s.findFile(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	body, err := s.storage.Download(ctx, f.ID.String())
	if err != nil {
		return nil, nil, err
	}
	return f, body, nil
}

// Delete удаляет файл: запись в базе и кэш — сразу, объект в MinIO — через воркер.
func (s *FileService) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	err := s.files.Delete(ctx, id, ownerID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrFileNotFound
	}
	if err != nil {
		return err
	}

	// Инвалидация кэша: удаляем устаревшую запись, иначе Get ещё
	// несколько минут «находил» бы уже удалённый файл.
	_ = s.cache.Delete(ctx, id)

	// Объект в MinIO удалит воркер по событию «файл удалён».
	// Для пользователя файл исчезает сразу (записи в базе уже нет),
	// а тяжёлая работа с хранилищем идёт в фоне.
	if err := s.events.PublishFileDeleted(ctx, id); err != nil {
		// Kafka недоступна — удаляем объект сами, сразу.
		s.log.Error("publish file.deleted, deleting object directly", "file_id", id, "error", err)
		return s.storage.Delete(ctx, id.String())
	}
	return nil
}
