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
}

// Load читает переменные окружения в Config.
// env-default — значение по умолчанию, env-required — без переменной будет ошибка.
func Load() (*Config, error) {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)
	return &cfg, err
}
