// Package source provides data source implementations for lvt-source blocks.
// Sources fetch data from external systems (exec, databases, APIs) and make
// it available to templates.
package source

import (
	"context"

	"github.com/livetemplate/livepage/internal/config"
)

// Source is the interface for data providers.
// Implementations fetch data from various backends (exec, pg, rest, etc.)
type Source interface {
	// Name returns the source identifier
	Name() string

	// Fetch retrieves data from the source.
	// Returns a slice of maps suitable for template iteration.
	Fetch(ctx context.Context) ([]map[string]interface{}, error)

	// Close releases any resources held by the source
	Close() error
}

// Registry holds configured sources for a site
type Registry struct {
	sources map[string]Source
}

// NewRegistry creates a source registry from config
func NewRegistry(cfg *config.Config, siteDir string) (*Registry, error) {
	r := &Registry{
		sources: make(map[string]Source),
	}

	if cfg.Sources == nil {
		return r, nil
	}

	for name, srcCfg := range cfg.Sources {
		src, err := createSource(name, srcCfg, siteDir)
		if err != nil {
			return nil, err
		}
		r.sources[name] = src
	}

	return r, nil
}

// Get returns a source by name
func (r *Registry) Get(name string) (Source, bool) {
	src, ok := r.sources[name]
	return src, ok
}

// Close releases all sources
func (r *Registry) Close() error {
	for _, src := range r.sources {
		if err := src.Close(); err != nil {
			return err
		}
	}
	return nil
}

// createSource instantiates a source based on config type
func createSource(name string, cfg config.SourceConfig, siteDir string) (Source, error) {
	switch cfg.Type {
	case "exec":
		return NewExecSource(name, cfg.Cmd, siteDir)
	case "pg":
		return NewPostgresSource(name, cfg.Query, cfg.Options)
	case "rest":
		return NewRestSource(name, cfg.URL, cfg.Options)
	case "json":
		return NewJSONFileSource(name, cfg.File, siteDir)
	case "csv":
		return NewCSVFileSource(name, cfg.File, siteDir, cfg.Options)
	default:
		return nil, &UnsupportedSourceError{Type: cfg.Type}
	}
}

// UnsupportedSourceError is returned for unknown source types
type UnsupportedSourceError struct {
	Type string
}

func (e *UnsupportedSourceError) Error() string {
	return "unsupported source type: " + e.Type
}
