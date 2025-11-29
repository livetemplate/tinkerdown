# Testing RPC Plugin Implementation

## Quick Manual Test

1. **Build the binary:**
   ```bash
   go build -o livepage ./cmd/livepage
   ```

2. **Start the server with debug output:**
   ```bash
   ./livepage serve examples/todos-workshop --debug
   ```

3. **Look for these SUCCESS indicators in the logs:**
   - `[Compiler] Plugin built: /var/folders/.../validation-state/validation-state` ✅
   - `[WS] Successfully compiled block: validation-state` ✅
   - `[DEBUG] plugin: plugin started` ✅
   - `[WS] RPC state for lvt-1: map[...]` ✅

4. **Open browser and test:**
   - Navigate to http://localhost:8080/validation
   - Page should load WITHOUT showing "Connecting..." message
   - You should see a todo list with initial items
   - Click a checkbox - it should toggle
   - Add a new todo - it should appear in the list

## Run Automated E2E Test

```bash
# Clean up first
killall livepage 2>/dev/null
lsof -ti:8080 | xargs kill -9 2>/dev/null

# Run the E2E test
go test -v -run TestInteractiveBlocksE2E -timeout 3m
```

## What Changed (RPC Plugin System)

### Before (Native Go Plugins)
- Used `-buildmode=plugin` to create `.so` shared libraries
- Required EXACT version matching between build and runtime
- Failed with: "plugin was built with a different version of package golang.org/x/sys/cpu"
- Would break when distributing pre-built binaries

### After (RPC Plugins)
- Compiles server blocks into standalone executables
- Uses HashiCorp's go-plugin for RPC communication over Unix sockets
- No version matching requirements - separate processes
- Works with distributed binaries

## Key Files Modified

1. **plugin/interface.go** (NEW - public package)
   - Moved from `internal/plugin` to be importable by compiled plugins
   - Defines `StatePlugin` RPC interface
   - Implements RPC client/server wrappers

2. **internal/compiler/serverblock.go**
   - Changed from `-buildmode=plugin` to regular executable compilation
   - Generates RPC plugin wrapper code
   - Implements `capitalizeMapKeys()` for template compatibility
   - Creates RPC client that implements `livetemplate.Store`

3. **internal/server/websocket.go**
   - Added `StateGetter` interface detection for RPC plugins
   - Fetches state via RPC instead of direct access

## Verifying Plugin Compilation

Check the temp directory for compiled plugins:
```bash
ls -la /var/folders/*/T/livepage-builds/*/
```

You should see:
- `main.go` (generated plugin source)
- `go.mod` and `go.sum` (copied from livepage)
- `validation-state` or `persistent-state` (compiled executable)

## Debugging Tips

1. **Enable debug mode to see RPC traffic:**
   ```bash
   ./livepage serve examples/todos-workshop --debug 2>&1 | grep -E "\[WS\]|\[Compiler\]|plugin:"
   ```

2. **Check if plugin processes are running:**
   ```bash
   ps aux | grep validation-state
   ```

3. **View generated plugin code:**
   ```bash
   cat /var/folders/*/T/livepage-builds/validation-state/main.go
   ```

## Common Issues

### "No compiled factory for state X"
- Plugin compilation failed
- Check debug logs for compilation errors
- Verify go.mod and go.sum are correctly copied

### "Failed to get state"
- RPC communication failed
- Check if plugin process started
- Look for go-plugin debug logs

### Empty/nil state data
- JSON unmarshaling issue
- Check `capitalizeMapKeys` is working correctly
- Verify state implements expected structure
