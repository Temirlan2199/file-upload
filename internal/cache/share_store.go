package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Temirlan2199/file-upload/internal/model"
)

// ShareStore хранит ссылки «поделиться».
// Ключ: "share:<token>", значение: model.ShareLink в JSON, TTL = срок жизни ссылки.
// Redis сам удалит ключ, когда срок выйдет, — отдельная чистка не нужна.
type ShareStore struct {
	rdb *redis.Client
}

func NewShareStore(rdb *redis.Client) *ShareStore {
	return &ShareStore{rdb: rdb}
}

func shareKey(token string) string {
	return "share:" + token
}

// Save сохраняет ссылку на время ttl.
func (s *ShareStore) Save(ctx context.Context, token string, link model.ShareLink, ttl time.Duration) error {
	data, err := json.Marshal(link)
	if err != nil {
		return err
	}
	// SET share:<token> <json> EX <ttl>
	return s.rdb.Set(ctx, shareKey(token), data, ttl).Err()
}

// Get находит ссылку по токену. Если её нет или срок истёк — ErrNotFound.
func (s *ShareStore) Get(ctx context.Context, token string) (*model.ShareLink, error) {
	data, err := s.rdb.Get(ctx, shareKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var link model.ShareLink
	if err := json.Unmarshal(data, &link); err != nil {
		return nil, err
	}
	return &link, nil
}
