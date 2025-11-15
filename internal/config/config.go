package config

import (
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

type ServerConfig struct {
	Host string `env:"SERVER_HOST,required"`
	Port string `env:"SERVER_PORT,required"`
}

type DatabaseConfig struct {
	User     string `env:"POSTGRES_USERNAME,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	Host     string `env:"POSTGRES_HOST,required"`
	Port     string `env:"POSTGRES_PORT,required"`
	Name     string `env:"POSTGRES_DATABASE,required"`
}

func Load() (*Config, error) {
	cfg := Config{}

	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) ParseLogLevel() slog.Level {
	levelStr := strings.ToLower(c.LogLevel)

	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
