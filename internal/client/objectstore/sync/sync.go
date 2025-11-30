package sync

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/beanbocchi/templar/internal/client/objectstore"
	"github.com/beanbocchi/templar/internal/utils/ioutil"
)

// SyncConfig configures the synchronized objectstore wrapper.
type SyncConfig struct {
	// Client is the underlying objectstore client to wrap with locking.
	Client objectstore.Client
}

// SyncClient wraps an objectstore client with per-key locking for concurrency safety.
type SyncClient struct {
	client objectstore.Client
	locks  sync.Map // map[string]*sync.RWMutex
}

// NewSyncClient creates a new synchronized objectstore client wrapper.
func NewSyncClient(cfg SyncConfig) (*SyncClient, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("client is required")
	}

	return &SyncClient{
		client: cfg.Client,
	}, nil
}

// getLock returns a per-key RWMutex, creating one if it doesn't exist.
func (c *SyncClient) getLock(key string) *sync.RWMutex {
	lock, _ := c.locks.LoadOrStore(key, &sync.RWMutex{})
	return lock.(*sync.RWMutex)
}

// Upload uploads an object with write locking.
func (c *SyncClient) Upload(ctx context.Context, key string, content io.Reader) error {
	lock := c.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	return c.client.Upload(ctx, key, content)
}

// Download downloads an object with read locking.
func (c *SyncClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	lock := c.getLock(key)
	lock.RLock()

	file, err := c.client.Download(ctx, key)
	if err != nil {
		lock.RUnlock()
		return nil, fmt.Errorf("download: %w", err)
	}

	return ioutil.NewLockedReadCloser(file, lock), nil
}

// Delete deletes an object with write locking.
func (c *SyncClient) Delete(ctx context.Context, key string) error {
	lock := c.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	return c.client.Delete(ctx, key)
}
