// Package config загружает настройки приложения из переменных окружения.
package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config — корневая структура настроек. Новые секции (Postgres, Redis, Kafka, MinIO)
// будем добавлять по мере того, как они понадобятся.
type Config struct {
	Env  string `env:"APP_ENV" env-default:"local"` // local | prod
	HTTP HTTPConfig
}

type HTTPConfig struct {
	Port string `env:"HTTP_PORT" env-default:"8000"`
	// ReadHeaderTimeout защищает от медленных клиентов (Slowloris).
	// Обычные ReadTimeout/WriteTimeout намеренно не ставим: они ограничивают
	// всё время запроса и оборвали бы загрузку/скачивание больших файлов.
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" env-default:"5s"`
	// ShutdownTimeout — сколько ждём завершения активных запросов при остановке.
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

// Load читает переменные окружения в Config, подставляя значения по умолчанию.
func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}
	return &cfg, nil
}
