package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func reqFromIP(ip string) *http.Request {
	r := httptest.NewRequest("GET", "/api/sources/test", nil)
	r.RemoteAddr = ip + ":12345"
	return r
}

// rateLimitWrap creates a rate-limited handler with a context that is
// cancelled when the test finishes, preventing goroutine leaks.
func rateLimitWrap(t *testing.T, rps float64, burst, maxIPs int, next http.Handler) http.Handler {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	mw, _ := RateLimitMiddleware(ctx, rps, burst, maxIPs)
	return mw(next)
}

// TestRateLimitLRUEviction verifies that when the IP map is full, a new IP
// evicts the least-recently-used entry instead of returning 503.
func TestRateLimitLRUEviction(t *testing.T) {
	wrapped := rateLimitWrap(t, 100, 100, 3, okHandler())

	// Fill to capacity with 3 IPs
	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, reqFromIP(ip))
		if w.Code != http.StatusOK {
			t.Fatalf("IP %s: expected 200, got %d", ip, w.Code)
		}
	}

	// 4th IP should succeed (LRU eviction), not 503
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("4.4.4.4"))
	if w.Code != http.StatusOK {
		t.Errorf("4th IP at capacity: expected 200, got %d", w.Code)
	}
}

// TestRateLimitEvictedIPGetsFreshLimiter verifies that an evicted IP returning
// gets a fresh token bucket, not a stale one.
func TestRateLimitEvictedIPGetsFreshLimiter(t *testing.T) {
	// burst=1 so the first request consumes the token
	wrapped := rateLimitWrap(t, 100, 1, 2, okHandler())

	// IP "1.1.1.1" uses its burst token
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("1.1.1.1"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Second request from same IP should be rate-limited (burst exhausted)
	w = httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("1.1.1.1"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	// Push 1.1.1.1 out by filling capacity with 2 other IPs
	for _, ip := range []string{"2.2.2.2", "3.3.3.3"} {
		w = httptest.NewRecorder()
		wrapped.ServeHTTP(w, reqFromIP(ip))
		if w.Code != http.StatusOK {
			t.Fatalf("IP %s: expected 200, got %d", ip, w.Code)
		}
	}

	// 1.1.1.1 returns — should get a fresh limiter with a full burst token
	w = httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("1.1.1.1"))
	if w.Code != http.StatusOK {
		t.Errorf("evicted IP returning: expected 200 (fresh limiter), got %d", w.Code)
	}
}

// TestRateLimitMRUNotEvicted verifies that accessing an IP moves it to the
// front of the LRU, protecting it from eviction.
func TestRateLimitMRUNotEvicted(t *testing.T) {
	wrapped := rateLimitWrap(t, 100, 100, 3, okHandler())

	// Fill: A, B, C (order: C=front, B, A=back)
	for _, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, reqFromIP(ip))
		if w.Code != http.StatusOK {
			t.Fatalf("IP %s: expected 200, got %d", ip, w.Code)
		}
	}

	// Touch A → moves to front (order: A=front, C, B=back)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("10.0.0.1"))
	if w.Code != http.StatusOK {
		t.Fatalf("touch A: expected 200, got %d", w.Code)
	}

	// New IP D → evicts B (back), not A
	w = httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("10.0.0.4"))
	if w.Code != http.StatusOK {
		t.Fatalf("new IP D: expected 200, got %d", w.Code)
	}

	// A should still be present (not evicted)
	w = httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("10.0.0.1"))
	if w.Code != http.StatusOK {
		t.Errorf("A after eviction: expected 200 (still present), got %d", w.Code)
	}
}

// TestRateLimitNo503AtCapacity is a regression test ensuring that 503 is never
// returned when the rate limiter is at capacity.
func TestRateLimitNo503AtCapacity(t *testing.T) {
	wrapped := rateLimitWrap(t, 100, 100, 5, okHandler())

	// Send requests from 20 unique IPs — all should get 200, never 503
	for i := 0; i < 20; i++ {
		ip := fmt.Sprintf("192.168.%d.%d", i/256, i%256)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, reqFromIP(ip))
		if w.Code == http.StatusServiceUnavailable {
			t.Fatalf("IP %s: got 503 (should never happen with LRU eviction)", ip)
		}
		if w.Code != http.StatusOK {
			t.Errorf("IP %s: expected 200, got %d", ip, w.Code)
		}
	}
}

// TestRateLimitConcurrentAccess verifies no races or panics under concurrent load.
func TestRateLimitConcurrentAccess(t *testing.T) {
	wrapped := rateLimitWrap(t, 1000, 1000, 100, okHandler())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.0.%d.%d", id/256, id%256)
			for j := 0; j < 10; j++ {
				w := httptest.NewRecorder()
				wrapped.ServeHTTP(w, reqFromIP(ip))
				if w.Code == http.StatusServiceUnavailable {
					t.Errorf("IP %s: got 503 under concurrent load", ip)
				}
			}
		}(i)
	}
	wg.Wait()
}

// TestGetMaxTrackedIPs tests the config accessor with defaults and explicit values.
func TestGetMaxTrackedIPs(t *testing.T) {
	// nil APIConfig → default
	var nilCfg *config.APIConfig
	if got := nilCfg.GetMaxTrackedIPs(); got != 10000 {
		t.Errorf("nil APIConfig: expected 10000, got %d", got)
	}

	// nil RateLimit → default
	cfg := &config.APIConfig{}
	if got := cfg.GetMaxTrackedIPs(); got != 10000 {
		t.Errorf("nil RateLimit: expected 10000, got %d", got)
	}

	// Zero value → default
	cfg = &config.APIConfig{RateLimit: &config.RateLimitConfig{MaxTrackedIPs: 0}}
	if got := cfg.GetMaxTrackedIPs(); got != 10000 {
		t.Errorf("zero MaxTrackedIPs: expected 10000, got %d", got)
	}

	// Explicit value
	cfg = &config.APIConfig{RateLimit: &config.RateLimitConfig{MaxTrackedIPs: 500}}
	if got := cfg.GetMaxTrackedIPs(); got != 500 {
		t.Errorf("explicit MaxTrackedIPs: expected 500, got %d", got)
	}
}

// TestRateLimitCleanupStopsOnCancel verifies that cancelling the context
// causes the cleanup goroutine to exit, confirmed via the done channel.
func TestRateLimitCleanupStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, done := RateLimitMiddleware(ctx, 100, 100, 100)

	cancel()

	select {
	case <-done:
		// Goroutine exited cleanly.
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup goroutine did not exit within 2s")
	}
}
