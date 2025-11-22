package cache

import (
	"context"
	"fmt"
	"io"

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

// Upload uploads to both cache and primary storage.
func (c *CacheClient) Upload(ctx context.Context, key string, content io.Reader) error {
	err := c.primary.Upload(ctx, key, content)
	if err != nil {
		return fmt.Errorf("upload to primary: %w", err)
	}

	// Upload to cache
	if err := c.cache.Upload(ctx, key, content); err != nil {
		fmt.Printf("Warning: failed to cache %s: %v\n", key, err)
	} else {
		// Notify eviction policy that item was successfully added, it returns the keys that should be evicted.
		keys := c.evictionPolicy.OnAdd(key)
		for _, evictKey := range keys {
			if err := c.cache.Delete(ctx, evictKey); err != nil {
				fmt.Printf("Warning: failed to evict %s: %v\n", evictKey, err)
			}
		}
	}

	return nil
}

// Get retrieves a file from cache first, then falls back to primary.
func (c *CacheClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	reader, err := c.cache.Download(ctx, key)
	if err == nil {
		// Found in cache - update access time for LRU
		c.evictionPolicy.OnAccess(key)
		return reader, nil
	}

	primaryReader, err := c.primary.Download(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get from primary: %w", err)
	}

	// Cache the file in the background
	go func() {
		if err := c.cache.Upload(context.Background(), key, primaryReader); err != nil {
			fmt.Printf("Warning: failed to cache %s: %v\n", key, err)
		}
	}()

	return primaryReader, nil
}

// Delete deletes a file from both cache and primary storage.
func (c *CacheClient) Delete(ctx context.Context, key string) error {
	if err := c.cache.Delete(ctx, key); err != nil {
		fmt.Printf("Warning: failed to delete from cache %s: %v\n", key, err)
	} else {
		c.evictionPolicy.OnRemove(key)
	}

	if err := c.primary.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete from primary: %w", err)
	}

	return nil
}
