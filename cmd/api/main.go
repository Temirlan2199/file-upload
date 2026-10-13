// Точка входа HTTP API. Здесь только «сборка» приложения:
// читаем конфиг, создаём зависимости, запускаем сервер и корректно его останавливаем.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/Temirlan2199/file-upload/internal/config"
	"github.com/Temirlan2199/file-upload/internal/handler"
	"github.com/Temirlan2199/file-upload/internal/logger"
)

func main() {
	// Вся логика в run(), чтобы возвращать ошибки обычным способом,
	// а os.Exit вызывать в одном месте (os.Exit не выполняет defer).
	if err := run(); err != nil {
		slog.Error("app stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Env)
	slog.SetDefault(log)

	if cfg.Env != "local" {
		gin.SetMode(gin.ReleaseMode)
	}

	// ctx отменится, когда придёт SIGINT (Ctrl+C) или SIGTERM (docker stop).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           handler.NewRouter(log),
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}

	// ListenAndServe блокирует горутину, поэтому запускаем его отдельно,
	// а ошибку старта (например, порт занят) передаём через канал.
	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	}

	// Graceful shutdown: перестаём принимать новые соединения
	// и даём активным запросам ShutdownTimeout на завершение.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	log.Info("server stopped gracefully")
	return nil
}
