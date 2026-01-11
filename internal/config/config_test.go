package config

import (
	"strings"
	"testing"
	"time"
)

func TestSourceConfigIsCacheEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cache    *CacheConfig
		expected bool
	}{
		{"nil cache", nil, false},
		{"empty TTL", &CacheConfig{TTL: ""}, false},
		{"with TTL", &CacheConfig{TTL: "5m"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SourceConfig{Cache: tt.cache}
			if got := cfg.IsCacheEnabled(); got != tt.expected {
				t.Errorf("IsCacheEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSourceConfigGetCacheTTL(t *testing.T) {
	tests := []struct {
		name     string
		cache    *CacheConfig
		expected time.Duration
	}{
		{"nil cache", nil, 0},
		{"empty TTL", &CacheConfig{TTL: ""}, 0},
		{"invalid TTL", &CacheConfig{TTL: "invalid"}, 0},
		{"5 minutes", &CacheConfig{TTL: "5m"}, 5 * time.Minute},
		{"1 hour", &CacheConfig{TTL: "1h"}, time.Hour},
		{"30 seconds", &CacheConfig{TTL: "30s"}, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SourceConfig{Cache: tt.cache}
			if got := cfg.GetCacheTTL(); got != tt.expected {
				t.Errorf("GetCacheTTL() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSourceConfigGetCacheStrategy(t *testing.T) {
	tests := []struct {
		name     string
		cache    *CacheConfig
		expected string
	}{
		{"nil cache", nil, "simple"},
		{"empty strategy", &CacheConfig{Strategy: ""}, "simple"},
		{"simple", &CacheConfig{Strategy: "simple"}, "simple"},
		{"stale-while-revalidate", &CacheConfig{Strategy: "stale-while-revalidate"}, "stale-while-revalidate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SourceConfig{Cache: tt.cache}
			if got := cfg.GetCacheStrategy(); got != tt.expected {
				t.Errorf("GetCacheStrategy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSourceConfigIsStaleWhileRevalidate(t *testing.T) {
	tests := []struct {
		name     string
		cache    *CacheConfig
		expected bool
	}{
		{"nil cache", nil, false},
		{"empty strategy", &CacheConfig{Strategy: ""}, false},
		{"simple", &CacheConfig{Strategy: "simple"}, false},
		{"stale-while-revalidate", &CacheConfig{Strategy: "stale-while-revalidate"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SourceConfig{Cache: tt.cache}
			if got := cfg.IsStaleWhileRevalidate(); got != tt.expected {
				t.Errorf("IsStaleWhileRevalidate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWebhookValidate(t *testing.T) {
	actions := map[string]*Action{
		"test-action": {Kind: "http", URL: "http://example.com"},
	}

	tests := []struct {
		name      string
		webhook   *Webhook
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil webhook",
			webhook:   nil,
			wantError: true,
			errorMsg:  "is nil",
		},
		{
			name:      "empty action",
			webhook:   &Webhook{Action: ""},
			wantError: true,
			errorMsg:  "action is required",
		},
		{
			name:      "non-existent action",
			webhook:   &Webhook{Action: "missing-action"},
			wantError: true,
			errorMsg:  "non-existent action",
		},
		{
			name:      "negative timestamp tolerance",
			webhook:   &Webhook{Action: "test-action", ValidateTimestamp: true, TimestampTolerance: -1},
			wantError: true,
			errorMsg:  "cannot be negative",
		},
		{
			name:      "valid webhook with secret",
			webhook:   &Webhook{Action: "test-action", Secret: "mysecret"},
			wantError: false,
		},
		{
			name:      "valid webhook with signature secret",
			webhook:   &Webhook{Action: "test-action", SignatureSecret: "mysignaturesecret"},
			wantError: false,
		},
		{
			name:      "valid webhook with timestamp validation",
			webhook:   &Webhook{Action: "test-action", ValidateTimestamp: true, TimestampTolerance: 300},
			wantError: false,
		},
		{
			name:      "valid webhook with default timestamp tolerance",
			webhook:   &Webhook{Action: "test-action", ValidateTimestamp: true},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.webhook.Validate("test-webhook", actions)
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConfigValidateWebhooks(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name:      "nil webhooks",
			config:    &Config{},
			wantError: false,
		},
		{
			name: "valid webhooks",
			config: &Config{
				Actions: map[string]*Action{
					"action1": {Kind: "http", URL: "http://example.com"},
					"action2": {Kind: "sql", Statement: "SELECT 1"},
				},
				Webhooks: map[string]*Webhook{
					"webhook1": {Action: "action1"},
					"webhook2": {Action: "action2", Secret: "secret"},
				},
			},
			wantError: false,
		},
		{
			name: "invalid webhook - missing action reference",
			config: &Config{
				Actions: map[string]*Action{
					"action1": {Kind: "http", URL: "http://example.com"},
				},
				Webhooks: map[string]*Webhook{
					"webhook1": {Action: "nonexistent"},
				},
			},
			wantError: true,
		},
		{
			name: "invalid webhook - empty action",
			config: &Config{
				Webhooks: map[string]*Webhook{
					"webhook1": {Action: ""},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateWebhooks()
			if tt.wantError && err == nil {
				t.Errorf("ValidateWebhooks() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateWebhooks() unexpected error = %v", err)
			}
		})
	}
}

func TestOutputConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		output    *OutputConfig
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil output",
			output:    nil,
			wantError: true,
			errorMsg:  "is nil",
		},
		{
			name:      "unsupported type",
			output:    &OutputConfig{Type: "discord"},
			wantError: true,
			errorMsg:  "unsupported type",
		},
		{
			name:      "slack without channel",
			output:    &OutputConfig{Type: "slack", Channel: ""},
			wantError: true,
			errorMsg:  "channel is required",
		},
		{
			name:      "email without to",
			output:    &OutputConfig{Type: "email", To: ""},
			wantError: true,
			errorMsg:  "to is required",
		},
		{
			name:      "valid slack",
			output:    &OutputConfig{Type: "slack", Channel: "#test"},
			wantError: false,
		},
		{
			name:      "valid email",
			output:    &OutputConfig{Type: "email", To: "user@example.com"},
			wantError: false,
		},
		{
			name:      "valid email with subject",
			output:    &OutputConfig{Type: "email", To: "user@example.com", Subject: "Alert"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.output.Validate("test-output")
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestOutputConfigGetChannel(t *testing.T) {
	// Test with nil
	var nilOutput *OutputConfig
	if got := nilOutput.GetChannel(); got != "" {
		t.Errorf("GetChannel() on nil = %q, want empty string", got)
	}

	// Test with empty
	output := &OutputConfig{Channel: ""}
	if got := output.GetChannel(); got != "" {
		t.Errorf("GetChannel() on empty = %q, want empty string", got)
	}

	// Test with value
	output = &OutputConfig{Channel: "#test-channel"}
	if got := output.GetChannel(); got != "#test-channel" {
		t.Errorf("GetChannel() = %q, want #test-channel", got)
	}

	// Test with env var expansion
	t.Setenv("TEST_CHANNEL", "#from-env")
	output = &OutputConfig{Channel: "${TEST_CHANNEL}"}
	if got := output.GetChannel(); got != "#from-env" {
		t.Errorf("GetChannel() with env var = %q, want #from-env", got)
	}
}

func TestOutputConfigGetTo(t *testing.T) {
	// Test with nil
	var nilOutput *OutputConfig
	if got := nilOutput.GetTo(); got != "" {
		t.Errorf("GetTo() on nil = %q, want empty string", got)
	}

	// Test with empty
	output := &OutputConfig{To: ""}
	if got := output.GetTo(); got != "" {
		t.Errorf("GetTo() on empty = %q, want empty string", got)
	}

	// Test with value
	output = &OutputConfig{To: "user@example.com"}
	if got := output.GetTo(); got != "user@example.com" {
		t.Errorf("GetTo() = %q, want user@example.com", got)
	}

	// Test with env var expansion
	t.Setenv("TEST_EMAIL", "env@example.com")
	output = &OutputConfig{To: "${TEST_EMAIL}"}
	if got := output.GetTo(); got != "env@example.com" {
		t.Errorf("GetTo() with env var = %q, want env@example.com", got)
	}
}

func TestConfigValidateOutputs(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name:      "nil outputs",
			config:    &Config{},
			wantError: false,
		},
		{
			name: "valid outputs",
			config: &Config{
				Outputs: map[string]*OutputConfig{
					"slack": {Type: "slack", Channel: "#alerts"},
					"email": {Type: "email", To: "alerts@example.com"},
				},
			},
			wantError: false,
		},
		{
			name: "invalid output - unsupported type",
			config: &Config{
				Outputs: map[string]*OutputConfig{
					"invalid": {Type: "discord"},
				},
			},
			wantError: true,
		},
		{
			name: "invalid output - slack missing channel",
			config: &Config{
				Outputs: map[string]*OutputConfig{
					"slack": {Type: "slack"},
				},
			},
			wantError: true,
		},
		{
			name: "invalid output - email missing to",
			config: &Config{
				Outputs: map[string]*OutputConfig{
					"email": {Type: "email"},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateOutputs()
			if tt.wantError && err == nil {
				t.Errorf("ValidateOutputs() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateOutputs() unexpected error = %v", err)
			}
		})
	}
}
