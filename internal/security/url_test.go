package security

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestValidateHTTPURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		// Valid URLs
		{name: "valid https url", url: "https://api.example.com/endpoint", wantErr: ""},
		{name: "valid http url", url: "http://api.example.com/endpoint", wantErr: ""},
		// Invalid schemes
		{name: "file scheme", url: "file:///etc/passwd", wantErr: "URL scheme must be http or https"},
		{name: "ftp scheme", url: "ftp://files.example.com/file.txt", wantErr: "URL scheme must be http or https"},
		// Localhost
		{name: "localhost", url: "http://localhost/admin", wantErr: "requests to localhost are not allowed"},
		{name: "localhost with port", url: "http://localhost:8080/admin", wantErr: "requests to localhost are not allowed"},
		// Loopback IPs
		{name: "127.0.0.1", url: "http://127.0.0.1/admin", wantErr: "requests to loopback addresses are not allowed"},
		{name: "127.0.0.1 with port", url: "http://127.0.0.1:8080/admin", wantErr: "requests to loopback addresses are not allowed"},
		{name: "ipv6 loopback", url: "http://[::1]/admin", wantErr: "requests to loopback addresses are not allowed"},
		// Private networks
		{name: "10.x.x.x", url: "http://10.0.0.1/admin", wantErr: "requests to private network addresses are not allowed"},
		{name: "172.16.x.x", url: "http://172.16.0.1/admin", wantErr: "requests to private network addresses are not allowed"},
		{name: "192.168.x.x", url: "http://192.168.1.1/admin", wantErr: "requests to private network addresses are not allowed"},
		// Link-local
		{name: "link-local metadata endpoint", url: "http://169.254.169.254/latest/meta-data", wantErr: "requests to link-local addresses are not allowed"},
		// Unspecified
		{name: "0.0.0.0", url: "http://0.0.0.0/admin", wantErr: "requests to unspecified addresses are not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetDNSCache()
			err := ValidateHTTPURL(tt.url)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateHTTPURL() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateHTTPURL() expected error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateHTTPURL() error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestValidateHTTPURL_TestBypass(t *testing.T) {
	SetTestBypassSSRF(true)
	t.Cleanup(func() { SetTestBypassSSRF(false) })

	// With bypass enabled, even localhost should be allowed
	if err := ValidateHTTPURL("http://localhost/admin"); err != nil {
		t.Errorf("ValidateHTTPURL() with bypass should allow localhost, got: %v", err)
	}
}

func TestValidateHTTPURL_DNSRebinding(t *testing.T) {
	ResetDNSCache()
	// Hostnames that resolve to loopback should be blocked
	err := ValidateHTTPURL("http://localhost.localdomain/admin")
	if err == nil {
		t.Error("ValidateHTTPURL() should block localhost.localdomain")
	}
}

func TestDNSCache_HitAndExpiry(t *testing.T) {
	c := &dnsCache{
		entries: make(map[string]dnsCacheEntry),
		ttl:     1 * time.Millisecond,
		maxSize: 10,
	}

	// Cache miss on empty cache.
	if _, ok := c.get("example.com"); ok {
		t.Fatal("expected cache miss on empty cache")
	}

	// Store a blocked error and verify cache hit returns it.
	blockedErr := fmt.Errorf("blocked")
	c.set("evil.com", blockedErr)
	if err, ok := c.get("evil.com"); !ok {
		t.Fatal("expected cache hit for blocked host")
	} else if err == nil || err.Error() != "blocked" {
		t.Fatalf("expected 'blocked' error, got %v", err)
	}

	// Wait for TTL expiry (5ms >> 1ms TTL, safe even on loaded CI).
	time.Sleep(5 * time.Millisecond)
	if _, ok := c.get("evil.com"); ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestDNSCache_MaxSize(t *testing.T) {
	blocked := fmt.Errorf("blocked")
	c := &dnsCache{
		entries: make(map[string]dnsCacheEntry),
		ttl:     time.Minute,
		maxSize: 3,
	}

	// Fill to capacity.
	c.set("a.com", blocked)
	c.set("b.com", blocked)
	c.set("c.com", blocked)
	if _, ok := c.get("a.com"); !ok {
		t.Fatal("expected cache hit before overflow")
	}

	// Adding one more should trigger expired-entry pruning first; since none
	// are expired, all entries are cleared, then the new entry is stored.
	c.set("d.com", blocked)
	if _, ok := c.get("d.com"); !ok {
		t.Fatal("expected cache hit for newly added entry after eviction")
	}
	if _, ok := c.get("a.com"); ok {
		t.Fatal("expected cache miss for old entry after eviction")
	}
}

func TestDNSCache_MaxSizePrunesExpired(t *testing.T) {
	blocked := fmt.Errorf("blocked")
	c := &dnsCache{
		entries: make(map[string]dnsCacheEntry),
		ttl:     1 * time.Millisecond,
		maxSize: 3,
	}

	// Fill to capacity with entries that will expire quickly.
	c.set("a.com", blocked)
	c.set("b.com", blocked)
	c.set("c.com", blocked)

	// Wait for all entries to expire.
	time.Sleep(5 * time.Millisecond)

	// Adding a new entry should prune expired entries instead of full clear.
	c.set("d.com", blocked)

	// d.com should be present.
	if _, ok := c.get("d.com"); !ok {
		t.Fatal("expected cache hit for newly added entry")
	}

	// Cache should have only 1 entry (d.com) since expired ones were pruned.
	c.mu.Lock()
	size := len(c.entries)
	c.mu.Unlock()
	if size != 1 {
		t.Fatalf("expected 1 entry after pruning, got %d", size)
	}
}

func TestDNSCache_ConcurrentAccess(t *testing.T) {
	blocked := fmt.Errorf("blocked")
	c := &dnsCache{
		entries: make(map[string]dnsCacheEntry),
		ttl:     time.Second,
		maxSize: 100,
	}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		host := fmt.Sprintf("host-%d.example.com", i)
		go func() {
			defer wg.Done()
			c.set(host, blocked)
		}()
		go func() {
			defer wg.Done()
			c.get(host)
		}()
	}
	wg.Wait()
}
