package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the livepage configuration
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
	Type    string            `yaml:"type"`              // "exec", "pg", "rest", "csv", "json"
	Cmd     string            `yaml:"cmd,omitempty"`     // For exec: command to run
	Query   string            `yaml:"query,omitempty"`   // For pg: SQL query
	URL     string            `yaml:"url,omitempty"`     // For rest: API endpoint
	File    string            `yaml:"file,omitempty"`    // For csv/json: file path
	Options map[string]string `yaml:"options,omitempty"` // Type-specific options
	Manual  bool              `yaml:"manual,omitempty"`  // For exec: require Run button click
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

// LoadFromDir looks for livepage.yaml in the given directory
// If not found, returns the default configuration
func LoadFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, "livepage.yaml")
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
