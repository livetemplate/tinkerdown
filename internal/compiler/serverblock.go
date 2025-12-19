package compiler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/livetemplate/livepage"
	"github.com/livetemplate/livepage/internal/config"
	livepageplugin "github.com/livetemplate/livepage/plugin"
)

// Store is a local interface for state objects that can handle actions.
// User state types implement this implicitly through method dispatch:
// - Action methods like Increment(), Decrement() are called via reflection-based dispatch
// - The HandleAction method is used by adapters to forward actions
type Store interface {
	HandleAction(action string, data map[string]interface{}) error
	// Close releases resources. Optional - returns nil if not implemented.
	Close() error
}

// ServerBlockCompiler compiles server blocks into loadable Go plugins
type ServerBlockCompiler struct {
	buildDir string
	debug    bool
}

// NewServerBlockCompiler creates a new compiler for server blocks
func NewServerBlockCompiler(debug bool) *ServerBlockCompiler {
	// Create temp directory for build artifacts
	buildDir := filepath.Join(os.TempDir(), "livepage-builds")
	os.MkdirAll(buildDir, 0755)

	return &ServerBlockCompiler{
		buildDir: buildDir,
		debug:    debug,
	}
}

// CompileServerBlock compiles a server block to a Go plugin and returns a state factory function
func (c *ServerBlockCompiler) CompileServerBlock(block *livepage.ServerBlock) (func() Store, error) {
	if c.debug {
		fmt.Printf("[Compiler] Compiling server block: %s\n", block.ID)
		fmt.Printf("[Compiler] Block content length: %d bytes\n", len(block.Content))
		fmt.Printf("[Compiler] Block content:\n%s\n", block.Content)
	}

	// Create plugin directory
	pluginDir := filepath.Join(c.buildDir, block.ID)
	os.MkdirAll(pluginDir, 0755)

	// Write the server block code to a Go file with plugin exports
	sourceFile := filepath.Join(pluginDir, "main.go")
	pluginCode := c.generatePluginCode(block)

	if err := os.WriteFile(sourceFile, []byte(pluginCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	// Copy go.mod and go.sum from livepage to ensure identical dependencies
	livepageDir := c.findLivepageModule()

	// Read livepage's go.mod
	livepageGoMod, err := os.ReadFile(filepath.Join(livepageDir, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("failed to read livepage go.mod: %w", err)
	}

	// Read livepage's go.sum
	livepageGoSum, err := os.ReadFile(filepath.Join(livepageDir, "go.sum"))
	if err != nil {
		return nil, fmt.Errorf("failed to read livepage go.sum: %w", err)
	}

	// Copy go.sum as-is
	goSumFile := filepath.Join(pluginDir, "go.sum")
	if err := os.WriteFile(goSumFile, livepageGoSum, 0644); err != nil {
		return nil, fmt.Errorf("failed to write go.sum: %w", err)
	}

	// Update module name in go.mod but keep all dependencies
	lines := strings.Split(string(livepageGoMod), "\n")
	if len(lines) > 0 {
		lines[0] = "module " + block.ID // Replace module name
	}

	// Check if we're in a Go workspace and add replace directives if needed
	workspacePath := c.findGoWorkspace()
	if workspacePath != "" {
		if c.debug {
			fmt.Printf("[Compiler] Found go.work at: %s\n", workspacePath)
		}

		// Parse go.work to find local module replacements
		workContent, err := os.ReadFile(workspacePath)
		if err == nil {
			// Add replace directives from workspace
			workLines := strings.Split(string(workContent), "\n")
			var replaceDirectives []string
			inUse := false

			for _, line := range workLines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "use (" {
					inUse = true
					continue
				}
				if inUse && trimmed == ")" {
					inUse = false
					continue
				}
				if inUse && trimmed != "" {
					// Convert "use" directive to "replace" directive
					// Extract module path from directory
					usePath := strings.TrimSpace(trimmed)
					if !strings.HasPrefix(usePath, "./") && !strings.HasPrefix(usePath, "../") {
						usePath = "./" + usePath
					}

					// Resolve relative to workspace file location
					workspaceDir := filepath.Dir(workspacePath)
					absPath := filepath.Join(workspaceDir, usePath)

					// Read the go.mod in that directory to get module name
					modContent, err := os.ReadFile(filepath.Join(absPath, "go.mod"))
					if err == nil {
						modLines := strings.Split(string(modContent), "\n")
						for _, ml := range modLines {
							if strings.HasPrefix(strings.TrimSpace(ml), "module ") {
								moduleName := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ml), "module "))
								replaceDirectives = append(replaceDirectives, fmt.Sprintf("replace %s => %s", moduleName, absPath))
								if c.debug {
									fmt.Printf("[Compiler] Added replace: %s => %s\n", moduleName, absPath)
								}
								break
							}
						}
					}
				}
			}

			// Add replace directives to go.mod
			if len(replaceDirectives) > 0 {
				lines = append(lines, "")
				lines = append(lines, replaceDirectives...)
			}

			// Check if the generated code imports components/datatable
			// and add a require statement if needed
			for _, rd := range replaceDirectives {
				if strings.Contains(rd, "github.com/livetemplate/components") {
					// Add require statement for components (before replace)
					// Find the closing of require block
					for i, line := range lines {
						if strings.TrimSpace(line) == ")" && i > 0 {
							// Check if previous lines look like require block content
							if strings.Contains(lines[i-1], "github.com") || strings.Contains(lines[i-1], "gopkg.in") {
								// Insert require before the closing paren
								lines = append(lines[:i+1], lines[i:]...)
								lines[i] = "\tgithub.com/livetemplate/components v0.0.0"
								break
							}
						}
					}
					break
				}
			}
		}
	}

	updatedGoMod := strings.Join(lines, "\n")

	modFile := filepath.Join(pluginDir, "go.mod")
	if err := os.WriteFile(modFile, []byte(updatedGoMod), 0644); err != nil {
		return nil, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Create a temporary go.work file in the plugin directory with absolute paths
	// This ensures the plugin uses the exact same workspace modules as the main binary
	if workspacePath != "" {
		workContent, err := os.ReadFile(workspacePath)
		if err == nil {
			workspaceDir := filepath.Dir(workspacePath)

			// Convert relative paths to absolute paths
			workLines := strings.Split(string(workContent), "\n")
			var updatedWorkLines []string
			inUse := false

			for _, line := range workLines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "use (" {
					inUse = true
					updatedWorkLines = append(updatedWorkLines, line)
					continue
				}
				if inUse && trimmed == ")" {
					// Add the plugin directory itself before closing paren
					updatedWorkLines = append(updatedWorkLines, "\t"+pluginDir)
					inUse = false
					updatedWorkLines = append(updatedWorkLines, line)
					continue
				}
				if inUse && trimmed != "" {
					// Convert relative path to absolute
					usePath := strings.TrimSpace(trimmed)
					absPath := filepath.Join(workspaceDir, usePath)
					updatedWorkLines = append(updatedWorkLines, "\t"+absPath)
				} else {
					updatedWorkLines = append(updatedWorkLines, line)
				}
			}

			updatedWorkContent := strings.Join(updatedWorkLines, "\n")
			pluginWorkFile := filepath.Join(pluginDir, "go.work")
			if err := os.WriteFile(pluginWorkFile, []byte(updatedWorkContent), 0644); err != nil {
				return nil, fmt.Errorf("failed to write go.work: %w", err)
			}
			if c.debug {
				fmt.Printf("[Compiler] Created go.work in plugin directory with absolute paths\n")
			}
		}
	}

	if c.debug {
		fmt.Printf("[Compiler] Copied go.mod and go.sum from livepage\n")
		fmt.Printf("[Compiler] Plugin will use same livetemplate version as main binary\n")
	}

	// Build the plugin as a standalone executable
	pluginFile := filepath.Join(pluginDir, block.ID)
	cmd := exec.Command("go", "build", "-o", pluginFile, sourceFile)
	cmd.Dir = pluginDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("plugin build failed: %w\nOutput: %s", err, output)
	}

	if c.debug {
		fmt.Printf("[Compiler] Plugin built: %s\n", pluginFile)
	}

	// Return a factory function that creates a new RPC client for each instance
	factory := func() Store {
		return c.createRPCClient(pluginFile, block.ID)
	}

	return factory, nil
}

// generatePluginCode wraps the server block code in RPC plugin boilerplate
func (c *ServerBlockCompiler) generatePluginCode(block *livepage.ServerBlock) string {
	// Extract the package declaration and imports from the block content
	lines := strings.Split(block.Content, "\n")
	var imports []string
	var bodyLines []string
	inImport := false
	foundPackage := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip package declaration
		if strings.HasPrefix(trimmed, "package ") {
			foundPackage = true
			continue
		}

		// Handle imports
		if strings.HasPrefix(trimmed, "import") {
			if strings.Contains(trimmed, "(") {
				inImport = true
				continue // Multi-line import start
			}
			// Single-line import
			imports = append(imports, trimmed)
			continue
		}

		if inImport {
			if trimmed == ")" {
				inImport = false
				continue
			}
			if trimmed != "" {
				imports = append(imports, trimmed)
			}
			continue
		}

		// Everything else is body
		if foundPackage || trimmed != "" {
			bodyLines = append(bodyLines, line)
		}
	}

	// Generate plugin code
	var code strings.Builder

	// Package declaration
	code.WriteString("package main\n\n")

	// Imports - add required RPC plugin imports
	code.WriteString("import (\n")
	code.WriteString("\t\"context\"\n")
	code.WriteString("\t\"encoding/json\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"reflect\"\n")
	code.WriteString("\t\"strings\"\n")
	code.WriteString("\t\"github.com/hashicorp/go-plugin\"\n")
	code.WriteString("\t\"github.com/livetemplate/livetemplate\"\n")
	code.WriteString("\tlivepageplugin \"github.com/livetemplate/livepage/plugin\"\n")

	// Add user imports
	for _, imp := range imports {
		// Clean up import line
		imp = strings.TrimPrefix(imp, "import ")
		imp = strings.TrimSpace(imp)
		// Allow imports that start with:
		// - " (regular import like "fmt")
		// - _ (blank import like _ "github.com/...")
		// - letter (aliased import like foo "github.com/...")
		if imp == "" {
			continue // Skip empty lines
		}
		firstChar := imp[0]
		if firstChar != '"' && firstChar != '_' && !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z')) {
			continue // Skip malformed imports
		}
		// Skip duplicates of what we already added (but keep components imports!)
		if strings.Contains(imp, "livetemplate/livetemplate") ||
			strings.Contains(imp, "livetemplate/livepage") ||
			strings.Contains(imp, "go-plugin") {
			continue
		}
		// Skip standard imports that are already in the base import list
		baseImports := []string{`"context"`, `"encoding/json"`, `"fmt"`, `"reflect"`, `"strings"`}
		isDuplicate := false
		for _, base := range baseImports {
			if strings.Contains(imp, base) {
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}
		code.WriteString("\t" + imp + "\n")
	}
	code.WriteString(")\n\n")

	// Original struct and method definitions
	if c.debug {
		fmt.Printf("[Compiler] bodyLines count: %d\n", len(bodyLines))
		if len(bodyLines) > 0 {
			fmt.Printf("[Compiler] First body line: %s\n", bodyLines[0])
			fmt.Printf("[Compiler] Last body line: %s\n", bodyLines[len(bodyLines)-1])
		}
	}
	code.WriteString(strings.Join(bodyLines, "\n"))
	code.WriteString("\n\n")

	// Detect state initialization code
	stateInitResult := c.detectStateInitialization(block.Content)

	// Generate RPC plugin wrapper with local dispatch
	code.WriteString(fmt.Sprintf(`// StatePluginImpl implements the plugin.StatePlugin interface
type StatePluginImpl struct {
	state interface{}
}

func NewStatePluginImpl() *StatePluginImpl {
	return &StatePluginImpl{
		state: %s,
	}
}

func (s *StatePluginImpl) Change(action string, data map[string]interface{}) error {
	ctx := livetemplate.NewContext(context.Background(), action, data)
	// Use local method dispatch - routes action to method by name
	return dispatchAction(s.state, action, ctx)
}

func (s *StatePluginImpl) GetState() (json.RawMessage, error) {
	return json.Marshal(s.state)
}

// dispatchAction routes an action to a method on the state object.
// It looks for methods with signature: func(*Context) error
// Method names are matched case-insensitively with the action name.
func dispatchAction(state interface{}, action string, ctx *livetemplate.Context) error {
	stateVal := reflect.ValueOf(state)
	stateType := stateVal.Type()

	// Try to find matching method (case-insensitive)
	actionLower := strings.ToLower(action)
	for i := 0; i < stateType.NumMethod(); i++ {
		method := stateType.Method(i)
		if strings.ToLower(method.Name) == actionLower {
			// Validate method signature: func(receiver, *Context) error
			methodType := method.Type
			if methodType.NumIn() != 2 || methodType.NumOut() != 1 {
				return fmt.Errorf("method %%s has invalid signature: expected func(*Context) error", method.Name)
			}

			// Found matching method, call it
			methodVal := stateVal.Method(i)
			results := methodVal.Call([]reflect.Value{reflect.ValueOf(ctx)})
			if len(results) == 1 {
				if results[0].IsNil() {
					return nil
				}
				if errVal, ok := results[0].Interface().(error); ok {
					return errVal
				}
				return fmt.Errorf("method %%s does not return error type", method.Name)
			}
			return nil
		}
	}
	return fmt.Errorf("method not found for action: %%s", action)
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: livepageplugin.Handshake,
		Plugins: map[string]plugin.Plugin{
			"state": &livepageplugin.StateRPCPlugin{Impl: NewStatePluginImpl()},
		},
	})
}
`, stateInitResult.code))

	return code.String()
}

// stateInitResult holds the result of state initialization detection
type stateInitResult struct {
	code       string // The initialization code
	hasError   bool   // Whether the constructor returns an error
	funcName   string // Name of the constructor function (if found)
}

// detectStateInitialization tries to find how to initialize the state
// It looks for either a NewXxxState() constructor function or generates an inline constructor
func (c *ServerBlockCompiler) detectStateInitialization(content string) stateInitResult {
	lines := strings.Split(content, "\n")

	// First, try to find a NewXxxState() constructor function
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for func NewXxxState() patterns that return (*State, error)
		if strings.HasPrefix(trimmed, "func New") && strings.Contains(trimmed, "State()") {
			// Check if it returns error
			returnsError := strings.Contains(trimmed, ", error)")

			// Extract function name
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				funcName := strings.TrimSuffix(parts[1], "()")
				if returnsError {
					// Constructor returns (*State, error) - generate error handling code
					return stateInitResult{
						code: fmt.Sprintf(`func() interface{} {
		s, err := %s()
		if err != nil {
			panic(fmt.Sprintf("failed to initialize state: %%v", err))
		}
		return s
	}()`, funcName),
						hasError: true,
						funcName: funcName,
					}
				}
				return stateInitResult{
					code:     funcName + "()",
					hasError: false,
					funcName: funcName,
				}
			}
		}
	}

	// If no constructor found, look for the state type and create inline constructor
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for type XxxState struct
		if strings.HasPrefix(trimmed, "type ") && strings.HasSuffix(trimmed, "State struct {") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				typeName := parts[1]
				return stateInitResult{
					code:     "&" + typeName + "{}",
					hasError: false,
				}
			}
		}
	}

	// Fallback - this should rarely happen
	return stateInitResult{
		code:     "&TodoState{}",
		hasError: false,
	}
}

// findGoWorkspace searches upward from current directory for go.work file
func (c *ServerBlockCompiler) findGoWorkspace() string {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Search upward for go.work
	for {
		workPath := filepath.Join(dir, "go.work")
		if _, err := os.Stat(workPath); err == nil {
			return workPath
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return ""
}

// findLivepageModule searches upward from current directory for livepage's go.mod
func (c *ServerBlockCompiler) findLivepageModule() string {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		if c.debug {
			fmt.Printf("[Compiler] Failed to get working directory: %v\n", err)
		}
		return "."
	}

	// Search upward for go.mod containing livepage module
	for {
		modPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			// Check if this is the livepage module
			content, err := os.ReadFile(modPath)
			if err == nil && strings.Contains(string(content), "module github.com/livetemplate/livepage") {
				if c.debug {
					fmt.Printf("[Compiler] Found livepage module at: %s\n", dir)
				}
				return dir
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	// Fallback to current directory
	if c.debug {
		fmt.Printf("[Compiler] Could not find livepage go.mod, using current directory\n")
	}
	return "."
}

// createRPCClient creates an RPC client wrapper that implements Store interface
func (c *ServerBlockCompiler) createRPCClient(pluginPath string, blockID string) Store {
	// Create an exec.Command to start the plugin
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: livepageplugin.Handshake,
		Plugins:         livepageplugin.PluginMap,
		Cmd:             exec.Command(pluginPath),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
		},
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		if c.debug {
			fmt.Printf("[Compiler] Failed to connect to plugin %s: %v\n", blockID, err)
		}
		// Return a stub that will error on use
		return &errorStore{err: err}
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("state")
	if err != nil {
		if c.debug {
			fmt.Printf("[Compiler] Failed to dispense plugin %s: %v\n", blockID, err)
		}
		client.Kill()
		return &errorStore{err: err}
	}

	// Cast to StatePlugin interface
	statePlugin := raw.(livepageplugin.StatePlugin)

	// Return an adapter that implements Store interface
	return &rpcStoreAdapter{
		plugin: statePlugin,
		client: client,
		debug:  c.debug,
	}
}

// rpcStoreAdapter adapts the RPC plugin to be used with livetemplate's method dispatch
type rpcStoreAdapter struct {
	plugin livepageplugin.StatePlugin
	client *plugin.Client
	debug  bool
}

// HandleAction forwards action to the RPC plugin
func (a *rpcStoreAdapter) HandleAction(action string, data map[string]interface{}) error {
	return a.plugin.Change(action, data)
}

// Close kills the RPC client process
func (a *rpcStoreAdapter) Close() error {
	if a.client != nil {
		a.client.Kill()
	}
	return nil
}

// GetStateAsInterface fetches the current state from the plugin as a generic interface{}
// This is used for template rendering
func (a *rpcStoreAdapter) GetStateAsInterface() (interface{}, error) {
	jsonData, err := a.plugin.GetState()
	if err != nil {
		return nil, err
	}

	// Unmarshal to a map first
	var rawMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Convert map keys to capitalize first letter (for template compatibility)
	state := capitalizeMapKeys(rawMap)

	return state, nil
}

// capitalizeMapKeys recursively capitalizes the first letter of all keys in a map
// This makes JSON data compatible with Go template field access (which expects capitalized fields)
func capitalizeMapKeys(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		// Capitalize first letter
		newKey := strings.ToUpper(k[:1]) + k[1:]

		// Recursively handle nested maps and slices
		switch val := v.(type) {
		case map[string]interface{}:
			result[newKey] = capitalizeMapKeys(val)
		case []interface{}:
			newSlice := make([]interface{}, len(val))
			for i, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					newSlice[i] = capitalizeMapKeys(itemMap)
				} else {
					newSlice[i] = convertNumber(item)
				}
			}
			result[newKey] = newSlice
		case float64:
			// JSON unmarshaling converts all numbers to float64
			// Convert whole numbers back to int for template compatibility
			result[newKey] = convertNumber(val)
		default:
			result[newKey] = v
		}
	}
	return result
}

// convertNumber converts float64 values that are whole numbers to int
func convertNumber(v interface{}) interface{} {
	if f, ok := v.(float64); ok {
		if f == float64(int(f)) {
			return int(f)
		}
	}
	return v
}

// errorStore is a stub that always returns an error
type errorStore struct {
	err error
}

// HandleAction always returns the stored error
func (e *errorStore) HandleAction(_ string, _ map[string]interface{}) error {
	return e.err
}

// Close is a no-op for error store
func (e *errorStore) Close() error {
	return nil
}

// Cleanup removes build artifacts
func (c *ServerBlockCompiler) Cleanup() {
	os.RemoveAll(c.buildDir)
}

// CompileAutoPersist compiles an auto-persist form into a Go plugin
// This generates all the boilerplate code (State struct, NewState, action methods)
// based on form fields extracted from the LVT template
func (c *ServerBlockCompiler) CompileAutoPersist(blockID string, lvtContent string, siteDBPath string) (func() Store, error) {
	if c.debug {
		fmt.Printf("[Compiler] Compiling auto-persist block: %s\n", blockID)
	}

	// Parse form fields from LVT content
	config, err := ParseFormFields(lvtContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse form fields: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("no lvt-persist form found in block %s", blockID)
	}

	if c.debug {
		fmt.Printf("[Compiler] Found %d form fields for table '%s'\n", len(config.Fields), config.TableName)
		for _, f := range config.Fields {
			fmt.Printf("[Compiler]   - %s: %s (%s)\n", f.Name, f.Type, f.HTMLType)
		}
	}

	// Generate the server block code
	generatedCode := GenerateAutoPersistCode(config, siteDBPath)

	// Create a virtual ServerBlock with the generated code
	block := &livepage.ServerBlock{
		ID:       blockID,
		Language: "go",
		Content:  generatedCode,
		Metadata: map[string]string{
			"auto-persist": config.TableName,
		},
	}

	// Compile using the standard flow
	return c.CompileServerBlock(block)
}

// CompileLvtSource compiles an lvt-source block into a Go plugin
// This generates code that fetches data from the configured source
func (c *ServerBlockCompiler) CompileLvtSource(blockID string, sourceName string, sourceCfg config.SourceConfig, siteDir string, metadata map[string]string) (func() Store, error) {
	if c.debug {
		fmt.Printf("[Compiler] Compiling lvt-source block: %s (source: %s, type: %s)\n", blockID, sourceName, sourceCfg.Type)
	}

	// Generate code for the source
	// Pass element type and other metadata for component-aware code generation
	generatedCode, err := GenerateLvtSourceCode(sourceName, sourceCfg, siteDir, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate lvt-source code: %w", err)
	}

	if c.debug {
		fmt.Printf("[Compiler] Generated lvt-source code:\n%s\n", generatedCode)
	}

	// Create a synthetic server block with the generated code
	// Preserve all metadata from the original block
	block := &livepage.ServerBlock{
		ID:       blockID,
		Language: "go",
		Content:  generatedCode,
		Metadata: metadata,
	}

	// Compile using the standard flow
	return c.CompileServerBlock(block)
}
