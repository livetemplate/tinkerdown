package source

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// JSONFileSource reads data from a JSON file
type JSONFileSource struct {
	name     string
	filePath string
	siteDir  string
}

// NewJSONFileSource creates a new JSON file source
func NewJSONFileSource(name, file, siteDir string) (*JSONFileSource, error) {
	if file == "" {
		return nil, fmt.Errorf("json source %q: file is required", name)
	}
	return &JSONFileSource{
		name:     name,
		filePath: file,
		siteDir:  siteDir,
	}, nil
}

// Name returns the source identifier
func (s *JSONFileSource) Name() string {
	return s.name
}

// Fetch reads and parses the JSON file
func (s *JSONFileSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	path := s.resolvePath(s.filePath)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("json source %q: failed to read file: %w", s.name, err)
	}

	return s.parseJSON(data)
}

// parseJSON handles both array and object JSON
func (s *JSONFileSource) parseJSON(data []byte) ([]map[string]interface{}, error) {
	data = []byte(strings.TrimSpace(string(data)))

	if len(data) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Try parsing as array first
	var arr []map[string]interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}

	// Try parsing as single object
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		// Check for "data" field (common pattern)
		if dataField, ok := obj["data"]; ok {
			if dataArr, ok := dataField.([]interface{}); ok {
				results := make([]map[string]interface{}, 0, len(dataArr))
				for _, item := range dataArr {
					if itemMap, ok := item.(map[string]interface{}); ok {
						results = append(results, itemMap)
					}
				}
				if len(results) > 0 {
					return results, nil
				}
			}
		}
		return []map[string]interface{}{obj}, nil
	}

	// Try parsing as newline-delimited JSON (NDJSON)
	lines := strings.Split(string(data), "\n")
	var results []map[string]interface{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return nil, fmt.Errorf("json source %q: invalid JSON: %w", s.name, err)
		}
		results = append(results, obj)
	}

	if len(results) > 0 {
		return results, nil
	}

	return nil, fmt.Errorf("json source %q: could not parse file as JSON", s.name)
}

// Close is a no-op for file sources
func (s *JSONFileSource) Close() error {
	return nil
}

// resolvePath makes a path absolute relative to siteDir
func (s *JSONFileSource) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.siteDir, path)
}

// CSVFileSource reads data from a CSV file
type CSVFileSource struct {
	name     string
	filePath string
	siteDir  string
	hasHeader bool
}

// NewCSVFileSource creates a new CSV file source
func NewCSVFileSource(name, file, siteDir string, options map[string]string) (*CSVFileSource, error) {
	if file == "" {
		return nil, fmt.Errorf("csv source %q: file is required", name)
	}

	hasHeader := true
	if options != nil && options["header"] == "false" {
		hasHeader = false
	}

	return &CSVFileSource{
		name:      name,
		filePath:  file,
		siteDir:   siteDir,
		hasHeader: hasHeader,
	}, nil
}

// Name returns the source identifier
func (s *CSVFileSource) Name() string {
	return s.name
}

// Fetch reads and parses the CSV file
func (s *CSVFileSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	path := s.resolvePath(s.filePath)

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("csv source %q: failed to open file: %w", s.name, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv source %q: failed to read CSV: %w", s.name, err)
	}

	if len(records) == 0 {
		return []map[string]interface{}{}, nil
	}

	var headers []string
	var dataRows [][]string

	if s.hasHeader {
		headers = records[0]
		dataRows = records[1:]
	} else {
		// Generate column names: col1, col2, col3, etc.
		if len(records) > 0 {
			headers = make([]string, len(records[0]))
			for i := range headers {
				headers[i] = fmt.Sprintf("col%d", i+1)
			}
		}
		dataRows = records
	}

	results := make([]map[string]interface{}, 0, len(dataRows))
	for _, row := range dataRows {
		rowMap := make(map[string]interface{})
		for i, value := range row {
			if i < len(headers) {
				rowMap[headers[i]] = value
			}
		}
		results = append(results, rowMap)
	}

	return results, nil
}

// Close is a no-op for file sources
func (s *CSVFileSource) Close() error {
	return nil
}

// resolvePath makes a path absolute relative to siteDir
func (s *CSVFileSource) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.siteDir, path)
}
