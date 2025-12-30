package cache

import (
	"testing"
	"time"
)

func TestMemoryCacheBasic(t *testing.T) {
	c := NewMemoryCache()
	defer c.Stop()

	// Initially empty
	_, found, _ := c.Get("test")
	if found {
		t.Error("expected cache miss for non-existent key")
	}

	// Set a value
	data := []map[string]interface{}{
		{"id": 1, "name": "test"},
	}
	c.Set("test", data, time.Minute)

	// Get the value back
	result, found, stale := c.Get("test")
	if !found {
		t.Error("expected cache hit")
	}
	if stale {
		t.Error("expected fresh data, got stale")
	}
	if len(result) != 1 || result[0]["name"] != "test" {
		t.Errorf("unexpected data: %v", result)
	}
}

func TestMemoryCacheTTL(t *testing.T) {
	c := NewMemoryCache()
	defer c.Stop()

	data := []map[string]interface{}{{"id": 1}}
	c.Set("short", data, 50*time.Millisecond)

	// Immediately available
	_, found, _ := c.Get("short")
	if !found {
		t.Error("expected cache hit immediately after set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	_, found, _ = c.Get("short")
	if found {
		t.Error("expected cache miss after TTL expired")
	}
}

func TestMemoryCacheInvalidate(t *testing.T) {
	c := NewMemoryCache()
	defer c.Stop()

	data := []map[string]interface{}{{"id": 1}}
	c.Set("test1", data, time.Minute)
	c.Set("test2", data, time.Minute)

	// Both exist
	_, found1, _ := c.Get("test1")
	_, found2, _ := c.Get("test2")
	if !found1 || !found2 {
		t.Error("expected both keys to exist")
	}

	// Invalidate one
	c.Invalidate("test1")

	_, found1, _ = c.Get("test1")
	_, found2, _ = c.Get("test2")
	if found1 {
		t.Error("expected test1 to be invalidated")
	}
	if !found2 {
		t.Error("expected test2 to still exist")
	}
}

func TestMemoryCacheInvalidateAll(t *testing.T) {
	c := NewMemoryCache()
	defer c.Stop()

	data := []map[string]interface{}{{"id": 1}}
	c.Set("test1", data, time.Minute)
	c.Set("test2", data, time.Minute)
	c.Set("test3", data, time.Minute)

	if c.Len() != 3 {
		t.Errorf("expected 3 entries, got %d", c.Len())
	}

	c.InvalidateAll()

	if c.Len() != 0 {
		t.Errorf("expected 0 entries after InvalidateAll, got %d", c.Len())
	}
}

func TestMemoryCacheStaleWhileRevalidate(t *testing.T) {
	c := NewMemoryCache()
	defer c.Stop()

	data := []map[string]interface{}{{"id": 1}}

	// Set with stale time (50ms) less than expire time (200ms)
	c.SetWithStale("swr", data, 50*time.Millisecond, 200*time.Millisecond)

	// Immediately fresh
	_, found, stale := c.Get("swr")
	if !found {
		t.Error("expected cache hit")
	}
	if stale {
		t.Error("expected fresh data immediately after set")
	}

	// Wait until stale but not expired
	time.Sleep(80 * time.Millisecond)

	_, found, stale = c.Get("swr")
	if !found {
		t.Error("expected cache hit (stale)")
	}
	if !stale {
		t.Error("expected stale data after stale time")
	}

	// Wait until expired
	time.Sleep(150 * time.Millisecond)

	_, found, _ = c.Get("swr")
	if found {
		t.Error("expected cache miss after expire time")
	}
}

func TestEntryIsExpired(t *testing.T) {
	now := time.Now()

	// Not expired
	entry := &Entry{
		Data:      nil,
		ExpiresAt: now.Add(time.Minute),
		StaleAt:   now.Add(30 * time.Second),
	}
	if entry.IsExpired() {
		t.Error("expected entry to not be expired")
	}

	// Expired
	entry.ExpiresAt = now.Add(-time.Minute)
	if !entry.IsExpired() {
		t.Error("expected entry to be expired")
	}
}

func TestEntryIsStale(t *testing.T) {
	now := time.Now()

	// Fresh (not stale)
	entry := &Entry{
		Data:      nil,
		StaleAt:   now.Add(time.Minute),
		ExpiresAt: now.Add(2 * time.Minute),
	}
	if entry.IsStale() {
		t.Error("expected entry to be fresh")
	}

	// Stale but not expired
	entry.StaleAt = now.Add(-30 * time.Second)
	if !entry.IsStale() {
		t.Error("expected entry to be stale")
	}

	// Expired (not stale because it's past the window)
	entry.ExpiresAt = now.Add(-time.Second)
	if entry.IsStale() {
		t.Error("expected entry to not be stale when expired")
	}
}

func TestMemoryCacheLen(t *testing.T) {
	c := NewMemoryCache()
	defer c.Stop()

	if c.Len() != 0 {
		t.Errorf("expected 0 entries, got %d", c.Len())
	}

	data := []map[string]interface{}{{"id": 1}}
	c.Set("a", data, time.Minute)
	c.Set("b", data, time.Minute)

	if c.Len() != 2 {
		t.Errorf("expected 2 entries, got %d", c.Len())
	}
}

func TestMemoryCacheStopIdempotent(t *testing.T) {
	c := NewMemoryCache()

	// Calling Stop() multiple times should not panic
	c.Stop()
	c.Stop()
	c.Stop()
}
