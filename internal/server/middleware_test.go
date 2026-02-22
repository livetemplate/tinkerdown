package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
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
	mw, done := RateLimitMiddleware(ctx, rps, burst, maxIPs)
	t.Cleanup(func() {
		cancel()
		<-done
	})
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

// rateLimitWrapInternal creates a rate-limited handler using the internal
// constructor with configurable durations, for testing cleanup and logging.
func rateLimitWrapInternal(t *testing.T, rps float64, burst, maxIPs int,
	sweepInterval, staleThreshold, evictionLogThreshold time.Duration,
	next http.Handler,
) http.Handler {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	mw, done := rateLimitMiddlewareInternal(ctx, rps, burst, maxIPs,
		sweepInterval, staleThreshold, evictionLogThreshold)
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return mw(next)
}

// countLogLines returns the number of lines in output that contain substr.
func countLogLines(output, substr string) int {
	n := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, substr) {
			n++
		}
	}
	return n
}

// TestRateLimitCleanupRemovesStaleEntries verifies that the cleanup goroutine
// removes IP entries that haven't been seen for longer than the stale threshold.
func TestRateLimitCleanupRemovesStaleEntries(t *testing.T) {
	// rps=0.001 ensures the token bucket won't refill naturally (1 token/1000s),
	// so a 200 after sleeping proves the cleanup created a fresh limiter.
	wrapped := rateLimitWrapInternal(t,
		0.001, 1, 100,
		50*time.Millisecond,  // sweepInterval
		100*time.Millisecond, // staleThreshold
		time.Hour,            // evictLogInterval (irrelevant here)
		okHandler(),
	)

	// First request uses the burst token → 200
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("5.5.5.5"))
	if w.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w.Code)
	}

	// Second request from same IP → 429 (burst exhausted, rps too low to refill)
	w = httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("5.5.5.5"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", w.Code)
	}

	// Wait for staleThreshold + sweepInterval + margin so the cleanup fires
	time.Sleep(250 * time.Millisecond)

	// Entry should be gone — new request gets a fresh limiter with burst token → 200
	w = httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("5.5.5.5"))
	if w.Code != http.StatusOK {
		t.Errorf("after cleanup: expected 200 (fresh limiter), got %d", w.Code)
	}
}

// TestRateLimitCleanupKeepsActiveEntries verifies that entries accessed
// recently (within the stale threshold) survive the cleanup goroutine.
func TestRateLimitCleanupKeepsActiveEntries(t *testing.T) {
	// burst=1, rps=0.001 — once burst is used, only a fresh limiter gives 200
	wrapped := rateLimitWrapInternal(t,
		0.001, 1, 100,
		50*time.Millisecond,  // sweepInterval
		150*time.Millisecond, // staleThreshold
		time.Hour,            // evictLogInterval (irrelevant here)
		okHandler(),
	)

	// Use burst token → 200
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, reqFromIP("6.6.6.6"))
	if w.Code != http.StatusOK {
		t.Fatalf("initial request: expected 200, got %d", w.Code)
	}

	// Keep entry alive by sending requests every 60ms (< 150ms staleThreshold)
	// across multiple sweep cycles (each 50ms)
	for i := 0; i < 4; i++ {
		time.Sleep(60 * time.Millisecond)
		w = httptest.NewRecorder()
		wrapped.ServeHTTP(w, reqFromIP("6.6.6.6"))
		// Should stay 429 — same exhausted limiter, not a fresh one
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("tick %d: expected 429 (entry kept alive), got %d", i, w.Code)
		}
	}
}

// TestRateLimitEvictionLogThrottling verifies that eviction log messages
// are throttled: at most one message per evictionLogThreshold window.
// NOTE: log.SetOutput mutates global state; this test cannot use t.Parallel().
func TestRateLimitEvictionLogThrottling(t *testing.T) {
	var buf bytes.Buffer
	origWriter := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origWriter) })

	wrapped := rateLimitWrapInternal(t,
		100, 100, 1,
		time.Hour,             // sweepInterval (irrelevant here)
		time.Hour,             // staleThreshold (irrelevant here)
		100*time.Millisecond,  // evictionLogThreshold
		okHandler(),
	)

	// 6 requests: 1st populates the slot, next 5 each evict the previous IP
	for i := 0; i < 6; i++ {
		ip := fmt.Sprintf("20.0.0.%d", i)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, reqFromIP(ip))
		if w.Code != http.StatusOK {
			t.Fatalf("IP %s: expected 200, got %d", ip, w.Code)
		}
	}

	// First batch: only 1 log line (first eviction triggers it, rest are throttled)
	lines := countLogLines(buf.String(), "[RateLimit] Evicted")
	if lines != 1 {
		t.Errorf("first batch: expected 1 log line, got %d\nlog output:\n%s", lines, buf.String())
	}

	// Wait for throttle window to expire
	time.Sleep(150 * time.Millisecond)

	// Trigger more evictions
	for i := 10; i < 16; i++ {
		ip := fmt.Sprintf("20.0.0.%d", i)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, reqFromIP(ip))
		if w.Code != http.StatusOK {
			t.Fatalf("IP %s: expected 200, got %d", ip, w.Code)
		}
	}

	// Second batch: should produce 1 more log line
	lines = countLogLines(buf.String(), "[RateLimit] Evicted")
	if lines != 2 {
		t.Errorf("second batch: expected 2 total log lines, got %d\nlog output:\n%s", lines, buf.String())
	}
}
