// Точка входа HTTP API: читаем конфиг, создаём зависимости,
// запускаем сервер и корректно останавливаем его.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Temirlan2199/file-upload/internal/config"
	"github.com/Temirlan2199/file-upload/internal/handler"
	"github.com/Temirlan2199/file-upload/internal/logger"
	"github.com/Temirlan2199/file-upload/internal/repository"
	"github.com/Temirlan2199/file-upload/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Env)

	// ctx отменится, когда придёт Ctrl+C (SIGINT) или docker stop (SIGTERM).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := repository.NewPostgresPool(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Error("connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Собираем слои снизу вверх: repository → service → handler.
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	h := handler.New(log, authService)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second, // защита от «медленных» клиентов
	}

	// ListenAndServe блокирует выполнение, поэтому запускаем его в горутине.
	go func() {
		log.Info("http server started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server", "error", err)
			stop() // не смогли запуститься (например, порт занят) — завершаемся
		}
	}()

	<-ctx.Done() // ждём сигнала остановки
	log.Info("shutting down")

	// Graceful shutdown: новые запросы не принимаем,
	// активным даём до 10 секунд на завершение.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "error", err)
	}
	log.Info("server stopped")
}
