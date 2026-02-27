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
	mu      sync.Mutex
	entries map[string]dnsCacheEntry
	ttl     time.Duration
	maxSize int
}

var hostCache = &dnsCache{
	entries: make(map[string]dnsCacheEntry),
	ttl:     30 * time.Second,
	maxSize: 1024,
}

func (c *dnsCache) get(host string) (error, bool) {
	c.mu.Lock()
	entry, ok := c.entries[host]
	if !ok {
		c.mu.Unlock()
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.entries, host)
		c.mu.Unlock()
		return nil, false
	}
	c.mu.Unlock()
	return entry.err, true
}

func (c *dnsCache) set(host string, err error) {
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
			c.entries = make(map[string]dnsCacheEntry)
		}
	}
	c.entries[host] = dnsCacheEntry{
		err:       err,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// ResetDNSCache clears the DNS validation cache. Intended for use in tests.
func ResetDNSCache() {
	hostCache.mu.Lock()
	hostCache.entries = make(map[string]dnsCacheEntry)
	hostCache.mu.Unlock()
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
