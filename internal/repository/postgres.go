// Package repository — слой доступа к данным. Здесь живёт весь SQL;
// выше (в service) про SQL и pgx ничего не знают.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Ошибки репозитория. Сервис проверяет их через errors.Is и
// переводит в свои, бизнесовые ошибки.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// Код ошибки Postgres для нарушения UNIQUE-ограничения.
const pgUniqueViolation = "23505"

// NewPostgresPool создаёт пул соединений и проверяет, что база доступна.
// Пул потокобезопасен: один на всё приложение, используется из всех горутин.
func NewPostgresPool(ctx context.Context, dsn string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// NewWithConfig подключается лениво, поэтому явно пингуем:
	// лучше упасть на старте, чем на первом запросе пользователя.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// isUniqueViolation проверяет, что err — нарушение UNIQUE в Postgres.
// errors.As достаёт *pgconn.PgError из цепочки обёрнутых ошибок.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}
