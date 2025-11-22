package service

import (
	"container/list"
	"context"
	"sync"

	"github.com/beanbocchi/templar/internal/client/objectstore/cache"
	"github.com/beanbocchi/templar/internal/db"
	"github.com/beanbocchi/templar/pkg/sqlc"
)

// lruEntry holds metadata for a cached item.
type lruEntry struct {
	key  string
	size int64
}

// LRUEvictionPolicy is an in-memory LRU implementation that tracks cache usage
// and total size, backed by template metadata stored in the database.
//
// NOTE: This currently only tracks usage and does not delete objects from the
// underlying cache storage. It can be extended later to call Delete on a cache
// client when items are evicted.
type LRUEvictionPolicy struct {
	mu sync.Mutex

	maxSizeBytes int64
	currentSize  int64

	// items maps cache keys to their position in the LRU list.
	items map[string]*list.Element
	// order keeps items ordered by recency (front = most recently used).
	order *list.List

	storage *sqlc.Storage
}

// NewLRUEvictionPolicy creates a new LRU eviction policy implementation that
// uses the provided storage to look up object metadata (e.g. file size) in the
// database. maxSizeBytes is the soft limit for total cached size.
func NewLRUEvictionPolicy(storage *sqlc.Storage, maxSizeBytes int64) cache.EvictionPolicy {
	return &LRUEvictionPolicy{
		maxSizeBytes: maxSizeBytes,
		items:        make(map[string]*list.Element),
		order:        list.New(),
		storage:      storage,
	}
}

// OnAccess is called when a cache key is accessed (read).
func (p *LRUEvictionPolicy) OnAccess(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if elem, ok := p.items[key]; ok {
		p.order.MoveToFront(elem)
	}
}

// OnAdd is called when a new item is successfully added to the cache.
// It returns the keys that should be evicted from the cache storage.
func (p *LRUEvictionPolicy) OnAdd(key string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If the key already exists, treat as access.
	if elem, ok := p.items[key]; ok {
		p.order.MoveToFront(elem)
		return nil
	}

	size := p.lookupSizeBytes(key)

	entry := &lruEntry{
		key:  key,
		size: size,
	}
	elem := p.order.PushFront(entry)
	p.items[key] = elem
	p.currentSize += size

	return p.evictIfNeeded()
}

// OnRemove is called when an item is removed from the cache.
func (p *LRUEvictionPolicy) OnRemove(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	elem, ok := p.items[key]
	if !ok {
		return
	}

	entry, _ := elem.Value.(*lruEntry)
	if entry != nil {
		p.currentSize -= entry.size
	}

	p.order.Remove(elem)
	delete(p.items, key)
}

// evictIfNeeded trims the LRU list so that total size stays within maxSizeBytes.
// It updates the in-memory tracking structure and returns the keys that should
// be evicted from the underlying cache storage.
func (p *LRUEvictionPolicy) evictIfNeeded() []string {
	if p.maxSizeBytes <= 0 {
		return nil
	}

	var evicted []string

	for p.currentSize > p.maxSizeBytes && p.order.Len() > 0 {
		back := p.order.Back()
		if back == nil {
			break
		}

		entry, _ := back.Value.(*lruEntry)
		if entry == nil {
			p.order.Remove(back)
			continue
		}

		delete(p.items, entry.key)
		p.order.Remove(back)
		p.currentSize -= entry.size
		evicted = append(evicted, entry.key)
	}

	return evicted
}

// lookupSizeBytes attempts to look up the file size for the given cache key
// using the template metadata stored in the database.
// If any step fails, this returns 0 and the item is still tracked but will not
// contribute towards the size limit.
func (p *LRUEvictionPolicy) lookupSizeBytes(key string) int64 {
	if p.storage == nil || p.storage.Queries == nil {
		return 0
	}

	tv, err := p.storage.Queries.GetTemplateVersion(context.Background(), db.GetTemplateVersionParams{
		ObjectKey: key,
	})
	if err != nil || tv.FileSize == nil {
		return 0
	}

	return *tv.FileSize
}
