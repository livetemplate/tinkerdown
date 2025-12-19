package source

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ExecSource runs a command and parses JSON output as data
type ExecSource struct {
	name    string
	cmd     string
	siteDir string
}

// NewExecSource creates a new exec source
func NewExecSource(name, cmd, siteDir string) (*ExecSource, error) {
	if cmd == "" {
		return nil, fmt.Errorf("exec source %q: cmd is required", name)
	}
	return &ExecSource{
		name:    name,
		cmd:     cmd,
		siteDir: siteDir,
	}, nil
}

// Name returns the source identifier
func (s *ExecSource) Name() string {
	return s.name
}

// Fetch executes the command and parses JSON output
func (s *ExecSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Parse command - split by spaces (simple parsing)
	parts := strings.Fields(s.cmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("exec source %q: empty command", s.name)
	}

	cmdName := parts[0]
	args := parts[1:]

	// Create command with context and timeout
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, cmdName, args...)
	cmd.Dir = s.siteDir

	// Execute and capture output
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("exec source %q: command failed: %s\nstderr: %s",
				s.name, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("exec source %q: %w", s.name, err)
	}

	// Parse JSON output
	return s.parseJSON(output)
}

// parseJSON handles both array and object JSON responses
func (s *ExecSource) parseJSON(data []byte) ([]map[string]interface{}, error) {
	// Trim whitespace
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
			return nil, fmt.Errorf("exec source %q: invalid JSON output: %w", s.name, err)
		}
		results = append(results, obj)
	}

	if len(results) > 0 {
		return results, nil
	}

	return nil, fmt.Errorf("exec source %q: could not parse output as JSON", s.name)
}

// Close is a no-op for exec sources
func (s *ExecSource) Close() error {
	return nil
}

// resolvePath makes a path absolute relative to siteDir
func (s *ExecSource) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.siteDir, path)
}
