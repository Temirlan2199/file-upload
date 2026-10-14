// Package repository — слой доступа к данным. Весь SQL живёт здесь.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

// NewPostgresPool создаёт пул соединений с базой.
// Пул один на всё приложение и безопасно используется из разных горутин.
func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	// pgxpool.New не подключается сразу, поэтому проверяем соединение явно.
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}
