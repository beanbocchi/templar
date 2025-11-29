package cache

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/beanbocchi/templar/internal/client/objectstore"
)

// EvictionPolicy defines the interface for cache eviction strategies.
type EvictionPolicy interface {
	// OnAccess is called when a cache key is accessed (read).
	OnAccess(key string)
	// OnAdd is called when a new item is successfully added to the cache, it returns the keys that should be evicted.
	OnAdd(key string) []string
	// OnRemove is called when an item is removed from the cache.
	OnRemove(key string)
}

// CacheConfig configures the cache storage.
type CacheConfig struct {
	// Cache is the cache storage client.
	Cache objectstore.Client
	// Primary is the primary storage client (e.g., Storj).
	Primary objectstore.Client
	// EvictionPolicy is the eviction policy for the cache (e.g., LRU with size management).
	EvictionPolicy EvictionPolicy
}

// CacheClient implements a caching layer over cache and primary storage with eviction support.
type CacheClient struct {
	cache          objectstore.Client
	primary        objectstore.Client
	evictionPolicy EvictionPolicy
}

// NewCacheClient creates a new cache storage client.
func NewCacheClient(cfg CacheConfig) (*CacheClient, error) {
	if cfg.Cache == nil {
		return nil, fmt.Errorf("cache storage client is required")
	}
	if cfg.Primary == nil {
		return nil, fmt.Errorf("primary storage client is required")
	}
	if cfg.EvictionPolicy == nil {
		return nil, fmt.Errorf("eviction policy is required")
	}

	return &CacheClient{
		cache:          cfg.Cache,
		primary:        cfg.Primary,
		evictionPolicy: cfg.EvictionPolicy,
	}, nil
}

func (c *CacheClient) cacheUpload(ctx context.Context, key string, content io.Reader) error {
	if err := c.cache.Upload(ctx, key, content); err != nil {
		return fmt.Errorf("upload to cache: %w", err)
	}
	keys := c.evictionPolicy.OnAdd(key)
	for _, evictKey := range keys {
		if err := c.cache.Delete(ctx, evictKey); err != nil {
			slog.Warn("failed to evict", "key", evictKey, "error", err)
		}
	}
	return nil
}

// Upload uploads to both cache and primary storage.
func (c *CacheClient) Upload(ctx context.Context, key string, content io.Reader) error {
	// Create a pipe for the cache
	pr, pw := io.Pipe()

	// one stream goes to primary, one to the pipe
	teeReader := io.TeeReader(content, pw)

	var cacheErr error
	var wg sync.WaitGroup

	// Upload to cache in parallel
	wg.Go(func() {
		defer pr.Close()
		if err := c.cacheUpload(ctx, key, pr); err != nil {
			cacheErr = err
		}
	})

	// Upload to primary (drives the teeReader)
	if err := c.primary.Upload(ctx, key, teeReader); err != nil {
		pw.CloseWithError(err)
		wg.Wait()
		return fmt.Errorf("upload to primary: %w", err)
	}
	if err := pw.Close(); err != nil {
		wg.Wait()
		return fmt.Errorf("close pipe writer: %w", err)
	}
	wg.Wait()

	if cacheErr != nil {
		slog.Warn("failed to cache", "key", key, "error", cacheErr)
	}

	return nil
}

// Get retrieves a file from cache first, then falls back to primary.
func (c *CacheClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	cacheReader, err := c.cache.Download(ctx, key)
	if err == nil {
		// Found in cache - update access time for LRU
		c.evictionPolicy.OnAccess(key)
		return cacheReader, nil
	}

	// teeReader(primaryReader) -> pipe -> cache

	primaryReader, err := c.primary.Download(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get from primary: %w", err)
	}

	pr, pw := io.Pipe()
	teeReader := io.TeeReader(primaryReader, pw)

	go func() {
		if err := c.cacheUpload(ctx, key, pr); err != nil {
			slog.Warn("failed to cache", "key", key, "error", err)
			// No need to close the pipe writer here, it will be closed when the teePipeReadCloser is closed.
		}
	}()

	// Return a reader that will close the pipe when closed (if we dont close it, the background upload will hang)
	return &teePipeReadCloser{
		Reader: teeReader,
		pipeW:  pw,
	}, nil
}

// teePipeReadCloser wraps a teeReader and closes the pipe when closed.
type teePipeReadCloser struct {
	io.Reader
	pipeW *io.PipeWriter
}

func (c *teePipeReadCloser) Close() error {
	return c.pipeW.Close()
}

// Delete deletes a file from both cache and primary storage.
func (c *CacheClient) Delete(ctx context.Context, key string) error {
	if err := c.cache.Delete(ctx, key); err != nil {
		slog.Warn("failed to delete from cache", "key", key, "error", err)
	} else {
		c.evictionPolicy.OnRemove(key)
	}

	if err := c.primary.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete from primary: %w", err)
	}

	return nil
}
