// Package security provides shared security validation functions.
package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// testBypassSSRF is a testing hook to bypass SSRF validation.
// Use SetTestBypassSSRF to toggle it safely from tests.
var testBypassSSRF atomic.Bool

// SetTestBypassSSRF enables or disables the SSRF validation bypass for testing.
// It is safe for concurrent use.
func SetTestBypassSSRF(enabled bool) {
	testBypassSSRF.Store(enabled)
}

type dnsCacheEntry struct {
	err       error
	expiresAt time.Time
}

type dnsCache struct {
	mu      sync.RWMutex
	entries map[string]dnsCacheEntry
	ttl     time.Duration
	maxSize int
}

var hostCache = &dnsCache{
	entries: make(map[string]dnsCacheEntry),
	ttl:     30 * time.Second,
	maxSize: 1024,
}

// get returns a cached validation error for host. On a cache miss or expired
// entry it returns (nil, false). Expired entries are left in the map for lazy
// cleanup by set() to avoid upgrading to a write lock on the read path.
func (c *dnsCache) get(host string) (cachedErr error, ok bool) {
	c.mu.RLock()
	entry, found := c.entries[host]
	c.mu.RUnlock()
	if !found || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.err, true
}

func (c *dnsCache) set(host string, err error) {
	if err == nil {
		return // never cache an allowed result
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= c.maxSize {
			// Evict one arbitrary entry to make room without discarding
			// the entire cache (avoids thrashing under adversarial load).
			for k := range c.entries {
				delete(c.entries, k)
				break
			}
		}
	}
	c.entries[host] = dnsCacheEntry{
		err:       err,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// ValidateHTTPURL checks for SSRF vulnerabilities by blocking requests to internal networks.
// It rejects localhost, private IP ranges, link-local addresses, and cloud metadata endpoints.
// Hostnames are resolved to detect DNS rebinding attacks targeting internal addresses.
func ValidateHTTPURL(rawURL string) error {
	if testBypassSSRF.Load() {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL must have a host")
	}

	// Block localhost variations
	hostLower := strings.ToLower(host)
	if hostLower == "localhost" || hostLower == "localhost.localdomain" {
		return fmt.Errorf("requests to localhost are not allowed")
	}

	// Parse as IP address
	ip := net.ParseIP(host)
	if ip != nil {
		return validateIP(ip)
	}

	// Check DNS cache for a previously blocked result (keyed by lowercase hostname).
	// Only blocked (error) results are cached — allowed results are always
	// re-validated to prevent DNS rebinding attacks where an attacker changes
	// DNS records from a public IP to an internal IP after the first lookup.
	if cachedErr, ok := hostCache.get(hostLower); ok {
		return cachedErr
	}

	// Cache miss: resolve and check all resulting IPs to prevent DNS rebinding.
	// If resolution fails (e.g., non-existent domain), allow the request — the
	// actual HTTP call will fail anyway. Only block when resolved IPs are internal.
	//
	// Note: this does not fully prevent DNS rebinding. Even when this function
	// returns nil (allowed), the HTTP client re-resolves the hostname at
	// connection time, creating a TOCTOU window where the DNS record could
	// change to an internal address. Only blocked results are cached; allowed
	// hosts are always re-validated on each call.
	addrs, err := net.LookupHost(host)
	if err == nil {
		for _, addr := range addrs {
			resolved := net.ParseIP(addr)
			if resolved != nil {
				if ipErr := validateIP(resolved); ipErr != nil {
					validationErr := fmt.Errorf("hostname resolves to blocked address: %w", ipErr)
					hostCache.set(hostLower, validationErr)
					return validationErr
				}
			}
		}
	}

	return nil
}

// validateIP checks a single IP against blocked ranges.
func validateIP(ip net.IP) error {
	if ip.IsLoopback() {
		return fmt.Errorf("requests to loopback addresses are not allowed")
	}
	if ip.IsPrivate() {
		return fmt.Errorf("requests to private network addresses are not allowed")
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("requests to link-local addresses are not allowed")
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("requests to unspecified addresses are not allowed")
	}
	return nil
}
