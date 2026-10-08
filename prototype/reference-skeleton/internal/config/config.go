package config

import (
	"errors"
	"fmt"
	"log/slog"
)

const (
	defaultAddr     = ":4000"
	defaultLogLevel = slog.LevelInfo
)

type Config struct {
	Addr     string
	LogLevel slog.Level
}

// Load reads every setting from environment variables, collecting all problems rather than stopping at the first.
func Load(getenv func(string) string) (Config, error) {
	logLevel, logLevelErr := parseLogLevel(getenv("LOG_LEVEL"))

	cfg := Config{
		Addr:     withDefault(getenv("ADDR"), defaultAddr),
		LogLevel: logLevel,
	}

	if err := errors.Join(logLevelErr); err != nil {
		return Config{}, fmt.Errorf("loading config: %w", err)
	}

	return cfg, nil
}

func withDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func parseLogLevel(value string) (slog.Level, error) {
	if value == "" {
		return defaultLogLevel, nil
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return 0, fmt.Errorf("LOG_LEVEL: %w", err)
	}

	return level, nil
}
