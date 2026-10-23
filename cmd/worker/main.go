// Воркер — отдельная программа, которая в фоне обрабатывает события из Kafka:
// считает checksum загруженных файлов и удаляет объекты удалённых файлов из MinIO.
// Запуск: make worker (параллельно с make run).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Temirlan2199/file-upload/internal/broker"
	"github.com/Temirlan2199/file-upload/internal/cache"
	"github.com/Temirlan2199/file-upload/internal/config"
	"github.com/Temirlan2199/file-upload/internal/logger"
	"github.com/Temirlan2199/file-upload/internal/repository"
	"github.com/Temirlan2199/file-upload/internal/service"
	"github.com/Temirlan2199/file-upload/internal/storage"
	"github.com/Temirlan2199/file-upload/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Env)

	// ctx отменится по Ctrl+C или docker stop — цикл чтения сообщений завершится.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Воркеру нужны те же хранилища, что и API: база, MinIO и Redis (для кэша).
	db, err := repository.NewPostgresPool(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Error("connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	fileStorage, err := storage.NewMinioStorage(ctx,
		cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL)
	if err != nil {
		log.Error("connect to minio", "error", err)
		os.Exit(1)
	}

	rdb, err := cache.NewRedisClient(ctx, cfg.RedisAddr)
	if err != nil {
		log.Error("connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Слои: repository/storage/cache → service → worker (вместо handler).
	fileRepo := repository.NewFileRepository(db)
	fileCache := cache.NewFileCache(rdb, 10*time.Minute)
	processing := service.NewProcessingService(fileRepo, fileStorage, fileCache)

	// Топики должны существовать до подписки: на отсутствующий топик
	// consumer не подпишется и его сообщения не увидит.
	if err := broker.CreateTopics(cfg.KafkaAddr); err != nil {
		log.Error("create kafka topics", "error", err)
		os.Exit(1)
	}

	consumer := worker.NewConsumer(cfg.KafkaAddr, processing, log)
	defer consumer.Close()

	log.Info("worker started", "kafka", cfg.KafkaAddr)

	// Run блокирует, пока ctx не отменят.
	if err := consumer.Run(ctx); err != nil {
		log.Error("consumer", "error", err)
		os.Exit(1)
	}
	log.Info("worker stopped")
}
