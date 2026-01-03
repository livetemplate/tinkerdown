// Package runtime provides in-process state handling for lvt-source blocks,
// replacing the previous plugin-based compilation approach.
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/livetemplate/components/datatable"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/source"
	"github.com/livetemplate/tinkerdown/internal/wasm"
)

// Store is the interface for state objects that can handle actions.
// All state types (GenericState for lvt-source) implement this interface.
type Store interface {
	HandleAction(action string, data map[string]interface{}) error
	// Close releases resources. Optional - returns nil if not implemented.
	Close() error
}

// GenericState holds runtime state for any source type.
// It replaces the code-generated State structs that were previously compiled as plugins.
type GenericState struct {
	// Common fields (JSON-serializable for templates)
	Data   []map[string]interface{} `json:"data"`
	Error  string                   `json:"error,omitempty"`
	Errors map[string]string        `json:"errors,omitempty"`

	// Datatable field - used when source is rendered in a table element
	Table *datatable.DataTable `json:"table,omitempty"`

	// Exec-specific fields
	Output     string   `json:"output,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	Duration   int64    `json:"duration,omitempty"`
	Status     string   `json:"status,omitempty"`
	Command    string   `json:"command,omitempty"`
	Args       []Arg    `json:"args,omitempty"`
	Executable string   `json:"executable,omitempty"`

	// Private runtime fields (not serialized)
	source       source.Source
	sourceCfg    config.SourceConfig
	sourceType   string
	sourceName   string
	siteDir      string
	elementType  string   // "table", "select", or "div"
	tableColumns []string // columns for datatable rendering
	mu           sync.RWMutex
}

// Arg represents an exec source argument
type Arg struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Value       string `json:"value,omitempty"`
}

// NewGenericState creates a new state for the given source configuration.
// This replaces the plugin compilation step - state is created directly in-process.
func NewGenericState(name string, cfg config.SourceConfig, siteDir, currentFile string) (*GenericState, error) {
	return NewGenericStateWithMetadata(name, cfg, siteDir, currentFile, nil)
}

// NewGenericStateWithMetadata creates a new state with block metadata for datatable support.
// Metadata should include "lvt-element" ("table", "select", or "div") and "lvt-columns" for tables.
func NewGenericStateWithMetadata(name string, cfg config.SourceConfig, siteDir, currentFile string, metadata map[string]string) (*GenericState, error) {
	// Create the underlying source using the existing factory
	src, err := createSource(name, cfg, siteDir, currentFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create source %q: %w", name, err)
	}

	s := &GenericState{
		source:     src,
		sourceCfg:  cfg,
		sourceType: cfg.Type,
		sourceName: name,
		siteDir:    siteDir,
		Errors:     make(map[string]string),
	}

	// Parse metadata for element type and columns
	if metadata != nil {
		s.elementType = metadata["lvt-element"]
		if columns := metadata["lvt-columns"]; columns != "" {
			// Parse "name:Name,email:Email" format
			for _, pair := range strings.Split(columns, ",") {
				parts := strings.SplitN(pair, ":", 2)
				if len(parts) > 0 {
					s.tableColumns = append(s.tableColumns, parts[0])
				}
			}
		}
	}

	// Set exec-specific fields if applicable
	if cfg.Type == "exec" {
		s.Command = cfg.Cmd
		s.Status = "ready"
		// Parse command-line arguments for form rendering
		s.Args = parseExecArgs(cfg.Cmd)
		// If manual mode, don't auto-fetch
		if cfg.Manual {
			return s, nil
		}
	}

	// Initial data fetch
	if err := s.refresh(); err != nil {
		s.Error = err.Error()
	}

	return s, nil
}

// createSource creates a source from config (mirrors source.createSource)
func createSource(name string, cfg config.SourceConfig, siteDir, currentFile string) (source.Source, error) {
	switch cfg.Type {
	case "exec":
		if !config.IsExecAllowed() {
			return nil, &source.ValidationError{
				Source: name,
				Field:  "type",
				Reason: "exec sources are disabled by default for security. Use --allow-exec flag to enable.",
			}
		}
		return source.NewExecSource(name, cfg.Cmd, siteDir)
	case "pg":
		return source.NewPostgresSource(name, cfg.Query, cfg.Options)
	case "rest":
		return source.NewRestSource(name, cfg.URL, cfg.Options)
	case "json":
		return source.NewJSONFileSource(name, cfg.File, siteDir)
	case "csv":
		return source.NewCSVFileSource(name, cfg.File, siteDir, cfg.Options)
	case "markdown":
		return source.NewMarkdownSource(name, cfg.File, cfg.Anchor, siteDir, currentFile, cfg.IsReadonly())
	case "sqlite":
		return source.NewSQLiteSource(name, cfg.DB, cfg.Table, siteDir, cfg.IsReadonly())
	case "wasm":
		return wasm.NewWasmSource(name, cfg.Path, siteDir, cfg.Options)
	case "graphql":
		return source.NewGraphQLSource(name, cfg, siteDir)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", cfg.Type)
	}
}

// HandleAction dispatches an action to the appropriate handler.
// This replaces the reflection-based dispatch used in generated plugins.
func (s *GenericState) HandleAction(action string, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear previous errors
	s.Errors = make(map[string]string)
	s.Error = ""

	// Normalize action to lowercase for matching
	actionLower := strings.ToLower(action)

	switch actionLower {
	case "refresh":
		return s.refresh()
	case "run":
		return s.runExec(data)
	case "add", "toggle", "delete", "update":
		return s.handleWriteAction(action, data)
	default:
		// Check for datatable actions (Sort_X, NextPage_X, PrevPage_X)
		if strings.HasPrefix(actionLower, "sort") ||
			strings.HasPrefix(actionLower, "nextpage") ||
			strings.HasPrefix(actionLower, "prevpage") {
			return s.handleDatatableAction(action, data)
		}
		return fmt.Errorf("unknown action: %s", action)
	}
}

// GetState returns the current state for template rendering.
// This replaces the RPC GetState() call.
func (s *GenericState) GetState() (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s)
}

// GetStateAsInterface returns the state as an interface{} for template rendering.
// This is used by the WebSocket handler to render templates.
func (s *GenericState) GetStateAsInterface() (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a map representation for template compatibility
	stateBytes, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(stateBytes, &rawMap); err != nil {
		return nil, err
	}

	// Process state map to add titlecase keys for template access
	// This allows templates to use both {{.data}} and {{.Data}}
	return processStateMap(rawMap), nil
}

// processStateMap processes a map to add titlecase keys alongside lowercase keys.
// This allows templates to use both {{.data}} and {{.Data}}, {{.status}} and {{.Status}}.
// It also processes nested maps and slices recursively.
func processStateMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		var processedValue interface{}

		switch val := v.(type) {
		case map[string]interface{}:
			processedValue = processMapValues(val)
		case []interface{}:
			newSlice := make([]interface{}, len(val))
			for i, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					newSlice[i] = processMapValues(itemMap)
				} else {
					newSlice[i] = convertNumber(item)
				}
			}
			processedValue = newSlice
		case float64:
			processedValue = convertNumber(val)
		default:
			processedValue = v
		}

		// Keep original key (e.g., "status", "data", "items")
		result[k] = processedValue

		// Also add titlecased key if different (e.g., "Status", "Data", "Items")
		// This allows templates to use either {{.status}} or {{.Status}}
		if len(k) > 0 {
			titleKey := strings.ToUpper(k[:1]) + k[1:]
			if titleKey != k {
				result[titleKey] = processedValue
			}
		}
	}
	return result
}

// processMapValues recursively processes map values to add titlecase keys.
func processMapValues(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		var processedValue interface{}

		switch val := v.(type) {
		case map[string]interface{}:
			processedValue = processMapValues(val)
		case []interface{}:
			newSlice := make([]interface{}, len(val))
			for i, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					newSlice[i] = processMapValues(itemMap)
				} else {
					newSlice[i] = convertNumber(item)
				}
			}
			processedValue = newSlice
		case float64:
			processedValue = convertNumber(val)
		default:
			processedValue = v
		}

		// Keep original key
		result[k] = processedValue

		// Also add titlecased key if different
		if len(k) > 0 {
			titleKey := strings.ToUpper(k[:1]) + k[1:]
			if titleKey != k {
				result[titleKey] = processedValue
			}
		}
	}
	return result
}

// convertNumber converts float64 values that are whole numbers to int.
// JSON unmarshals all numbers as float64, but int is often more useful.
func convertNumber(v interface{}) interface{} {
	if f, ok := v.(float64); ok {
		if f == float64(int(f)) {
			return int(f)
		}
	}
	return v
}

// Close releases any resources held by the source.
func (s *GenericState) Close() error {
	if s.source != nil {
		return s.source.Close()
	}
	return nil
}

// refresh fetches data from the source
func (s *GenericState) refresh() error {
	if s.source == nil {
		return fmt.Errorf("source not initialized")
	}

	ctx := context.Background()
	data, err := s.source.Fetch(ctx)
	if err != nil {
		s.Error = err.Error()
		return err
	}

	s.Data = data
	s.Error = ""

	// Build DataTable if this is a table element
	if s.elementType == "table" {
		s.Table = s.buildDataTable()
	}

	return nil
}

// buildDataTable creates a datatable.DataTable from the current Data
func (s *GenericState) buildDataTable() *datatable.DataTable {
	if len(s.Data) == 0 {
		return nil
	}

	// Determine columns - use explicit columns or auto-discover from first row
	var columns []datatable.Column
	if len(s.tableColumns) > 0 {
		for _, col := range s.tableColumns {
			label := col
			if len(label) > 0 {
				label = strings.ToUpper(label[:1]) + label[1:]
			}
			columns = append(columns, datatable.Column{
				ID:       col,
				Label:    label,
				Sortable: true,
			})
		}
	} else {
		// Auto-discover columns from first row
		for key := range s.Data[0] {
			// Skip internal fields starting with uppercase (title-cased duplicates)
			if len(key) > 0 && key[0] >= 'A' && key[0] <= 'Z' {
				continue
			}
			label := key
			if len(label) > 0 {
				label = strings.ToUpper(label[:1]) + label[1:]
			}
			columns = append(columns, datatable.Column{
				ID:       key,
				Label:    label,
				Sortable: true,
			})
		}
	}

	// Build rows
	var rows []datatable.Row
	for i, item := range s.Data {
		data := make(map[string]any)
		for _, col := range columns {
			if val, ok := item[col.ID]; ok {
				data[col.ID] = val
			} else {
				// Try titlecase key
				titleKey := col.ID
				if len(titleKey) > 0 {
					titleKey = strings.ToUpper(titleKey[:1]) + titleKey[1:]
				}
				if val, ok := item[titleKey]; ok {
					data[col.ID] = val
				}
			}
		}
		// Generate row ID
		rowID := fmt.Sprintf("row-%d", i)
		if id, ok := item["id"]; ok {
			rowID = fmt.Sprintf("%v", id)
		} else if id, ok := item["Id"]; ok {
			rowID = fmt.Sprintf("%v", id)
		}
		rows = append(rows, datatable.Row{ID: rowID, Data: data})
	}

	return datatable.New(s.sourceName, datatable.WithColumns(columns), datatable.WithRows(rows))
}

// parseExecArgs parses command-line arguments from a command string.
// It extracts --flag value pairs and infers types from values.
// Example: "./script.sh --name World --count 3 --verbose true"
// Returns Args with Name, Label, Type, and Value set.
func parseExecArgs(cmd string) []Arg {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	var args []Arg

	// Skip the executable (first part)
	for i := 1; i < len(parts); i++ {
		part := parts[i]

		// Check for --flag or -flag
		if !strings.HasPrefix(part, "-") {
			continue
		}

		// Extract flag name (remove leading dashes)
		name := strings.TrimLeft(part, "-")
		if name == "" {
			continue
		}

		// Get the value (next part if available and doesn't start with -)
		value := ""
		if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
			i++
			value = parts[i]
		}

		// Infer type from value
		argType := "string"
		if value == "true" || value == "false" {
			argType = "bool"
		} else if isNumeric(value) {
			argType = "number"
		}

		// Create label from name (capitalize first letter)
		label := name
		if len(label) > 0 {
			label = strings.ToUpper(label[:1]) + label[1:]
		}

		args = append(args, Arg{
			Name:    name,
			Label:   label,
			Type:    argType,
			Value:   value,
			Default: value,
		})
	}

	return args
}

// isNumeric checks if a string represents a number
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	// Allow optional leading minus sign
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	hasDecimal := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if hasDecimal {
				return false
			}
			hasDecimal = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
