// Package config загружает настройки приложения из переменных окружения.
package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string        `env:"APP_ENV" env-default:"local"` // local | prod
	HTTPPort    string        `env:"HTTP_PORT" env-default:"8000"`
	PostgresDSN string        `env:"POSTGRES_DSN" env-required:"true"`
	JWTSecret   string        `env:"JWT_SECRET" env-required:"true"`
	JWTTTL      time.Duration `env:"JWT_TTL" env-default:"15m"`

	MinioEndpoint  string `env:"MINIO_ENDPOINT" env-default:"localhost:9000"`
	MinioAccessKey string `env:"MINIO_ACCESS_KEY" env-required:"true"`
	MinioSecretKey string `env:"MINIO_SECRET_KEY" env-required:"true"`
	MinioBucket    string `env:"MINIO_BUCKET" env-default:"files"`
	MinioUseSSL    bool   `env:"MINIO_USE_SSL" env-default:"false"`

	RedisAddr string `env:"REDIS_ADDR" env-default:"localhost:6379"`

	// PublicURL — внешний адрес сервиса, из него собираются ссылки «поделиться».
	PublicURL string `env:"PUBLIC_URL" env-default:"http://localhost:8000"`
}

// Load читает переменные окружения в Config.
// env-default — значение по умолчанию, env-required — без переменной будет ошибка.
func Load() (*Config, error) {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)
	return &cfg, err
}
