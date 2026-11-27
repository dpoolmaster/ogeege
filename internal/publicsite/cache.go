package publicsite

import (
	"sync"
	"time"
)

const (
	defaultCacheTTL   = 60 * time.Second
	defaultCacheLimit = 1000
)

type cacheEntry struct {
	body      []byte
	createdAt time.Time
	expiresAt time.Time
}

// Cache — in-memory TTL-кэш HTML-ответов публичных страниц.
type Cache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	ttl     time.Duration
	max     int
	now     func() time.Time
}

// NewCache создаёт кэш с TTL и верхней границей записей.
func NewCache(ttl time.Duration, maxEntries int) *Cache {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	if maxEntries <= 0 {
		maxEntries = defaultCacheLimit
	}

	return &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		max:     maxEntries,
		now:     time.Now,
	}
}

// Get возвращает сохранённый ответ, если он не истёк.
func (c *Cache) Get(key string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if !c.now().Before(entry.expiresAt) {
		delete(c.entries, key)
		return nil, false
	}

	return entry.body, true
}

// Set сохраняет ответ до истечения TTL.
func (c *Cache) Set(key string, body []byte) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if len(c.entries) >= c.max {
		c.evict(now)
	}

	copied := make([]byte, len(body))
	copy(copied, body)
	c.entries[key] = cacheEntry{
		body:      copied,
		createdAt: now,
		expiresAt: now.Add(c.ttl),
	}
}

// Clear очищает кэш.
func (c *Cache) Clear() {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
}

func (c *Cache) evict(now time.Time) {
	for key, entry := range c.entries {
		if !now.Before(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
	if len(c.entries) < c.max {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	for key, entry := range c.entries {
		if oldestKey == "" || entry.createdAt.Before(oldestTime) || (entry.createdAt.Equal(oldestTime) && key < oldestKey) {
			oldestKey = key
			oldestTime = entry.createdAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}
