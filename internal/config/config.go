// Package config загружает настройки приложения из переменных окружения.
package config

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config — корневая структура настроек. Новые секции (Redis, Kafka, MinIO)
// будем добавлять по мере того, как они понадобятся.
type Config struct {
	Env      string `env:"APP_ENV" env-default:"local"` // local | prod
	HTTP     HTTPConfig
	Postgres PostgresConfig
	JWT      JWTConfig
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

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST" env-default:"localhost"`
	Port     int    `env:"POSTGRES_PORT" env-default:"5432"`
	User     string `env:"POSTGRES_USER" env-required:"true"`
	Password string `env:"POSTGRES_PASSWORD" env-required:"true"`
	DB       string `env:"POSTGRES_DB" env-required:"true"`
	SSLMode  string `env:"POSTGRES_SSLMODE" env-default:"disable"`
	// MaxConns — максимум соединений в пуле pgxpool.
	MaxConns int32 `env:"POSTGRES_MAX_CONNS" env-default:"10"`
}

// DSN собирает строку подключения вида postgres://user:pass@host:port/db?sslmode=disable.
// url.URL сам экранирует спецсимволы в пароле.
func (c PostgresConfig) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     c.Host + ":" + strconv.Itoa(c.Port),
		Path:     c.DB,
		RawQuery: "sslmode=" + c.SSLMode,
	}
	return u.String()
}

type JWTConfig struct {
	// Secret — ключ подписи токенов (HS256). Без него приложение не стартует.
	Secret    string        `env:"JWT_SECRET" env-required:"true"`
	AccessTTL time.Duration `env:"JWT_ACCESS_TTL" env-default:"15m"`
}

// Load читает переменные окружения в Config, подставляя значения по умолчанию.
func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}
	return &cfg, nil
}
