package source

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/livetemplate/tinkerdown/internal/cache"
	"github.com/livetemplate/tinkerdown/internal/config"
)

// CacheInfoProvider is implemented by sources that expose cache metadata
type CacheInfoProvider interface {
	GetCacheInfo() *cache.CacheInfo
}

// CachedSource wraps a Source with caching behavior
type CachedSource struct {
	inner    Source
	cache    cache.Cache
	name     string
	ttl      time.Duration
	strategy string // "simple" or "stale-while-revalidate"

	// Limits for cached data
	maxRows  int
	maxBytes int

	// For stale-while-revalidate: track in-flight revalidations
	mu           sync.Mutex
	revalidating bool

	// State tracking for CacheInfo
	lastFromCache bool
	lastAge       time.Duration
	lastExpiresIn time.Duration
	lastStale     bool
	lastCachedAt  time.Time

	// For cancellation of background operations
	cancelCtx    context.Context
	cancelFunc   context.CancelFunc
}

// NewCachedSource creates a new cached source wrapper
func NewCachedSource(inner Source, c cache.Cache, cfg config.SourceConfig) *CachedSource {
	ctx, cancel := context.WithCancel(context.Background())
	return &CachedSource{
		inner:      inner,
		cache:      c,
		name:       inner.Name(),
		ttl:        cfg.GetCacheTTL(),
		strategy:   cfg.GetCacheStrategy(),
		maxRows:    cfg.GetCacheMaxRows(),
		maxBytes:   cfg.GetCacheMaxBytes(),
		cancelCtx:  ctx,
		cancelFunc: cancel,
	}
}

// Name returns the source name
func (s *CachedSource) Name() string {
	return s.name
}

// Fetch retrieves data, using cache if available
func (s *CachedSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cacheKey := s.cacheKey()

	// Try to get from cache
	data, found, stale := s.cache.Get(cacheKey)
	if found {
		// Update cache state for CacheInfo (holding lock for entire block)
		s.mu.Lock()
		s.lastFromCache = true
		s.lastStale = stale
		if !s.lastCachedAt.IsZero() {
			s.lastAge = time.Since(s.lastCachedAt)
			s.lastExpiresIn = s.ttl - s.lastAge
			if s.lastExpiresIn < 0 {
				s.lastExpiresIn = 0
			}
		}

		// Check if we need to trigger background revalidation
		shouldRevalidate := stale && s.strategy == "stale-while-revalidate" && !s.revalidating
		if shouldRevalidate {
			// Set revalidating flag BEFORE launching goroutine to avoid race condition
			// where GetCacheInfo() is called before the goroutine starts
			s.revalidating = true
		}
		s.mu.Unlock()

		if shouldRevalidate {
			go s.revalidateInBackground()
		}
		return data, nil
	}

	// Cache miss - fetch fresh data
	return s.fetchAndCache(ctx)
}

// fetchAndCache fetches from the underlying source and caches the result
func (s *CachedSource) fetchAndCache(ctx context.Context) ([]map[string]interface{}, error) {
	data, err := s.inner.Fetch(ctx)
	if err != nil {
		return nil, err
	}

	// Apply max_rows limit
	if s.maxRows > 0 && len(data) > s.maxRows {
		data = data[:s.maxRows]
	}

	// Apply max_bytes limit
	if s.maxBytes > 0 {
		data = truncateToMaxBytes(data, s.maxBytes)
	}

	// Update cache state
	s.mu.Lock()
	s.lastFromCache = false
	s.lastCachedAt = time.Now()
	s.lastAge = 0
	s.lastExpiresIn = s.ttl
	s.lastStale = false
	s.mu.Unlock()

	if s.strategy == "stale-while-revalidate" {
		// For SWR: data is fresh for half the TTL, then stale for the other half
		staleAfter := s.ttl / 2
		s.cache.SetWithStale(s.cacheKey(), data, staleAfter, s.ttl)
	} else {
		s.cache.Set(s.cacheKey(), data, s.ttl)
	}

	return data, nil
}

// revalidateInBackground fetches fresh data in the background
// Note: revalidating flag is set to true by caller (Fetch) before launching this goroutine
func (s *CachedSource) revalidateInBackground() {
	defer func() {
		s.mu.Lock()
		s.revalidating = false
		s.mu.Unlock()
	}()

	// Use cancelCtx as parent so revalidation stops when source is closed
	ctx, cancel := context.WithTimeout(s.cancelCtx, 30*time.Second)
	defer cancel()

	_, err := s.fetchAndCache(ctx)
	if err != nil {
		// Don't log if cancelled due to shutdown
		if s.cancelCtx.Err() == nil {
			log.Printf("[cache/%s] Background revalidation failed: %v", s.name, err)
		}
	}
}

// cacheKey returns the cache key for this source
func (s *CachedSource) cacheKey() string {
	return "source:" + s.name
}

// Close closes the underlying source and cancels any background operations
func (s *CachedSource) Close() error {
	// Cancel any in-flight background revalidations
	s.cancelFunc()
	return s.inner.Close()
}

// Invalidate removes this source's data from cache
func (s *CachedSource) Invalidate() {
	s.cache.Invalidate(s.cacheKey())
}

// GetCacheInfo returns cache metadata for UI display
// Implements CacheInfoProvider interface
func (s *CachedSource) GetCacheInfo() *cache.CacheInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Format durations as human-readable strings
	age := ""
	expiresIn := ""
	if s.lastAge > 0 {
		age = s.lastAge.Round(time.Second).String()
	}
	if s.lastExpiresIn > 0 {
		expiresIn = s.lastExpiresIn.Round(time.Second).String()
	}

	return &cache.CacheInfo{
		Cached:     s.lastFromCache,
		Age:        age,
		ExpiresIn:  expiresIn,
		Stale:      s.lastStale,
		Refreshing: s.revalidating,
	}
}

// GetInner returns the underlying source
// This is useful for accessing WritableSource methods
func (s *CachedSource) GetInner() Source {
	return s.inner
}

// CachedWritableSource wraps a WritableSource with caching behavior
// It invalidates the cache on write operations
type CachedWritableSource struct {
	*CachedSource
	writable WritableSource
}

// NewCachedWritableSource creates a new cached writable source wrapper
func NewCachedWritableSource(inner WritableSource, c cache.Cache, cfg config.SourceConfig) *CachedWritableSource {
	return &CachedWritableSource{
		CachedSource: NewCachedSource(inner, c, cfg),
		writable:     inner,
	}
}

// WriteItem performs a write and invalidates the cache
func (s *CachedWritableSource) WriteItem(ctx context.Context, action string, data map[string]interface{}) error {
	err := s.writable.WriteItem(ctx, action, data)
	if err != nil {
		return err
	}

	// Invalidate cache after successful write
	s.Invalidate()
	return nil
}

// IsReadonly returns whether the source is in read-only mode
func (s *CachedWritableSource) IsReadonly() bool {
	return s.writable.IsReadonly()
}

// truncateToMaxBytes removes rows from the end until data fits within maxBytes
func truncateToMaxBytes(data []map[string]interface{}, maxBytes int) []map[string]interface{} {
	for len(data) > 0 {
		size := estimateSize(data)
		if size <= maxBytes {
			return data
		}
		// Remove last row
		data = data[:len(data)-1]
	}
	return data
}

// estimateSize returns the approximate JSON-serialized size of the data in bytes
func estimateSize(data []map[string]interface{}) int {
	b, err := json.Marshal(data)
	if err != nil {
		return 0
	}
	return len(b)
}
