package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/source"
)

// maxRequestBodySize limits the size of incoming request bodies (1MB)
const maxRequestBodySize = 1 << 20

// defaultPageLimit is the default pagination limit when none is specified
const defaultPageLimit = 100

// APIHandler handles REST API requests for sources.
type APIHandler struct {
	config  *config.Config
	rootDir string
	sources map[string]source.Source // Cached sources
	mu      sync.RWMutex
}

// NewAPIHandler creates a new API handler.
func NewAPIHandler(cfg *config.Config, rootDir string) *APIHandler {
	return &APIHandler{
		config:  cfg,
		rootDir: rootDir,
		sources: make(map[string]source.Source),
	}
}

// Close releases all cached sources.
func (h *APIHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, src := range h.sources {
		if err := src.Close(); err != nil {
			log.Printf("[API] Error closing source: %v", err)
		}
	}
	h.sources = make(map[string]source.Source)
	return nil
}

// ServeHTTP handles API requests.
// Expected path format: /api/sources/{name}[/{itemID}]
func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract source name and optional item ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/sources/")
	if path == "" || path == r.URL.Path {
		writeError(w, http.StatusBadRequest, "source name required")
		return
	}

	parts := strings.SplitN(path, "/", 2)
	sourceName := parts[0]
	var itemID string
	if len(parts) > 1 {
		itemID = parts[1]
	}

	// Get or create the source
	src, err := h.getSource(sourceName)
	if err != nil {
		writeError(w, http.StatusNotFound, "source not found: "+sourceName)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, src)
	case http.MethodPost:
		h.handlePost(w, r, src)
	case http.MethodPut:
		h.handlePut(w, r, src, itemID)
	case http.MethodDelete:
		h.handleDelete(w, r, src, itemID)
	default:
		// Note: OPTIONS (preflight) is handled by CORS middleware before reaching here
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getSource retrieves or creates a source by name.
func (h *APIHandler) getSource(name string) (source.Source, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check cache first
	if src, exists := h.sources[name]; exists {
		return src, nil
	}

	// Get source config
	srcCfg, found := h.getSourceConfig(name)
	if !found {
		return nil, &sourceNotFoundError{name: name}
	}

	// Create the source
	src, err := h.createSource(name, srcCfg)
	if err != nil {
		return nil, err
	}

	h.sources[name] = src
	return src, nil
}

// getSourceConfig gets source configuration from site config.
func (h *APIHandler) getSourceConfig(name string) (config.SourceConfig, bool) {
	if h.config != nil && h.config.Sources != nil {
		if src, ok := h.config.Sources[name]; ok {
			return src, true
		}
	}
	return config.SourceConfig{}, false
}

// createSource creates a source instance from config.
func (h *APIHandler) createSource(name string, cfg config.SourceConfig) (source.Source, error) {
	switch cfg.Type {
	case "sqlite":
		return source.NewSQLiteSource(name, cfg.DB, cfg.Table, h.rootDir, cfg.IsReadonly())
	case "json":
		return source.NewJSONFileSource(name, cfg.File, h.rootDir)
	case "csv":
		return source.NewCSVFileSource(name, cfg.File, h.rootDir, cfg.Options)
	case "markdown":
		return source.NewMarkdownSource(name, cfg.File, cfg.Anchor, h.rootDir, "", cfg.IsReadonly())
	case "rest":
		return source.NewRestSourceWithConfig(name, cfg)
	case "pg":
		return source.NewPostgresSourceWithConfig(name, cfg.Query, cfg.Options, cfg)
	case "graphql":
		return source.NewGraphQLSource(name, cfg, h.rootDir)
	default:
		return nil, &unsupportedSourceTypeError{sourceType: cfg.Type}
	}
}

// handleGet fetches data from a source.
func (h *APIHandler) handleGet(w http.ResponseWriter, r *http.Request, src source.Source) {
	ctx := r.Context()

	data, err := src.Fetch(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Apply filter if provided
	if filter := r.URL.Query().Get("filter"); filter != "" {
		data = applyFilter(data, filter)
	}

	// Apply pagination with default limit to prevent unbounded results
	limit := parseIntParam(r, "limit", defaultPageLimit)
	offset := parseIntParam(r, "offset", 0)

	totalCount := len(data)
	data = paginate(data, offset, limit)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":   data,
		"count":  len(data),
		"total":  totalCount,
		"offset": offset,
		"limit":  limit,
	})
}

// handlePost creates a new item.
func (h *APIHandler) handlePost(w http.ResponseWriter, r *http.Request, src source.Source) {
	writable, ok := src.(source.WritableSource)
	if !ok || writable.IsReadonly() {
		writeError(w, http.StatusForbidden, "source is read-only")
		return
	}

	// Limit request body size to prevent DoS
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := writable.WriteItem(r.Context(), "add", data); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// handlePut updates an existing item.
func (h *APIHandler) handlePut(w http.ResponseWriter, r *http.Request, src source.Source, itemID string) {
	if itemID == "" {
		writeError(w, http.StatusBadRequest, "item ID required")
		return
	}

	writable, ok := src.(source.WritableSource)
	if !ok || writable.IsReadonly() {
		writeError(w, http.StatusForbidden, "source is read-only")
		return
	}

	// Limit request body size to prevent DoS
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Add the ID to the data
	data["id"] = itemID

	if err := writable.WriteItem(r.Context(), "update", data); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// handleDelete removes an item.
func (h *APIHandler) handleDelete(w http.ResponseWriter, r *http.Request, src source.Source, itemID string) {
	if itemID == "" {
		writeError(w, http.StatusBadRequest, "item ID required")
		return
	}

	writable, ok := src.(source.WritableSource)
	if !ok || writable.IsReadonly() {
		writeError(w, http.StatusForbidden, "source is read-only")
		return
	}

	if err := writable.WriteItem(r.Context(), "delete", map[string]interface{}{
		"id": itemID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[API] Error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("[API] Error encoding error response: %v", err)
	}
}

func parseIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

// applyFilter applies a simple filter to data.
// Filter format: "field=value" or "field!=value"
func applyFilter(data []map[string]interface{}, filter string) []map[string]interface{} {
	if filter == "" {
		return data
	}

	// Parse filter
	var field, value string
	var negate bool

	if idx := strings.Index(filter, "!="); idx != -1 {
		field = filter[:idx]
		value = filter[idx+2:]
		negate = true
	} else if idx := strings.Index(filter, "="); idx != -1 {
		field = filter[:idx]
		value = filter[idx+1:]
	} else {
		return data // Invalid filter format
	}

	// Apply filter
	result := make([]map[string]interface{}, 0)
	for _, item := range data {
		itemValue, ok := item[field]
		if !ok {
			if negate {
				result = append(result, item)
			}
			continue
		}

		// Convert to string for comparison
		itemStr := toString(itemValue)
		matches := itemStr == value
		if negate {
			matches = !matches
		}

		if matches {
			result = append(result, item)
		}
	}

	return result
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	default:
		return ""
	}
}

// paginate applies offset and limit to data.
func paginate(data []map[string]interface{}, offset, limit int) []map[string]interface{} {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(data) {
		return []map[string]interface{}{}
	}

	data = data[offset:]

	if limit > 0 && limit < len(data) {
		data = data[:limit]
	}

	return data
}

// Error types

type sourceNotFoundError struct {
	name string
}

func (e *sourceNotFoundError) Error() string {
	return "source not found: " + e.name
}

type unsupportedSourceTypeError struct {
	sourceType string
}

func (e *unsupportedSourceTypeError) Error() string {
	return "unsupported source type for API: " + e.sourceType
}
