# Source Caching Enhancements Design

**Issue:** #39 - Source Caching
**Date:** 2026-01-04
**Status:** Approved

## Overview

Enhance the existing source caching system with row/byte limits and UI indicators for stale data. The core caching (TTL, stale-while-revalidate, thundering herd prevention) is already implemented.

## What's Already Implemented

- `internal/cache/cache.go` - MemoryCache with TTL and stale detection
- `internal/source/cached.go` - CachedSource with stale-while-revalidate
- `internal/config/config.go` - CacheConfig with TTL and Strategy

## What's Missing

1. `max_rows` and `max_bytes` limits
2. CacheInfo exposure to templates
3. UI stale/refresh indicator

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Limit behavior | Truncate silently | Simple, predictable; matches issue spec |
| CacheInfo exposure | Interface-based detection | Non-breaking, follows Go patterns |
| Duration format | Human-readable strings | Template-friendly ("2m30s") |
| UI indicator | CSS data-attributes | Minimal JS, works with hot-reload |

## Implementation

### 1. Configuration Changes

**File:** `internal/config/config.go`

```go
type CacheConfig struct {
    TTL      string `yaml:"ttl,omitempty"`       // Cache TTL (e.g., "5m", "1h")
    Strategy string `yaml:"strategy,omitempty"`  // "simple" or "stale-while-revalidate"
    MaxRows  int    `yaml:"max_rows,omitempty"`  // Limit cached rows (truncates if exceeded)
    MaxBytes int    `yaml:"max_bytes,omitempty"` // Limit cached size in bytes (truncates if exceeded)
}

func (c SourceConfig) GetCacheMaxRows() int {
    if c.Cache == nil { return 0 }
    return c.Cache.MaxRows
}

func (c SourceConfig) GetCacheMaxBytes() int {
    if c.Cache == nil { return 0 }
    return c.Cache.MaxBytes
}
```

### 2. CacheInfo Struct

**File:** `internal/cache/cache.go`

```go
// CacheInfo provides cache metadata for UI display
type CacheInfo struct {
    Cached     bool   `json:"cached"`      // Whether data came from cache
    Age        string `json:"age"`         // How old the cached data is (e.g., "2m30s")
    ExpiresIn  string `json:"expires_in"`  // Time until expiry (e.g., "4m15s")
    Stale      bool   `json:"stale"`       // Whether data is stale (but still usable)
    Refreshing bool   `json:"refreshing"`  // Whether background refresh is in progress
}
```

### 3. CacheInfoProvider Interface

**File:** `internal/source/cached.go`

```go
// CacheInfoProvider is implemented by sources that expose cache metadata
type CacheInfoProvider interface {
    GetCacheInfo() *cache.CacheInfo
}

type CachedSource struct {
    // ... existing fields ...

    // Config for limits
    maxRows  int
    maxBytes int

    // State tracking for CacheInfo
    lastFromCache  bool
    lastAge        time.Duration
    lastExpiresIn  time.Duration
    lastStale      bool
}

func (s *CachedSource) GetCacheInfo() *cache.CacheInfo {
    s.mu.Lock()
    defer s.mu.Unlock()

    return &cache.CacheInfo{
        Cached:     s.lastFromCache,
        Age:        s.lastAge.Round(time.Second).String(),
        ExpiresIn:  s.lastExpiresIn.Round(time.Second).String(),
        Stale:      s.lastStale,
        Refreshing: s.revalidating,
    }
}
```

### 4. Limit Enforcement

**File:** `internal/source/cached.go`

```go
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

    // Cache the (possibly truncated) data
    // ... existing cache logic ...

    return data, nil
}

func truncateToMaxBytes(data []map[string]interface{}, maxBytes int) []map[string]interface{} {
    for len(data) > 0 {
        size := estimateSize(data)
        if size <= maxBytes {
            return data
        }
        data = data[:len(data)-1]
    }
    return data
}

func estimateSize(data []map[string]interface{}) int {
    // JSON marshal to estimate size
    b, _ := json.Marshal(data)
    return len(b)
}
```

### 5. GenericState Integration

**File:** `internal/runtime/state.go`

```go
type GenericState struct {
    // ... existing fields ...

    // Cache metadata for UI display
    CacheInfo *cache.CacheInfo `json:"cache_info,omitempty"`
}

func (s *GenericState) refresh() error {
    // ... existing fetch logic ...

    // Populate CacheInfo if source supports it
    if provider, ok := s.source.(source.CacheInfoProvider); ok {
        s.CacheInfo = provider.GetCacheInfo()
    }

    // ... rest of method ...
}
```

### 6. UI Indicator

**HTML attributes added during render:**
```html
<div lvt-state="users" data-cache-stale="true" data-cache-refreshing="true">
  <!-- content -->
</div>
```

**CSS in client styles:**
```css
[data-cache-stale="true"] {
    position: relative;
}
[data-cache-stale="true"]::before {
    content: "⟳ Refreshing...";
    position: absolute;
    top: 0; right: 0;
    font-size: 0.75rem;
    color: #666;
    background: #f0f0f0;
    padding: 2px 6px;
    border-radius: 3px;
}
[data-cache-refreshing="true"]::before {
    animation: pulse 1s infinite;
}
@keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
}
```

## Example Usage

```yaml
sources:
  # Simple TTL cache
  users:
    type: pg
    query: SELECT * FROM users
    cache:
      ttl: 5m

  # Stale-while-revalidate for better UX
  products:
    type: rest
    from: https://api.example.com/products
    cache:
      ttl: 1m
      strategy: stale-while-revalidate

  # Limit cached data
  logs:
    type: pg
    query: SELECT * FROM logs ORDER BY created_at DESC
    cache:
      ttl: 30s
      max_rows: 1000
      max_bytes: 1048576  # 1MB
```

Template usage:
```html
{{if .CacheInfo}}
  {{if .CacheInfo.Stale}}
    <span class="cache-stale">Data is {{.CacheInfo.Age}} old</span>
  {{else if .CacheInfo.Cached}}
    <span class="cache-fresh">Refreshes in {{.CacheInfo.ExpiresIn}}</span>
  {{end}}
{{end}}
```

## Testing

1. **TestCacheMaxRows** - Verify row truncation
2. **TestCacheMaxBytes** - Verify byte limit truncation
3. **TestCacheInfoProvider** - Verify interface implementation
4. **TestCacheInfoStaleState** - Verify stale/refreshing states
5. **TestCacheInfoNotCached** - Verify nil CacheInfo for uncached sources
6. **TestCacheAttributesInHTML** - Verify data-* attributes in output

## Files to Modify

- `internal/config/config.go` - Add MaxRows, MaxBytes to CacheConfig
- `internal/cache/cache.go` - Add CacheInfo struct
- `internal/source/cached.go` - Add CacheInfoProvider, limit enforcement
- `internal/runtime/state.go` - Add CacheInfo field, populate in refresh()
- `internal/server/websocket.go` - Add data-cache-* attributes to HTML
- `internal/assets/client/styles.css` - Add cache indicator styles
