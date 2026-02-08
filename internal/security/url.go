// Package security provides shared security validation functions.
package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
)

// testBypassSSRF is a testing hook to bypass SSRF validation.
// Use SetTestBypassSSRF to toggle it safely from tests.
var testBypassSSRF atomic.Bool

// SetTestBypassSSRF enables or disables the SSRF validation bypass for testing.
// It is safe for concurrent use.
func SetTestBypassSSRF(enabled bool) {
	testBypassSSRF.Store(enabled)
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

	// Hostname: resolve and check all resulting IPs to prevent DNS rebinding.
	// If resolution fails (e.g., non-existent domain), allow the request — the
	// actual HTTP call will fail anyway. Only block when resolved IPs are internal.
	// Note: This does not fully prevent DNS rebinding with TTL=0 tricks where the
	// DNS response changes between validation and the actual HTTP request.
	addrs, err := net.LookupHost(host)
	if err == nil {
		for _, addr := range addrs {
			resolved := net.ParseIP(addr)
			if resolved != nil {
				if err := validateIP(resolved); err != nil {
					return fmt.Errorf("hostname resolves to blocked address: %w", err)
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
