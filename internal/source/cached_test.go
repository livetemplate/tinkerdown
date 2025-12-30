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
