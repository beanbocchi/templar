package cache

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/beanbocchi/templar/internal/client/objectstore"
	"github.com/beanbocchi/templar/internal/utils/ioutil"
)

// EvictionPolicy defines the interface for cache eviction strategies.
type EvictionPolicy interface {
	// Access updates the access time of a key in the LRU list.
	Access(key string)
	// Add adds a new item to the LRU list and returns the keys that should be evicted from the cache storage.
	Add(key string, size int64) []string
	// Remove removes an item from the LRU list and returns the keys that should be evicted from the cache storage.
	Remove(key string)
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
	sizeReader := ioutil.NewSizeReader(content)
	if err := c.cache.Upload(ctx, key, sizeReader); err != nil {
		return fmt.Errorf("upload to cache: %w", err)
	}
	keys := c.evictionPolicy.Add(key, sizeReader.Size)
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		// If cacheUpload fails, we MUST drain the pipe to prevent the primary upload from blocking
		if err := c.cacheUpload(ctx, key, pr); err != nil {
			cacheErr = err
			// Drain the reader to keep the writer unblocked
			_, _ = io.Copy(io.Discard, pr)
		}
	}()

	// Upload to primary (drives the teeReader)
	if err := c.primary.Upload(ctx, key, teeReader); err != nil {
		pw.CloseWithError(err) // Close writer with error to stop the reader
		wg.Wait()
		return fmt.Errorf("upload to primary: %w", err)
	}
	// Successful download from primary means everything was read.
	// Close the pipe writer to signal EOF to the cache reader
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

// CreateMultipart starts a multipart upload on the primary store. We defer
// caching until completion to avoid duplicating multipart state.
func (c *CacheClient) CreateMultipart(ctx context.Context, key string) (string, error) {
	return c.primary.CreateMultipart(ctx, key)
}

// UploadPart forwards multipart parts to the primary store.
func (c *CacheClient) UploadPart(
	ctx context.Context,
	key string,
	uploadID string,
	partNumber int,
	content io.Reader,
) error {
	return c.primary.UploadPart(ctx, key, uploadID, partNumber, content)
}

// CompleteMultipart finalizes the upload on primary, then caches the finished
// object best-effort.
func (c *CacheClient) CompleteMultipart(ctx context.Context, key string, uploadID string) error {
	if err := c.primary.CompleteMultipart(ctx, key, uploadID); err != nil {
		return fmt.Errorf("complete multipart on primary: %w", err)
	}

	// Refresh cache with the new object contents.
	reader, err := c.primary.Download(ctx, key)
	if err != nil {
		slog.Warn("cache refresh after multipart complete failed (download)", "key", key, "error", err)
		return nil
	}
	defer reader.Close()

	if err := c.cacheUpload(ctx, key, reader); err != nil {
		slog.Warn("cache refresh after multipart complete failed (upload)", "key", key, "error", err)
	}

	return nil
}

// AbortMultipart forwards abort to the primary store.
func (c *CacheClient) AbortMultipart(ctx context.Context, key, uploadID string) error {
	return c.primary.AbortMultipart(ctx, key, uploadID)
}

// Get retrieves a file from cache first, then falls back to primary.
func (c *CacheClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	cacheReader, err := c.cache.Download(ctx, key)
	if err == nil {
		// Found in cache - update access time for LRU
		c.evictionPolicy.Access(key)
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
			// Drain pipe to prevent blocking the download
			_, _ = io.Copy(io.Discard, pr)
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
		c.evictionPolicy.Remove(key)
	}

	if err := c.primary.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete from primary: %w", err)
	}

	return nil
}
