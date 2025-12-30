package config

import (
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
