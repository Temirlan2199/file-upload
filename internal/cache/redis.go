// Package cache — всё, что хранится в Redis: кэш метаданных файлов,
// счётчики rate limit и ссылки «поделиться».
//
// Redis — хранилище «ключ → значение» в оперативной памяти: очень быстрое,
// умеет автоматически удалять ключи по истечении времени (TTL).
// Поэтому в нём держат временные данные, которые не жалко потерять.
package cache

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound — ключа нет в Redis (или он уже истёк).
var ErrNotFound = errors.New("not found in cache")

// NewRedisClient подключается к Redis и проверяет соединение.
func NewRedisClient(ctx context.Context, addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})

	// Как и pgxpool, клиент подключается лениво — проверяем соединение явно.
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}
