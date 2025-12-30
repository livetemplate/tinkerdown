// Package cache provides caching functionality for data sources.
package cache

import (
	"sync"
	"time"
)

// Entry represents a cached data entry
type Entry struct {
	Data      []map[string]interface{}
	ExpiresAt time.Time
	StaleAt   time.Time // For stale-while-revalidate: when data becomes stale (but still usable)
}

// IsExpired returns true if the entry has expired
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// IsStale returns true if the entry is stale but not expired
// Used for stale-while-revalidate strategy
func (e *Entry) IsStale() bool {
	now := time.Now()
	return now.After(e.StaleAt) && now.Before(e.ExpiresAt)
}

// Cache defines the interface for source caching
type Cache interface {
	// Get retrieves data from the cache
	// Returns (data, found, stale) where stale indicates data is stale but usable
	Get(key string) ([]map[string]interface{}, bool, bool)

	// Set stores data in the cache with the given TTL
	Set(key string, data []map[string]interface{}, ttl time.Duration)

	// SetWithStale stores data with separate stale and expire times
	// staleAfter: duration until data becomes stale
	// expireAfter: duration until data expires completely
	SetWithStale(key string, data []map[string]interface{}, staleAfter, expireAfter time.Duration)

	// Invalidate removes an entry from the cache
	Invalidate(key string)

	// InvalidateAll removes all entries from the cache
	InvalidateAll()
}

// MemoryCache is an in-memory cache implementation with TTL support
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*Entry

	// For background cleanup
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	stopOnce        sync.Once // Ensures Stop() is idempotent
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		entries:         make(map[string]*Entry),
		cleanupInterval: time.Minute,
		stopCleanup:     make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// Get retrieves data from the cache
// Returns (data, found, stale) where stale indicates data is stale but usable
func (c *MemoryCache) Get(key string) ([]map[string]interface{}, bool, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false, false
	}

	if entry.IsExpired() {
		// Entry has completely expired, remove it
		c.Invalidate(key)
		return nil, false, false
	}

	return entry.Data, true, entry.IsStale()
}

// Set stores data in the cache with the given TTL
func (c *MemoryCache) Set(key string, data []map[string]interface{}, ttl time.Duration) {
	c.SetWithStale(key, data, ttl, ttl)
}

// SetWithStale stores data with separate stale and expire times
func (c *MemoryCache) SetWithStale(key string, data []map[string]interface{}, staleAfter, expireAfter time.Duration) {
	now := time.Now()
	entry := &Entry{
		Data:      data,
		StaleAt:   now.Add(staleAfter),
		ExpiresAt: now.Add(expireAfter),
	}

	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
}

// Invalidate removes an entry from the cache
func (c *MemoryCache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// InvalidateAll removes all entries from the cache
func (c *MemoryCache) InvalidateAll() {
	c.mu.Lock()
	c.entries = make(map[string]*Entry)
	c.mu.Unlock()
}

// cleanupLoop periodically removes expired entries
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes all expired entries
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// Stop stops the background cleanup goroutine
// Safe to call multiple times
func (c *MemoryCache) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCleanup)
	})
}

// Len returns the number of entries in the cache (for testing)
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
