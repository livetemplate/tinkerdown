package security

import (
	"strings"
	"testing"
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
	TestBypassSSRF = true
	defer func() { TestBypassSSRF = false }()

	// With bypass enabled, even localhost should be allowed
	if err := ValidateHTTPURL("http://localhost/admin"); err != nil {
		t.Errorf("ValidateHTTPURL() with bypass should allow localhost, got: %v", err)
	}
}
