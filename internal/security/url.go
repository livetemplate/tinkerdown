// Package security provides shared security validation functions.
package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// TestBypassSSRF is a testing hook to bypass SSRF validation.
// This should only be set in tests.
var TestBypassSSRF bool

// ValidateHTTPURL checks for SSRF vulnerabilities by blocking requests to internal networks.
// It rejects localhost, private IP ranges, link-local addresses, and cloud metadata endpoints.
func ValidateHTTPURL(rawURL string) error {
	if TestBypassSSRF {
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
	if ip == nil {
		// Not an IP address - could be a hostname that resolves to internal IP.
		// A more complete solution would resolve the hostname and check the IP.
		return nil
	}

	// Block loopback addresses (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return fmt.Errorf("requests to loopback addresses are not allowed")
	}

	// Block private network addresses
	if ip.IsPrivate() {
		return fmt.Errorf("requests to private network addresses are not allowed")
	}

	// Block link-local addresses (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("requests to link-local addresses are not allowed")
	}

	// Block unspecified addresses (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return fmt.Errorf("requests to unspecified addresses are not allowed")
	}

	return nil
}
