package internal

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/beanbocchi/templar/config"
	"github.com/beanbocchi/templar/internal/service"
	"github.com/beanbocchi/templar/internal/transport"
	"github.com/beanbocchi/templar/pkg/binder"
	"github.com/beanbocchi/templar/pkg/validator"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "modernc.org/sqlite"
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

func SetupDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "templar.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	driver, err := sqlite.WithInstance(db, &sqlite.Config{
		MigrationsTable: "schema_migrations",
		DatabaseName:    "templar",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"sqlite", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func Start() error {
	db, err := SetupDatabase()
	if err != nil {
		return fmt.Errorf("failed to setup database: %w", err)
	}

	service, err := service.NewService(config.GetConfig(), db)
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	e := echo.New()
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	customVal, err := validator.New()
	if err != nil {
		return fmt.Errorf("failed to create validator: %v", err)
	}
	e.Validator = customVal
	e.Binder = binder.NewCustomBinder()
	transport.SetupRoute(e, service)
	go func() {
		if err := e.Start(":8080"); err != nil {
			log.Panicf("failed to start server: %v", err)
		}
	}()

	return nil
}
