package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the tinkerdown configuration
type Config struct {
	Title       string                  `yaml:"title"`
	Description string                  `yaml:"description"`
	Type        string                  `yaml:"type"` // "tutorial" or "site"
	Site        *SiteConfig             `yaml:"site,omitempty"`
	Navigation  []NavSection            `yaml:"navigation,omitempty"`
	Server      ServerConfig            `yaml:"server"`
	Styling     StylingConfig           `yaml:"styling"`
	Blocks      BlocksConfig            `yaml:"blocks"`
	Features    FeaturesConfig          `yaml:"features"`
	Ignore      []string                `yaml:"ignore"`
	Sources     map[string]SourceConfig `yaml:"sources,omitempty"`
}

// SourceConfig defines a data source for lvt-source blocks
type SourceConfig struct {
	Type       string                 `yaml:"type"`                  // "exec", "pg", "rest", "csv", "json", "markdown", "sqlite", "wasm", "graphql"
	Cmd        string                 `yaml:"cmd,omitempty"`         // For exec: command to run
	Query      string                 `yaml:"query,omitempty"`       // For pg: SQL query
	URL        string                 `yaml:"url,omitempty"`         // For rest/graphql: API endpoint
	File       string                 `yaml:"file,omitempty"`        // For csv/json/markdown: file path
	Anchor     string                 `yaml:"anchor,omitempty"`      // For markdown: section anchor (e.g., "#todos")
	DB         string                 `yaml:"db,omitempty"`          // For sqlite: database file path (default: ./tinkerdown.db)
	Table      string                 `yaml:"table,omitempty"`       // For sqlite: table name
	Path       string                 `yaml:"path,omitempty"`        // For wasm: path to .wasm file
	QueryFile  string                 `yaml:"query_file,omitempty"`  // For graphql: path to .graphql file
	Variables  map[string]interface{} `yaml:"variables,omitempty"`   // For graphql: query variables
	ResultPath string                 `yaml:"result_path,omitempty"` // For graphql: dot-path to extract array (e.g., "repository.issues.nodes")
	Readonly   *bool                  `yaml:"readonly,omitempty"`    // For markdown/sqlite: read-only mode (default: true, set to false for writes)
	Options    map[string]string      `yaml:"options,omitempty"`     // Type-specific options (also used for wasm init config)
	Manual     bool                   `yaml:"manual,omitempty"`      // For exec: require Run button click
	Timeout    string                 `yaml:"timeout,omitempty"`     // Request timeout (e.g., "30s", "1m"). Default: 10s
	Retry      *RetryConfig           `yaml:"retry,omitempty"`       // Retry configuration
	Cache      *CacheConfig           `yaml:"cache,omitempty"`       // Cache configuration
}

// RetryConfig configures retry behavior for a source
type RetryConfig struct {
	MaxRetries int    `yaml:"max_retries,omitempty"` // Maximum retry attempts (default: 3)
	BaseDelay  string `yaml:"base_delay,omitempty"`  // Initial delay (e.g., "100ms"). Default: 100ms
	MaxDelay   string `yaml:"max_delay,omitempty"`   // Maximum delay (e.g., "5s"). Default: 5s
}

// CacheConfig configures caching behavior for a source
type CacheConfig struct {
	TTL      string `yaml:"ttl,omitempty"`      // Cache TTL (e.g., "5m", "1h"). Default: disabled (empty)
	Strategy string `yaml:"strategy,omitempty"` // Cache strategy: "simple" or "stale-while-revalidate". Default: "simple"
}

// IsReadonly returns true if the source is read-only (default: true for markdown sources)
func (c SourceConfig) IsReadonly() bool {
	if c.Readonly == nil {
		return true // Default to read-only for safety
	}
	return *c.Readonly
}

// GetTimeout returns the parsed timeout duration (default: 10s)
func (c SourceConfig) GetTimeout() time.Duration {
	if c.Timeout == "" {
		return 10 * time.Second
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

// GetRetryMaxRetries returns the max retries (default: 3, set to 0 to disable retries)
func (c SourceConfig) GetRetryMaxRetries() int {
	if c.Retry == nil {
		return 3
	}
	if c.Retry.MaxRetries < 0 {
		return 3
	}
	return c.Retry.MaxRetries
}

// GetRetryBaseDelay returns the base delay (default: 100ms)
func (c SourceConfig) GetRetryBaseDelay() time.Duration {
	if c.Retry == nil || c.Retry.BaseDelay == "" {
		return 100 * time.Millisecond
	}
	d, err := time.ParseDuration(c.Retry.BaseDelay)
	if err != nil {
		return 100 * time.Millisecond
	}
	return d
}

// GetRetryMaxDelay returns the max delay (default: 5s)
func (c SourceConfig) GetRetryMaxDelay() time.Duration {
	if c.Retry == nil || c.Retry.MaxDelay == "" {
		return 5 * time.Second
	}
	d, err := time.ParseDuration(c.Retry.MaxDelay)
	if err != nil {
		return 5 * time.Second
	}
	return d
}

// IsCacheEnabled returns true if caching is enabled for this source
func (c SourceConfig) IsCacheEnabled() bool {
	return c.Cache != nil && c.Cache.TTL != ""
}

// GetCacheTTL returns the cache TTL (0 if caching is disabled)
func (c SourceConfig) GetCacheTTL() time.Duration {
	if c.Cache == nil || c.Cache.TTL == "" {
		return 0
	}
	d, err := time.ParseDuration(c.Cache.TTL)
	if err != nil {
		return 0
	}
	return d
}

// GetCacheStrategy returns the cache strategy (default: "simple")
func (c SourceConfig) GetCacheStrategy() string {
	if c.Cache == nil || c.Cache.Strategy == "" {
		return "simple"
	}
	return c.Cache.Strategy
}

// IsStaleWhileRevalidate returns true if using stale-while-revalidate strategy
func (c SourceConfig) IsStaleWhileRevalidate() bool {
	return c.GetCacheStrategy() == "stale-while-revalidate"
}

// SiteConfig holds site-level configuration
type SiteConfig struct {
	Home       string `yaml:"home"`        // Homepage markdown file (e.g., "index.md")
	Logo       string `yaml:"logo"`        // Logo path (e.g., "/assets/logo.svg")
	Repository string `yaml:"repository"`  // GitHub repository URL
}

// NavSection represents a navigation section with pages
type NavSection struct {
	Title     string    `yaml:"title"`              // Section title (e.g., "Getting Started")
	Path      string    `yaml:"path"`               // Section path (e.g., "getting-started")
	Collapsed bool      `yaml:"collapsed"`          // Whether section is collapsed by default
	Pages     []NavPage `yaml:"pages,omitempty"`    // Pages in this section
}

// NavPage represents a single page in navigation
type NavPage struct {
	Title string `yaml:"title"` // Page title (e.g., "Installation")
	Path  string `yaml:"path"`  // Page path (e.g., "getting-started/installation.md")
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port  int    `yaml:"port"`
	Host  string `yaml:"host"`
	Debug bool   `yaml:"debug"`
}

// StylingConfig holds styling-related configuration
type StylingConfig struct {
	Theme        string `yaml:"theme"`
	PrimaryColor string `yaml:"primary_color"`
	Font         string `yaml:"font"`
}

// BlocksConfig holds block-related configuration
type BlocksConfig struct {
	AutoID          bool   `yaml:"auto_id"`
	IDFormat        string `yaml:"id_format"`
	ShowLineNumbers bool   `yaml:"show_line_numbers"`
}

// FeaturesConfig holds feature flags
type FeaturesConfig struct {
	HotReload bool `yaml:"hot_reload"`
	Sidebar   bool `yaml:"sidebar"` // Show navigation sidebar (default: false)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Title:       "LiveTemplate Tutorial",
		Description: "Interactive tutorial powered by LiveTemplate",
		Type:        "tutorial", // Default to tutorial mode
		Server: ServerConfig{
			Port:  8080,
			Host:  "localhost",
			Debug: false,
		},
		Styling: StylingConfig{
			Theme:        "clean",
			PrimaryColor: "#007bff",
			Font:         "system-ui",
		},
		Blocks: BlocksConfig{
			AutoID:          true,
			IDFormat:        "kebab-case",
			ShowLineNumbers: true,
		},
		Features: FeaturesConfig{
			HotReload: true,
		},
		Ignore: []string{
			"drafts/**",
			"_*.md",
		},
	}
}

// IsSiteMode returns true if the config is for a multi-page site
func (c *Config) IsSiteMode() bool {
	return c.Type == "site"
}

// IsTutorialMode returns true if the config is for a single tutorial
func (c *Config) IsTutorialMode() bool {
	return c.Type == "tutorial" || c.Type == ""
}

// Load loads configuration from a YAML file
// If the file doesn't exist, returns the default configuration
func Load(configPath string) (*Config, error) {
	// If no config path provided, use default
	if configPath == "" {
		return DefaultConfig(), nil
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config := DefaultConfig() // Start with defaults
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// LoadFromDir looks for tinkerdown.yaml, lmt.yaml, or livemdtools.yaml in the given directory
// tinkerdown.yaml is checked first, then lmt.yaml (short form), then livemdtools.yaml (legacy)
// If none is found, returns the default configuration
func LoadFromDir(dir string) (*Config, error) {
	// Check for tinkerdown.yaml first (new name)
	tinkerdownPath := filepath.Join(dir, "tinkerdown.yaml")
	if _, err := os.Stat(tinkerdownPath); err == nil {
		return Load(tinkerdownPath)
	}

	// Check for lmt.yaml (short form)
	lmtPath := filepath.Join(dir, "lmt.yaml")
	if _, err := os.Stat(lmtPath); err == nil {
		return Load(lmtPath)
	}

	// Check for livemdtools.yaml (legacy, backwards compatibility)
	configPath := filepath.Join(dir, "livemdtools.yaml")
	return Load(configPath)
}

// Save writes the configuration to a YAML file
func (c *Config) Save(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
