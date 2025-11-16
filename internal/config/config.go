package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the livepage configuration
type Config struct {
	Title       string       `yaml:"title"`
	Description string       `yaml:"description"`
	Server      ServerConfig `yaml:"server"`
	Styling     StylingConfig `yaml:"styling"`
	Blocks      BlocksConfig `yaml:"blocks"`
	Features    FeaturesConfig `yaml:"features"`
	Ignore      []string     `yaml:"ignore"`
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
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Title:       "LiveTemplate Tutorial",
		Description: "Interactive tutorial powered by LiveTemplate",
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
