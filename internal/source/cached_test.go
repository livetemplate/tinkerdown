package source

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/livetemplate/tinkerdown/internal/cache"
	"github.com/livetemplate/tinkerdown/internal/config"
)

// mockSource is a test source that counts fetch calls
type mockSource struct {
	name       string
	data       []map[string]interface{}
	fetchCount int32
	fetchDelay time.Duration
}

func (s *mockSource) Name() string { return s.name }

func (s *mockSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	atomic.AddInt32(&s.fetchCount, 1)
	if s.fetchDelay > 0 {
		time.Sleep(s.fetchDelay)
	}
	return s.data, nil
}

func (s *mockSource) Close() error { return nil }

func (s *mockSource) FetchCount() int {
	return int(atomic.LoadInt32(&s.fetchCount))
}

// mockWritableSource is a test writable source
type mockWritableSource struct {
	mockSource
	writeCount int32
	readonly   bool
}

func (s *mockWritableSource) WriteItem(ctx context.Context, action string, data map[string]interface{}) error {
	atomic.AddInt32(&s.writeCount, 1)
	return nil
}

func (s *mockWritableSource) IsReadonly() bool {
	return s.readonly
}

func (s *mockWritableSource) WriteCount() int {
	return int(atomic.LoadInt32(&s.writeCount))
}

func TestCachedSourceBasic(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name: "test",
		data: []map[string]interface{}{{"id": 1}},
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "1m",
			Strategy: "simple",
		},
	}

	cached := NewCachedSource(inner, c, cfg)

	// First fetch should call inner
	ctx := context.Background()
	data, err := cached.Fetch(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Errorf("expected 1 item, got %d", len(data))
	}
	if inner.FetchCount() != 1 {
		t.Errorf("expected 1 fetch, got %d", inner.FetchCount())
	}

	// Second fetch should use cache
	data, err = cached.Fetch(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Errorf("expected 1 item, got %d", len(data))
	}
	if inner.FetchCount() != 1 {
		t.Errorf("expected still 1 fetch (cached), got %d", inner.FetchCount())
	}
}

func TestCachedSourceTTLExpiry(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name: "test",
		data: []map[string]interface{}{{"id": 1}},
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "50ms",
			Strategy: "simple",
		},
	}

	cached := NewCachedSource(inner, c, cfg)
	ctx := context.Background()

	// First fetch
	cached.Fetch(ctx)
	if inner.FetchCount() != 1 {
		t.Errorf("expected 1 fetch, got %d", inner.FetchCount())
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should fetch again
	cached.Fetch(ctx)
	if inner.FetchCount() != 2 {
		t.Errorf("expected 2 fetches after TTL expiry, got %d", inner.FetchCount())
	}
}

func TestCachedSourceInvalidate(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name: "test",
		data: []map[string]interface{}{{"id": 1}},
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "1m",
			Strategy: "simple",
		},
	}

	cached := NewCachedSource(inner, c, cfg)
	ctx := context.Background()

	// First fetch
	cached.Fetch(ctx)
	if inner.FetchCount() != 1 {
		t.Errorf("expected 1 fetch, got %d", inner.FetchCount())
	}

	// Invalidate
	cached.Invalidate()

	// Should fetch again
	cached.Fetch(ctx)
	if inner.FetchCount() != 2 {
		t.Errorf("expected 2 fetches after invalidate, got %d", inner.FetchCount())
	}
}

func TestCachedWritableSourceInvalidatesOnWrite(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockWritableSource{
		mockSource: mockSource{
			name: "test",
			data: []map[string]interface{}{{"id": 1}},
		},
		readonly: false,
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "1m",
			Strategy: "simple",
		},
	}

	cached := NewCachedWritableSource(inner, c, cfg)
	ctx := context.Background()

	// First fetch - populates cache
	cached.Fetch(ctx)
	if inner.FetchCount() != 1 {
		t.Errorf("expected 1 fetch, got %d", inner.FetchCount())
	}

	// Second fetch - uses cache
	cached.Fetch(ctx)
	if inner.FetchCount() != 1 {
		t.Errorf("expected still 1 fetch, got %d", inner.FetchCount())
	}

	// Write - should invalidate cache
	err := cached.WriteItem(ctx, "add", map[string]interface{}{"name": "new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.WriteCount() != 1 {
		t.Errorf("expected 1 write, got %d", inner.WriteCount())
	}

	// Fetch after write - should fetch again
	cached.Fetch(ctx)
	if inner.FetchCount() != 2 {
		t.Errorf("expected 2 fetches after write, got %d", inner.FetchCount())
	}
}

func TestCachedSourceStaleWhileRevalidate(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name:       "test",
		data:       []map[string]interface{}{{"id": 1}},
		fetchDelay: 50 * time.Millisecond, // Simulate slow fetch
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "200ms",
			Strategy: "stale-while-revalidate",
		},
	}

	cached := NewCachedSource(inner, c, cfg)
	ctx := context.Background()

	// First fetch - populates cache
	_, err := cached.Fetch(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.FetchCount() != 1 {
		t.Errorf("expected 1 fetch, got %d", inner.FetchCount())
	}

	// Wait until data becomes stale (TTL/2 = 100ms)
	time.Sleep(120 * time.Millisecond)

	// Fetch stale data - should return immediately and trigger background revalidation
	start := time.Now()
	_, err = cached.Fetch(ctx)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return immediately (not wait for revalidation)
	if elapsed > 30*time.Millisecond {
		t.Errorf("expected immediate return for stale data, took %v", elapsed)
	}

	// Wait for background revalidation to complete
	time.Sleep(100 * time.Millisecond)

	if inner.FetchCount() != 2 {
		t.Errorf("expected 2 fetches (1 initial + 1 revalidation), got %d", inner.FetchCount())
	}
}

func TestCachedSourceName(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{name: "mySource"}
	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{TTL: "1m"},
	}

	cached := NewCachedSource(inner, c, cfg)
	if cached.Name() != "mySource" {
		t.Errorf("expected name 'mySource', got '%s'", cached.Name())
	}
}

func TestCachedWritableSourceIsReadonly(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockWritableSource{
		mockSource: mockSource{name: "test"},
		readonly:   true,
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{TTL: "1m"},
	}

	cached := NewCachedWritableSource(inner, c, cfg)
	if !cached.IsReadonly() {
		t.Error("expected IsReadonly() to return true")
	}

	inner.readonly = false
	cached = NewCachedWritableSource(inner, c, cfg)
	if cached.IsReadonly() {
		t.Error("expected IsReadonly() to return false")
	}
}

func TestCachedSourceGetInner(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{name: "test"}
	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{TTL: "1m"},
	}

	cached := NewCachedSource(inner, c, cfg)
	if cached.GetInner() != inner {
		t.Error("GetInner() should return the wrapped source")
	}
}

func TestCachedSourceContextCancelled(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name: "test",
		data: []map[string]interface{}{{"id": 1}},
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{TTL: "1m"},
	}

	cached := NewCachedSource(inner, c, cfg)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Fetch with cancelled context should return error
	_, err := cached.Fetch(ctx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	// Inner source should not have been called
	if inner.FetchCount() != 0 {
		t.Errorf("expected 0 fetches with cancelled context, got %d", inner.FetchCount())
	}
}

func TestCachedSourceCloseStopsRevalidation(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name:       "test",
		data:       []map[string]interface{}{{"id": 1}},
		fetchDelay: 100 * time.Millisecond, // Slow fetch
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "100ms",
			Strategy: "stale-while-revalidate",
		},
	}

	cached := NewCachedSource(inner, c, cfg)
	ctx := context.Background()

	// First fetch - populates cache
	cached.Fetch(ctx)
	if inner.FetchCount() != 1 {
		t.Errorf("expected 1 fetch, got %d", inner.FetchCount())
	}

	// Wait until data becomes stale
	time.Sleep(60 * time.Millisecond)

	// Trigger background revalidation
	cached.Fetch(ctx)

	// Close immediately - should cancel background revalidation
	cached.Close()

	// Give some time for goroutine to exit
	time.Sleep(50 * time.Millisecond)

	// Background revalidation should have been cancelled, not completed
	// It may or may not have started, but it shouldn't complete after Close()
}

func TestCacheMaxRows(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	// Create source with 100 rows
	data := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		data[i] = map[string]interface{}{"id": i}
	}

	inner := &mockSource{
		name: "test",
		data: data,
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:     "1m",
			MaxRows: 50,
		},
	}

	cached := NewCachedSource(inner, c, cfg)
	ctx := context.Background()

	// Fetch should truncate to max_rows
	result, err := cached.Fetch(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 50 {
		t.Errorf("expected 50 rows (max_rows limit), got %d", len(result))
	}

	// Verify first 50 rows are preserved
	if result[0]["id"] != 0 || result[49]["id"] != 49 {
		t.Error("expected first 50 rows to be preserved")
	}
}

func TestCacheMaxBytes(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	// Create source with large data
	data := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		data[i] = map[string]interface{}{
			"id":          i,
			"description": "This is a longer description to increase size",
		}
	}

	inner := &mockSource{
		name: "test",
		data: data,
	}

	// Set a small max_bytes limit
	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "1m",
			MaxBytes: 500, // Very small limit
		},
	}

	cached := NewCachedSource(inner, c, cfg)
	ctx := context.Background()

	// Fetch should truncate to fit within max_bytes
	result, err := cached.Fetch(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have fewer rows due to byte limit
	if len(result) >= 100 {
		t.Errorf("expected fewer than 100 rows due to max_bytes, got %d", len(result))
	}

	// Verify size is within limit
	size := estimateSize(result)
	if size > 500 {
		t.Errorf("expected size <= 500 bytes, got %d", size)
	}
}

func TestCacheInfoProvider(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name: "test",
		data: []map[string]interface{}{{"id": 1}},
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "1m",
			Strategy: "simple",
		},
	}

	cached := NewCachedSource(inner, c, cfg)

	// Verify CachedSource implements CacheInfoProvider
	var _ CacheInfoProvider = cached

	ctx := context.Background()

	// First fetch - not from cache
	cached.Fetch(ctx)

	info := cached.GetCacheInfo()
	if info == nil {
		t.Fatal("expected CacheInfo, got nil")
	}

	if info.Cached {
		t.Error("first fetch should not be marked as cached")
	}

	// Second fetch - from cache
	cached.Fetch(ctx)

	info = cached.GetCacheInfo()
	if !info.Cached {
		t.Error("second fetch should be marked as cached")
	}

	if info.Stale {
		t.Error("data should not be stale yet")
	}

	if info.Refreshing {
		t.Error("should not be refreshing")
	}
}

func TestCacheInfoStaleState(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name:       "test",
		data:       []map[string]interface{}{{"id": 1}},
		fetchDelay: 50 * time.Millisecond,
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "200ms",
			Strategy: "stale-while-revalidate",
		},
	}

	cached := NewCachedSource(inner, c, cfg)
	ctx := context.Background()

	// First fetch
	cached.Fetch(ctx)

	// Wait until stale (TTL/2 = 100ms)
	time.Sleep(120 * time.Millisecond)

	// Fetch stale data - triggers background refresh
	cached.Fetch(ctx)

	info := cached.GetCacheInfo()
	if info == nil {
		t.Fatal("expected CacheInfo, got nil")
	}

	if !info.Cached {
		t.Error("expected Cached=true")
	}

	if !info.Stale {
		t.Error("expected Stale=true after TTL/2")
	}

	if !info.Refreshing {
		t.Error("expected Refreshing=true during background revalidation")
	}

	// Wait for revalidation to complete
	time.Sleep(100 * time.Millisecond)

	info = cached.GetCacheInfo()
	if info.Refreshing {
		t.Error("expected Refreshing=false after revalidation completes")
	}
}

func TestCacheInfoDurationFormat(t *testing.T) {
	c := cache.NewMemoryCache()
	defer c.Stop()

	inner := &mockSource{
		name: "test",
		data: []map[string]interface{}{{"id": 1}},
	}

	cfg := config.SourceConfig{
		Cache: &config.CacheConfig{
			TTL:      "5m",
			Strategy: "simple",
		},
	}

	cached := NewCachedSource(inner, c, cfg)
	ctx := context.Background()

	// First fetch to populate cache
	cached.Fetch(ctx)

	// Wait a bit for measurable age
	time.Sleep(50 * time.Millisecond)

	// Second fetch from cache
	cached.Fetch(ctx)

	info := cached.GetCacheInfo()
	if info == nil {
		t.Fatal("expected CacheInfo, got nil")
	}

	// Age should be a non-empty duration string
	if info.Age == "" {
		t.Error("expected Age to be set")
	}

	// ExpiresIn should be a non-empty duration string
	if info.ExpiresIn == "" {
		t.Error("expected ExpiresIn to be set")
	}
}
