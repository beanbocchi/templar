package cache

import (
	"container/list"
	"sync"
)

// lruEntry holds metadata for a cached item.
type lruEntry struct {
	key  string
	size int64
}

// LRUEvictionPolicy is an in-memory LRU implementation that tracks cache usage
// and total size, backed by template metadata stored in the database.
type LRUEvictionPolicy struct {
	mu sync.Mutex

	maxSizeBytes int64
	currentSize  int64

	// items maps cache keys to their position in the LRU list.
	items map[string]*list.Element
	// order keeps items ordered by recency (front = most recently used).
	order *list.List
}

// NewLRUEvictionPolicy creates a new LRU eviction policy implementation that
// uses the provided storage to look up object metadata (e.g. file size) in the
// database. maxSizeBytes is the soft limit for total cached size.
func NewLRUEvictionPolicy(maxSizeBytes int64) EvictionPolicy {
	return &LRUEvictionPolicy{
		maxSizeBytes: maxSizeBytes,
		items:        make(map[string]*list.Element),
		order:        list.New(),
	}
}

// Access updates the access time of a key in the LRU list.
func (p *LRUEvictionPolicy) Access(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if elem, ok := p.items[key]; ok {
		p.order.MoveToFront(elem)
	}
}

// Add adds a new item to the LRU list and returns the keys that should be evicted from the cache storage.
func (p *LRUEvictionPolicy) Add(key string, size int64) []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If the key already exists, treat as access.
	if elem, ok := p.items[key]; ok {
		p.order.MoveToFront(elem)
		return nil
	}

	entry := &lruEntry{
		key:  key,
		size: size,
	}
	elem := p.order.PushFront(entry)
	p.items[key] = elem
	p.currentSize += size

	return p.evictIfNeeded()
}

// Remove removes an item from the LRU list and returns the keys that should be evicted from the cache storage.
func (p *LRUEvictionPolicy) Remove(key string) {
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
