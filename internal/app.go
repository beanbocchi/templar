package internal

import (
	"log/slog"
	"os"

	"github.com/beanbocchi/templar/config"
)

// NewConfig provides the application configuration
func NewConfig() *config.Config {
	return config.GetConfig()
}

func SetupLogger() {
	cfg := config.GetConfig().Log

	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	})))
}
