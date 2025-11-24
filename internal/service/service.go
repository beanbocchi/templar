package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/beanbocchi/templar/config"
	"github.com/beanbocchi/templar/internal/client/objectstore"
	"github.com/beanbocchi/templar/internal/client/objectstore/cache"
	"github.com/beanbocchi/templar/internal/client/objectstore/local"
	"github.com/beanbocchi/templar/internal/client/objectstore/stoj"
	"github.com/beanbocchi/templar/pkg/sqlc"
)

type Service struct {
	objectStore objectstore.Client
	storage     *sqlc.Storage

	jobs chan func()
}

func NewService(config *config.Config, sqliteDB *sql.DB) (*Service, error) {
	storage := sqlc.NewStorage(sqliteDB)

	localStore, err := local.NewClient(local.LocalConfig{
		Root:    config.Objectstore.Local.Root,
		BaseURL: config.Objectstore.Local.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create local store: %w", err)
	}

	storjStore, err := stoj.NewClient(context.Background(), stoj.StorjConfig{
		Bucket:      config.Objectstore.Storj.Bucket,
		AccessGrant: config.Objectstore.Storj.AccessGrant,
		BaseURL:     config.Objectstore.Storj.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create storj store: %w", err)
	}

	// Create LRU eviction policy with max size from config (convert MB to bytes)
	maxSizeBytes := config.Objectstore.Cache.MaxSize * 1024 * 1024

	cacheStore, err := cache.NewCacheClient(cache.CacheConfig{
		Cache:          localStore,
		Primary:        storjStore,
		EvictionPolicy: NewLRUEvictionPolicy(storage, maxSizeBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("create cache store: %w", err)
	}

	// Create a job queue
	jobs := make(chan func(), config.App.JobBuffer)
	go func() {
		for job := range jobs {
			job()
		}
	}()

	return &Service{
		objectStore: cacheStore,
		storage:     storage,
		jobs:        jobs,
	}, nil
}

func getKey(templateID uuid.UUID, version int64) string {
	return fmt.Sprintf("templates/%s/%d", templateID.String(), version)
}
