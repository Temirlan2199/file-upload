// Мигратор применяет SQL-миграции из пакета migrations.
//
//	go run ./cmd/migrator up       — применить все новые
//	go run ./cmd/migrator down     — откатить последнюю
//	go run ./cmd/migrator status   — показать состояние
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx" для database/sql
	"github.com/pressly/goose/v3"

	"github.com/Temirlan2199/file-upload/internal/config"
	"github.com/Temirlan2199/file-upload/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	// Мигратору нужен только Postgres, поэтому читаем одну секцию,
	// а не весь Config (иначе потребовался бы, например, JWT_SECRET).
	var cfg config.PostgresConfig
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return fmt.Errorf("read env: %w", err)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	// goose работает через стандартный database/sql, поэтому здесь
	// используем pgx как драйвер database/sql, а не pgxpool.
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.RunContext(context.Background(), command, db, ".", os.Args[min(2, len(os.Args)):]...); err != nil {
		return fmt.Errorf("goose %s: %w", command, err)
	}
	return nil
}
