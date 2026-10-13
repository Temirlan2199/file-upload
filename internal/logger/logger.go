// Package logger создаёт slog.Logger в зависимости от окружения.
package logger

import (
	"log/slog"
	"os"
)

// New возвращает логгер: локально — читаемый текст с debug-уровнем,
// в проде — JSON (его удобно парсить системами сбора логов).
func New(env string) *slog.Logger {
	var h slog.Handler
	switch env {
	case "local":
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	default:
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	return slog.New(h)
}
