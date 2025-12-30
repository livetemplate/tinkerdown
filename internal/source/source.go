// Package source provides data source implementations for lvt-source blocks.
// Sources fetch data from external systems (exec, databases, APIs) and make
// it available to templates.
package source

import (
	"context"

	"github.com/livetemplate/tinkerdown/internal/cache"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/wasm"
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

// WritableSource extends Source with write capability.
// Sources like markdown and sqlite implement this to support Add, Update, Delete actions.
type WritableSource interface {
	Source

	// WriteItem performs a write operation on the source.
	// action is one of: "add", "toggle", "delete", "update"
	// data contains the item data (e.g., form fields, id for delete)
	WriteItem(ctx context.Context, action string, data map[string]interface{}) error

	// IsReadonly returns whether the source is in read-only mode
	IsReadonly() bool
}

// Registry holds configured sources for a site
type Registry struct {
	sources map[string]Source
	cache   cache.Cache
	cfg     *config.Config
}

// NewRegistry creates a source registry from config
func NewRegistry(cfg *config.Config, siteDir string) (*Registry, error) {
	return NewRegistryWithFile(cfg, siteDir, "")
}

// NewRegistryWithFile creates a source registry with knowledge of the current markdown file
// This is needed for markdown sources that reference anchors in the current file
func NewRegistryWithFile(cfg *config.Config, siteDir, currentFile string) (*Registry, error) {
	memCache := cache.NewMemoryCache()
	r := &Registry{
		sources: make(map[string]Source),
		cache:   memCache,
		cfg:     cfg,
	}

	if cfg.Sources == nil {
		return r, nil
	}

	for name, srcCfg := range cfg.Sources {
		src, err := createSource(name, srcCfg, siteDir, currentFile)
		if err != nil {
			// Stop cache cleanup goroutine to avoid leak on initialization error
			memCache.Stop()
			return nil, err
		}

		// Wrap with caching if enabled
		if srcCfg.IsCacheEnabled() {
			if ws, ok := src.(WritableSource); ok {
				src = NewCachedWritableSource(ws, r.cache, srcCfg)
			} else {
				src = NewCachedSource(src, r.cache, srcCfg)
			}
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

// Close releases all sources and stops the cache
func (r *Registry) Close() error {
	// Stop the cache cleanup goroutine
	if mc, ok := r.cache.(*cache.MemoryCache); ok {
		mc.Stop()
	}

	for _, src := range r.sources {
		if err := src.Close(); err != nil {
			return err
		}
	}
	return nil
}

// InvalidateCache invalidates the cache for a specific source
func (r *Registry) InvalidateCache(name string) {
	src, ok := r.sources[name]
	if !ok {
		// Source not found; nothing to invalidate
		return
	}

	if cs, ok := src.(*CachedSource); ok {
		cs.Invalidate()
	} else if cws, ok := src.(*CachedWritableSource); ok {
		cws.Invalidate()
	}
}

// InvalidateAllCaches invalidates all cached data
func (r *Registry) InvalidateAllCaches() {
	r.cache.InvalidateAll()
}

// createSource instantiates a source based on config type
func createSource(name string, cfg config.SourceConfig, siteDir, currentFile string) (Source, error) {
	switch cfg.Type {
	case "exec":
		return NewExecSource(name, cfg.Cmd, siteDir)
	case "pg":
		return NewPostgresSourceWithConfig(name, cfg.Query, cfg.Options, cfg)
	case "rest":
		return NewRestSourceWithConfig(name, cfg.URL, cfg.Options, cfg)
	case "json":
		return NewJSONFileSource(name, cfg.File, siteDir)
	case "csv":
		return NewCSVFileSource(name, cfg.File, siteDir, cfg.Options)
	case "markdown":
		// Use IsReadonly() which defaults to true if not specified
		return NewMarkdownSource(name, cfg.File, cfg.Anchor, siteDir, currentFile, cfg.IsReadonly())
	case "sqlite":
		return NewSQLiteSource(name, cfg.DB, cfg.Table, siteDir, cfg.IsReadonly())
	case "wasm":
		return wasm.NewWasmSource(name, cfg.Path, siteDir, cfg.Options)
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
