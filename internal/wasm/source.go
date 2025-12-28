// Package wasm provides WASM-based data source support for tinkerdown.
// This allows community sources to be distributed as WASM modules.
package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WasmSource implements source.Source using a WASM module.
// WASM modules must export the following functions:
//
//   - fetch() -> i32 (ptr to JSON array, call get_result_len() after)
//   - get_result_len() -> i32 (length of last result)
//   - free_result() (free the last result memory)
//   - name() -> i32 (ptr to name string)
//   - get_name_len() -> i32 (length of name)
//
// Optional for writable sources:
//   - write(action_ptr i32, action_len i32, data_ptr i32, data_len i32) -> i32 (0=success, 1=error)
//   - get_error() -> i32 (ptr to error string if write failed)
//   - get_error_len() -> i32 (length of error string)
// Memory offsets for WASM string passing.
// These are fixed offsets used for simple string passing to WASM modules.
// TODO: Implement proper memory allocation via malloc/free exports for
// production use with larger data or concurrent access.
const (
	wasmActionOffset = uint32(1024) // Offset for action string in WASM memory
	wasmDataOffset   = uint32(2048) // Offset for data string in WASM memory
)

type WasmSource struct {
	name     string
	runtime  wazero.Runtime
	module   api.Module
	wasmPath string
	siteDir  string
	mu       sync.RWMutex

	// Cached function exports
	fetchFn        api.Function
	getResultLenFn api.Function
	freeResultFn   api.Function
	writeFn        api.Function
	getErrorFn     api.Function
	getErrorLenFn  api.Function
}

// NewWasmSource creates a new WASM-based source.
// path is the path to the .wasm file (relative to siteDir or absolute).
// initConfig contains initialization parameters to pass to the module.
func NewWasmSource(name, path, siteDir string, initConfig map[string]string) (*WasmSource, error) {
	// Resolve path
	wasmPath := path
	if !filepath.IsAbs(wasmPath) {
		wasmPath = filepath.Join(siteDir, path)
	}

	// Read WASM file
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM file %s: %w", wasmPath, err)
	}

	// Create runtime
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	// Instantiate WASI for system calls
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Compile and instantiate the module
	compiledModule, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to compile WASM module: %w", err)
	}

	// Create module config with optional init data
	// Use WithStartFunctions() to prevent auto-running _start,
	// making this a "reactor" module that stays alive for function calls.
	moduleConfig := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithArgs(name).
		WithStartFunctions()

	// Add init config as environment variables
	for k, v := range initConfig {
		moduleConfig = moduleConfig.WithEnv(k, v)
	}

	module, err := r.InstantiateModule(ctx, compiledModule, moduleConfig)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASM module: %w", err)
	}

	s := &WasmSource{
		name:     name,
		runtime:  r,
		module:   module,
		wasmPath: wasmPath,
		siteDir:  siteDir,
	}

	// Cache function exports
	s.fetchFn = module.ExportedFunction("fetch")
	s.getResultLenFn = module.ExportedFunction("get_result_len")
	s.freeResultFn = module.ExportedFunction("free_result")
	s.writeFn = module.ExportedFunction("write")
	s.getErrorFn = module.ExportedFunction("get_error")
	s.getErrorLenFn = module.ExportedFunction("get_error_len")

	if s.fetchFn == nil {
		r.Close(ctx)
		return nil, fmt.Errorf("WASM module missing required export 'fetch'")
	}
	if s.getResultLenFn == nil {
		r.Close(ctx)
		return nil, fmt.Errorf("WASM module missing required export 'get_result_len'")
	}

	return s, nil
}

// Fetch retrieves data from the WASM source.
func (s *WasmSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Call fetch() to get pointer to result
	results, err := s.fetchFn.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("WASM fetch failed [%s]: %w", s.wasmPath, err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("WASM fetch returned no pointer [%s]", s.wasmPath)
	}
	resultPtr := uint32(results[0])

	// Get result length
	lenResults, err := s.getResultLenFn.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("WASM get_result_len failed [%s]: %w", s.wasmPath, err)
	}
	if len(lenResults) == 0 {
		return nil, fmt.Errorf("WASM get_result_len returned no value [%s]", s.wasmPath)
	}
	resultLen := uint32(lenResults[0])

	if resultLen == 0 {
		return []map[string]interface{}{}, nil
	}

	// Read result from module memory
	memory := s.module.Memory()
	if memory == nil {
		return nil, fmt.Errorf("WASM module has no memory export")
	}

	resultBytes, ok := memory.Read(resultPtr, resultLen)
	if !ok {
		return nil, fmt.Errorf("failed to read WASM memory at ptr=%d len=%d", resultPtr, resultLen)
	}

	// Free result memory if function exists
	if s.freeResultFn != nil {
		_, _ = s.freeResultFn.Call(ctx)
	}

	// Parse JSON result
	var data []map[string]interface{}
	if err := json.Unmarshal(resultBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse WASM result as JSON: %w", err)
	}

	return data, nil
}

// WriteItem writes data to the WASM source (if supported).
func (s *WasmSource) WriteItem(ctx context.Context, action string, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writeFn == nil {
		return fmt.Errorf("WASM source %q does not support write operations", s.name)
	}

	// Serialize data to JSON
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	// Allocate memory for action and data strings
	memory := s.module.Memory()
	if memory == nil {
		return fmt.Errorf("WASM module has no memory export [%s]", s.wasmPath)
	}

	// Use fixed memory offsets for simple string passing
	// See TODO in constants for future malloc-based implementation
	actionBytes := []byte(action)
	actionPtr := wasmActionOffset
	dataPtr := wasmDataOffset

	if !memory.Write(actionPtr, actionBytes) {
		return fmt.Errorf("failed to write action to WASM memory [%s]", s.wasmPath)
	}
	if !memory.Write(dataPtr, dataBytes) {
		return fmt.Errorf("failed to write data to WASM memory [%s]", s.wasmPath)
	}

	// Call write(action_ptr, action_len, data_ptr, data_len)
	results, err := s.writeFn.Call(ctx,
		uint64(actionPtr), uint64(len(actionBytes)),
		uint64(dataPtr), uint64(len(dataBytes)))
	if err != nil {
		return fmt.Errorf("WASM write failed [%s]: %w", s.wasmPath, err)
	}

	// Check return value (0 = success)
	if len(results) > 0 && results[0] != 0 {
		// Try to get error message
		if s.getErrorFn != nil && s.getErrorLenFn != nil {
			errPtrResults, _ := s.getErrorFn.Call(ctx)
			errLenResults, _ := s.getErrorLenFn.Call(ctx)
			if len(errPtrResults) > 0 && len(errLenResults) > 0 {
				errPtr := uint32(errPtrResults[0])
				errLen := uint32(errLenResults[0])
				if errBytes, ok := memory.Read(errPtr, errLen); ok {
					return fmt.Errorf("WASM write error [%s]: %s", s.wasmPath, string(errBytes))
				}
			}
		}
		return fmt.Errorf("WASM write returned error code %d [%s]", results[0], s.wasmPath)
	}

	return nil
}

// IsReadonly returns whether this source supports write operations.
func (s *WasmSource) IsReadonly() bool {
	return s.writeFn == nil
}

// Close releases all resources held by the WASM runtime.
func (s *WasmSource) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.runtime != nil {
		return s.runtime.Close(context.Background())
	}
	return nil
}

// Name returns the source name.
func (s *WasmSource) Name() string {
	return s.name
}
