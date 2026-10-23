package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Temirlan2199/file-upload/internal/model"
)

// FileCache хранит метаданные файлов в Redis, чтобы не ходить каждый раз в Postgres.
// Ключ: "file:<id>", значение: model.File в JSON.
type FileCache struct {
	rdb *redis.Client
	ttl time.Duration // сколько живёт запись в кэше
}

func NewFileCache(rdb *redis.Client, ttl time.Duration) *FileCache {
	return &FileCache{rdb: rdb, ttl: ttl}
}

func fileKey(id uuid.UUID) string {
	return "file:" + id.String()
}

// Get достаёт файл из кэша. Если ключа нет — ErrNotFound.
func (c *FileCache) Get(ctx context.Context, id uuid.UUID) (*model.File, error) {
	// GET file:<id> → строка с JSON.
	data, err := c.rdb.Get(ctx, fileKey(id)).Bytes()
	if errors.Is(err, redis.Nil) { // redis.Nil — «такого ключа нет»
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Превращаем JSON обратно в структуру.
	var f model.File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Set кладёт файл в кэш на c.ttl. По истечении времени Redis удалит ключ сам.
func (c *FileCache) Set(ctx context.Context, f *model.File) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	// SET file:<id> <json> EX <ttl>
	return c.rdb.Set(ctx, fileKey(f.ID), data, c.ttl).Err()
}

// Delete убирает файл из кэша (вызываем при удалении файла).
func (c *FileCache) Delete(ctx context.Context, id uuid.UUID) error {
	// DEL file:<id>
	return c.rdb.Del(ctx, fileKey(id)).Err()
}
