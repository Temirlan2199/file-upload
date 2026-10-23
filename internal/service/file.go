package service

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/Temirlan2199/file-upload/internal/cache"
	"github.com/Temirlan2199/file-upload/internal/model"
	"github.com/Temirlan2199/file-upload/internal/repository"
)

var (
	ErrFileNotFound  = errors.New("file not found")
	ErrShareNotFound = errors.New("share link not found or expired")
)

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

// FileCache — кэш метаданных файлов (Redis).
type FileCache interface {
	Get(ctx context.Context, id uuid.UUID) (*model.File, error)
	Set(ctx context.Context, f *model.File) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ShareStore — ссылки «поделиться» (Redis).
type ShareStore interface {
	Save(ctx context.Context, token string, link model.ShareLink, ttl time.Duration) error
	Get(ctx context.Context, token string) (*model.ShareLink, error)
}

type FileService struct {
	files   FileRepository
	storage FileStorage
	cache   FileCache
	shares  ShareStore
}

func NewFileService(files FileRepository, storage FileStorage, cache FileCache, shares ShareStore) *FileService {
	return &FileService{files: files, storage: storage, cache: cache, shares: shares}
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
//
// Паттерн cache-aside («кэш сбоку»):
//  1. ищем в кэше (Redis) — если нашли, в базу не ходим;
//  2. не нашли — берём из базы (Postgres);
//  3. кладём результат в кэш, чтобы следующий запрос был быстрым.
func (s *FileService) Get(ctx context.Context, ownerID, id uuid.UUID) (*model.File, error) {
	// 1. Кэш. Любая ошибка (нет ключа или Redis недоступен) — не страшно,
	// просто идём в базу: кэш только ускоряет, без него всё работает.
	if f, err := s.cache.Get(ctx, id); err == nil {
		// В кэше ключ — только id файла, без владельца. Поэтому проверку
		// «файл принадлежит пользователю» делаем здесь, иначе через кэш
		// можно было бы получить чужой файл.
		if f.OwnerID != ownerID {
			return nil, ErrFileNotFound
		}
		return f, nil
	}

	// 2. База.
	f, err := s.files.GetByID(ctx, id, ownerID)
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

func (s *FileService) List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.File, error) {
	return s.files.ListByOwner(ctx, ownerID, limit, offset)
}

// Download возвращает метаданные (нужны для заголовков ответа) и поток с байтами.
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

// Delete удаляет файл: сначала запись в базе, потом кэш и объект в MinIO.
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

	// Инвалидация кэша: удаляем устаревшую запись, иначе Get ещё
	// несколько минут «находил» бы уже удалённый файл.
	_ = s.cache.Delete(ctx, id)

	return s.storage.Delete(ctx, id.String())
}

// Share создаёт публичную ссылку на файл, которая работает ttl времени.
// Возвращает токен ссылки и момент, когда она истечёт.
func (s *FileService) Share(ctx context.Context, ownerID, id uuid.UUID, ttl time.Duration) (string, time.Time, error) {
	// Делиться можно только своим существующим файлом.
	if _, err := s.Get(ctx, ownerID, id); err != nil {
		return "", time.Time{}, err
	}

	// rand.Text() — 26 случайных символов (128 бит) из криптостойкого
	// генератора. Угадать такой токен перебором невозможно.
	token := rand.Text()

	link := model.ShareLink{FileID: id, OwnerID: ownerID}
	if err := s.shares.Save(ctx, token, link, ttl); err != nil {
		return "", time.Time{}, err
	}
	return token, time.Now().Add(ttl), nil
}

// DownloadShared — скачивание по публичной ссылке, без авторизации.
func (s *FileService) DownloadShared(ctx context.Context, token string) (*model.File, io.ReadCloser, error) {
	link, err := s.shares.Get(ctx, token)
	if errors.Is(err, cache.ErrNotFound) {
		return nil, nil, ErrShareNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	// Дальше — обычное скачивание от имени владельца.
	// Если файл уже удалили, Download вернёт ErrFileNotFound.
	return s.Download(ctx, link.OwnerID, link.FileID)
}
