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
