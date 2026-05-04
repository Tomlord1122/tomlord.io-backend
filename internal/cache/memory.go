package cache

import (
	"strings"
	"sync"
	"time"
)

type item struct {
	value      any
	expiration int64
}

// MemoryCache is a simple in-memory cache with TTL.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]item
}

var (
	instance *MemoryCache
	once     sync.Once
)

// GetInstance returns the singleton MemoryCache instance.
func GetInstance() *MemoryCache {
	once.Do(func() {
		instance = &MemoryCache{
			items: make(map[string]item),
		}
		go instance.cleanup()
	})
	return instance
}

// Set stores a value with a TTL.
func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = item{
		value:      value,
		expiration: time.Now().Add(ttl).UnixNano(),
	}
}

// Get retrieves a value if it hasn't expired.
func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().UnixNano() > it.expiration {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	return it.value, true
}

// Delete removes a key immediately.
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// DeletePrefix removes all keys starting with prefix.
func (c *MemoryCache) DeletePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.items {
		if strings.HasPrefix(key, prefix) {
			delete(c.items, key)
		}
	}
}

// cleanup runs periodically to evict expired entries.
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now().UnixNano()
		for key, it := range c.items {
			if now > it.expiration {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
