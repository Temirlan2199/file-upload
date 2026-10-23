package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter ограничивает число запросов: не больше limit за окно window.
//
// Алгоритм «фиксированное окно» (fixed window):
//   - для каждого клиента в Redis есть счётчик "ratelimit:<ключ>";
//   - каждый запрос увеличивает счётчик на 1 (INCR);
//   - при первом запросе ставим счётчику срок жизни = window;
//   - когда срок вышел, Redis удаляет счётчик, и отсчёт начинается заново;
//   - если счётчик больше limit — запрос отклоняем.
type RateLimiter struct {
	rdb    *redis.Client
	limit  int64
	window time.Duration
}

func NewRateLimiter(rdb *redis.Client, limit int64, window time.Duration) *RateLimiter {
	return &RateLimiter{rdb: rdb, limit: limit, window: window}
}

// Allow засчитывает запрос и отвечает, можно ли его пропустить.
func (l *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := "ratelimit:" + key

	// TxPipeline отправляет обе команды одним пакетом и выполняет их атомарно
	// (MULTI/EXEC): между INCR и EXPIRE никто не вклинится.
	pipe := l.rdb.TxPipeline()
	incr := pipe.Incr(ctx, redisKey) // счётчик + 1 (если ключа не было — создаст со значением 1)
	// EXPIRE ... NX — поставить срок жизни, только если его ещё нет.
	// Без NX каждый запрос продлевал бы окно, и при постоянном потоке
	// запросов счётчик не сбрасывался бы никогда.
	pipe.ExpireNX(ctx, redisKey, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	return incr.Val() <= l.limit, nil
}
