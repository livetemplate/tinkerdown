package server

import (
	"context"
	"encoding/json"
	"flag"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/livetemplate/tinkerdown/internal/source"
)

// updateGolden is a flag to update golden files
var updateGolden = flag.Bool("update-golden", false, "update golden files")

// wsTestClient is a helper for WebSocket protocol testing
type wsTestClient struct {
	conn    *websocket.Conn
	t       *testing.T
	timeout time.Duration
}

// newWSTestClient creates a new WebSocket test client connected to the test server
func newWSTestClient(t *testing.T, server *httptest.Server) *wsTestClient {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	return &wsTestClient{
		conn:    conn,
		t:       t,
		timeout: 1 * time.Second, // Fast timeout for protocol tests
	}
}

// send sends a message envelope to the server
func (c *wsTestClient) send(envelope MessageEnvelope) {
	c.t.Helper()
	data, err := json.Marshal(envelope)
	if err != nil {
		c.t.Fatalf("Failed to marshal message: %v", err)
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.t.Fatalf("Failed to send message: %v", err)
	}
}

// sendJSON sends a raw JSON message
func (c *wsTestClient) sendJSON(msg string) {
	c.t.Helper()
	if err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		c.t.Fatalf("Failed to send message: %v", err)
	}
}

// receive receives a message envelope with timeout
func (c *wsTestClient) receive() (MessageEnvelope, error) {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return MessageEnvelope{}, err
	}
	var envelope MessageEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return MessageEnvelope{}, err
	}
	return envelope, nil
}

// receiveAll receives all pending messages until timeout
func (c *wsTestClient) receiveAll() []MessageEnvelope {
	c.t.Helper()
	var messages []MessageEnvelope
	for {
		msg, err := c.receive()
		if err != nil {
			break
		}
		messages = append(messages, msg)
	}
	return messages
}

// close closes the WebSocket connection
func (c *wsTestClient) close() {
	c.conn.Close()
}

// mockWSSource is a mock source for WebSocket testing
type mockWSSource struct {
	mu       sync.Mutex
	name     string
	data     []map[string]interface{}
	readonly bool
	writes   []mockWSWrite
}

type mockWSWrite struct {
	action string
	data   map[string]interface{}
}

func (m *mockWSSource) Name() string { return m.name }

func (m *mockWSSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid data races
	result := make([]map[string]interface{}, len(m.data))
	for i, item := range m.data {
		copied := make(map[string]interface{})
		for k, v := range item {
			copied[k] = v
		}
		result[i] = copied
	}
	return result, nil
}

func (m *mockWSSource) Close() error { return nil }

func (m *mockWSSource) WriteItem(ctx context.Context, action string, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the write
	m.writes = append(m.writes, mockWSWrite{action: action, data: data})

	// Apply the action to simulate state changes
	switch action {
	case "toggle":
		if id, ok := data["id"]; ok {
			for i, item := range m.data {
				if item["id"] == id {
					if done, ok := item["done"].(bool); ok {
						m.data[i]["done"] = !done
					}
				}
			}
		}
	case "add":
		m.data = append(m.data, data)
	case "delete":
		if id, ok := data["id"]; ok {
			for i, item := range m.data {
				if item["id"] == id {
					m.data = append(m.data[:i], m.data[i+1:]...)
					break
				}
			}
		}
	}
	return nil
}

func (m *mockWSSource) IsReadonly() bool { return m.readonly }

func (m *mockWSSource) getWrites() []mockWSWrite {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]mockWSWrite, len(m.writes))
	copy(result, m.writes)
	return result
}

// Ensure mockWSSource implements WritableSource
var _ source.WritableSource = (*mockWSSource)(nil)

// Golden file helpers

const goldenDir = "testdata/ws_golden"

// loadGolden loads a golden file
func loadGolden(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(goldenDir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("Failed to read golden file %s: %v", path, err)
	}
	return data
}

// saveGolden saves a golden file
func saveGolden(t *testing.T, name string, data []byte) {
	t.Helper()
	path := filepath.Join(goldenDir, name+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create golden dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write golden file: %v", err)
	}
}

// compareGolden compares data against a golden file
func compareGolden(t *testing.T, name string, got interface{}) {
	t.Helper()

	gotJSON, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	if *updateGolden {
		saveGolden(t, name, gotJSON)
		return
	}

	wantJSON := loadGolden(t, name)
	if wantJSON == nil {
		t.Fatalf("Golden file %s does not exist. Run with -update-golden to create it.", name)
	}

	// Normalize JSON for comparison
	var gotNorm, wantNorm interface{}
	json.Unmarshal(gotJSON, &gotNorm)
	json.Unmarshal(wantJSON, &wantNorm)

	gotNormJSON, _ := json.MarshalIndent(gotNorm, "", "  ")
	wantNormJSON, _ := json.MarshalIndent(wantNorm, "", "  ")

	if string(gotNormJSON) != string(wantNormJSON) {
		t.Errorf("Golden file mismatch for %s:\ngot:\n%s\n\nwant:\n%s", name, gotNormJSON, wantNormJSON)
	}
}

// Test message envelope structure
func TestMessageEnvelopeStructure(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(t *testing.T, env MessageEnvelope)
	}{
		{
			name:    "valid toggle action",
			input:   `{"blockID":"lvt-0","action":"Toggle","data":{"id":"1"}}`,
			wantErr: false,
			validate: func(t *testing.T, env MessageEnvelope) {
				if env.BlockID != "lvt-0" {
					t.Errorf("BlockID = %q, want %q", env.BlockID, "lvt-0")
				}
				if env.Action != "Toggle" {
					t.Errorf("Action = %q, want %q", env.Action, "Toggle")
				}
			},
		},
		{
			name:    "valid add action",
			input:   `{"blockID":"lvt-0","action":"Add","data":{"text":"New task"}}`,
			wantErr: false,
			validate: func(t *testing.T, env MessageEnvelope) {
				if env.Action != "Add" {
					t.Errorf("Action = %q, want %q", env.Action, "Add")
				}
			},
		},
		{
			name:    "valid delete action",
			input:   `{"blockID":"lvt-0","action":"Delete","data":{"id":"1"}}`,
			wantErr: false,
			validate: func(t *testing.T, env MessageEnvelope) {
				if env.Action != "Delete" {
					t.Errorf("Action = %q, want %q", env.Action, "Delete")
				}
			},
		},
		{
			name:    "action with exec metadata",
			input:   `{"blockID":"exec-0","action":"tree","data":{},"execMeta":{"status":"success","duration":100}}`,
			wantErr: false,
			validate: func(t *testing.T, env MessageEnvelope) {
				if env.ExecMeta == nil {
					t.Error("ExecMeta should not be nil")
					return
				}
				if env.ExecMeta.Status != "success" {
					t.Errorf("ExecMeta.Status = %q, want %q", env.ExecMeta.Status, "success")
				}
				if env.ExecMeta.Duration != 100 {
					t.Errorf("ExecMeta.Duration = %d, want %d", env.ExecMeta.Duration, 100)
				}
			},
		},
		{
			name:    "action with cache metadata",
			input:   `{"blockID":"cached-0","action":"tree","data":{},"cacheMeta":{"cached":true,"age":"5m","stale":false}}`,
			wantErr: false,
			validate: func(t *testing.T, env MessageEnvelope) {
				if env.CacheMeta == nil {
					t.Error("CacheMeta should not be nil")
					return
				}
				if !env.CacheMeta.Cached {
					t.Error("CacheMeta.Cached should be true")
				}
				if env.CacheMeta.Age != "5m" {
					t.Errorf("CacheMeta.Age = %q, want %q", env.CacheMeta.Age, "5m")
				}
			},
		},
		{
			name:    "malformed JSON",
			input:   `{"blockID":"lvt-0","action":"Toggle",`,
			wantErr: true,
		},
		{
			name:    "empty JSON",
			input:   `{}`,
			wantErr: false,
			validate: func(t *testing.T, env MessageEnvelope) {
				if env.BlockID != "" {
					t.Errorf("BlockID = %q, want empty", env.BlockID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var env MessageEnvelope
			err := json.Unmarshal([]byte(tt.input), &env)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, env)
			}
		})
	}
}

// TestMessageEnvelopeSerialization tests JSON round-trip serialization
func TestMessageEnvelopeSerialization(t *testing.T) {
	original := MessageEnvelope{
		BlockID: "test-block",
		Action:  "tree",
		Data:    json.RawMessage(`{"items":[1,2,3]}`),
		ExecMeta: &ExecMeta{
			Status:   "success",
			Duration: 150,
			Output:   "Hello",
			Command:  "echo Hello",
		},
		CacheMeta: &CacheMeta{
			Cached:    true,
			Age:       "10s",
			ExpiresIn: "50s",
			Stale:     false,
		},
	}

	// Serialize
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Deserialize
	var restored MessageEnvelope
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Compare
	if restored.BlockID != original.BlockID {
		t.Errorf("BlockID mismatch: got %q, want %q", restored.BlockID, original.BlockID)
	}
	if restored.Action != original.Action {
		t.Errorf("Action mismatch: got %q, want %q", restored.Action, original.Action)
	}
	if string(restored.Data) != string(original.Data) {
		t.Errorf("Data mismatch: got %s, want %s", restored.Data, original.Data)
	}
	if restored.ExecMeta.Status != original.ExecMeta.Status {
		t.Errorf("ExecMeta.Status mismatch")
	}
	if restored.CacheMeta.Age != original.CacheMeta.Age {
		t.Errorf("CacheMeta.Age mismatch")
	}
}

// TestExpressionsBlockID verifies the constant matches expected value
func TestExpressionsBlockID(t *testing.T) {
	if ExpressionsBlockID != "__expressions__" {
		t.Errorf("ExpressionsBlockID = %q, want %q", ExpressionsBlockID, "__expressions__")
	}
}

// Benchmark tests for protocol performance

func BenchmarkMessageEnvelopeMarshal(b *testing.B) {
	env := MessageEnvelope{
		BlockID: "lvt-0",
		Action:  "tree",
		Data:    json.RawMessage(`{"items":[{"id":"1","text":"Task 1","done":false},{"id":"2","text":"Task 2","done":true}]}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(env)
	}
}

func BenchmarkMessageEnvelopeUnmarshal(b *testing.B) {
	data := []byte(`{"blockID":"lvt-0","action":"Toggle","data":{"id":"1"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var env MessageEnvelope
		json.Unmarshal(data, &env)
	}
}

// TestProtocolPerformance verifies message processing is fast
func TestProtocolPerformance(t *testing.T) {
	// This test verifies that basic protocol operations complete quickly
	// Target: < 50ms for 1000 message encode/decode cycles
	start := time.Now()

	iterations := 1000
	for i := 0; i < iterations; i++ {
		// Simulate message send
		env := MessageEnvelope{
			BlockID: "lvt-0",
			Action:  "Toggle",
			Data:    json.RawMessage(`{"id":"1"}`),
		}
		data, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		// Simulate message receive
		var received MessageEnvelope
		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
	}

	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Errorf("Protocol operations too slow: %v for %d iterations (target: <50ms)", elapsed, iterations)
	}
	t.Logf("Protocol performance: %d iterations in %v (%.2f µs/op)", iterations, elapsed, float64(elapsed.Microseconds())/float64(iterations))
}

// TestConcurrentMessageParsing verifies thread-safe message handling
func TestConcurrentMessageParsing(t *testing.T) {
	messages := []string{
		`{"blockID":"lvt-0","action":"Toggle","data":{"id":"1"}}`,
		`{"blockID":"lvt-1","action":"Add","data":{"text":"New"}}`,
		`{"blockID":"lvt-2","action":"Delete","data":{"id":"2"}}`,
		`{"blockID":"lvt-0","action":"Refresh","data":{}}`,
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Spawn multiple concurrent parsers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				msg := messages[j%len(messages)]
				var env MessageEnvelope
				if err := json.Unmarshal([]byte(msg), &env); err != nil {
					errors <- err
				}
				// Verify parsed correctly
				if env.BlockID == "" {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		t.Errorf("Got %d errors during concurrent parsing: %v", len(errs), errs[0])
	}
}

// TestActionDataParsing tests parsing of action-specific data payloads
func TestActionDataParsing(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		data     string
		validate func(t *testing.T, data map[string]interface{})
	}{
		{
			name:   "toggle with id",
			action: "Toggle",
			data:   `{"id":"task-1"}`,
			validate: func(t *testing.T, data map[string]interface{}) {
				if data["id"] != "task-1" {
					t.Errorf("id = %v, want %q", data["id"], "task-1")
				}
			},
		},
		{
			name:   "add with text",
			action: "Add",
			data:   `{"text":"New task","priority":"high"}`,
			validate: func(t *testing.T, data map[string]interface{}) {
				if data["text"] != "New task" {
					t.Errorf("text = %v, want %q", data["text"], "New task")
				}
				if data["priority"] != "high" {
					t.Errorf("priority = %v, want %q", data["priority"], "high")
				}
			},
		},
		{
			name:   "delete with id",
			action: "Delete",
			data:   `{"id":"task-2"}`,
			validate: func(t *testing.T, data map[string]interface{}) {
				if data["id"] != "task-2" {
					t.Errorf("id = %v, want %q", data["id"], "task-2")
				}
			},
		},
		{
			name:   "filter with expression",
			action: "Filter",
			data:   `{"filter":"done"}`,
			validate: func(t *testing.T, data map[string]interface{}) {
				if data["filter"] != "done" {
					t.Errorf("filter = %v, want %q", data["filter"], "done")
				}
			},
		},
		{
			name:   "sort with column",
			action: "Sort",
			data:   `{"column":"priority"}`,
			validate: func(t *testing.T, data map[string]interface{}) {
				if data["column"] != "priority" {
					t.Errorf("column = %v, want %q", data["column"], "priority")
				}
			},
		},
		{
			name:   "empty data",
			action: "Refresh",
			data:   `{}`,
			validate: func(t *testing.T, data map[string]interface{}) {
				if len(data) != 0 {
					t.Errorf("expected empty data, got %d fields", len(data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgJSON := `{"blockID":"lvt-0","action":"` + tt.action + `","data":` + tt.data + `}`

			var env MessageEnvelope
			if err := json.Unmarshal([]byte(msgJSON), &env); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			var data map[string]interface{}
			if err := json.Unmarshal(env.Data, &data); err != nil {
				t.Fatalf("Failed to unmarshal data: %v", err)
			}

			tt.validate(t, data)
		})
	}
}

// TestMalformedMessageHandling tests error handling for invalid messages
func TestMalformedMessageHandling(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "truncated JSON",
			input:   `{"blockID":"lvt-0"`,
			wantErr: true,
		},
		{
			name:    "invalid JSON syntax",
			input:   `{"blockID": "lvt-0", action: "Toggle"}`,
			wantErr: true,
		},
		{
			name:    "wrong type for blockID",
			input:   `{"blockID": 123, "action": "Toggle"}`,
			wantErr: false, // JSON will coerce to string in some cases
		},
		{
			name:    "null data",
			input:   `{"blockID":"lvt-0","action":"Toggle","data":null}`,
			wantErr: false,
		},
		{
			name:    "array instead of object",
			input:   `["blockID","action"]`,
			wantErr: true,
		},
		{
			name:    "deeply nested data",
			input:   `{"blockID":"lvt-0","action":"Custom","data":{"a":{"b":{"c":{"d":"deep"}}}}}`,
			wantErr: false,
		},
		{
			name:    "unicode in data",
			input:   `{"blockID":"lvt-0","action":"Add","data":{"text":"任务 🎉"}}`,
			wantErr: false,
		},
		{
			name:    "special characters in blockID",
			input:   `{"blockID":"lvt-with-special-chars_123","action":"Toggle"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var env MessageEnvelope
			err := json.Unmarshal([]byte(tt.input), &env)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestExecMetaSerialization tests ExecMeta JSON handling
func TestExecMetaSerialization(t *testing.T) {
	tests := []struct {
		name string
		meta ExecMeta
	}{
		{
			name: "minimal",
			meta: ExecMeta{Status: "ready"},
		},
		{
			name: "success with output",
			meta: ExecMeta{
				Status:   "success",
				Duration: 250,
				Output:   "Hello, World!",
				Command:  "echo Hello, World!",
			},
		},
		{
			name: "error with stderr",
			meta: ExecMeta{
				Status:   "error",
				Duration: 50,
				Stderr:   "command not found",
				Command:  "nonexistent",
			},
		},
		{
			name: "running",
			meta: ExecMeta{
				Status:  "running",
				Command: "sleep 10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := json.Marshal(tt.meta)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Deserialize
			var restored ExecMeta
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Compare
			if restored.Status != tt.meta.Status {
				t.Errorf("Status = %q, want %q", restored.Status, tt.meta.Status)
			}
			if restored.Duration != tt.meta.Duration {
				t.Errorf("Duration = %d, want %d", restored.Duration, tt.meta.Duration)
			}
			if restored.Output != tt.meta.Output {
				t.Errorf("Output = %q, want %q", restored.Output, tt.meta.Output)
			}
		})
	}
}

// TestCacheMetaSerialization tests CacheMeta JSON handling
func TestCacheMetaSerialization(t *testing.T) {
	tests := []struct {
		name string
		meta CacheMeta
	}{
		{
			name: "not cached",
			meta: CacheMeta{Cached: false},
		},
		{
			name: "cached fresh",
			meta: CacheMeta{
				Cached:    true,
				Age:       "30s",
				ExpiresIn: "30s",
				Stale:     false,
			},
		},
		{
			name: "stale refreshing",
			meta: CacheMeta{
				Cached:     true,
				Age:        "2m",
				Stale:      true,
				Refreshing: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := json.Marshal(tt.meta)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Deserialize
			var restored CacheMeta
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Compare
			if restored.Cached != tt.meta.Cached {
				t.Errorf("Cached = %v, want %v", restored.Cached, tt.meta.Cached)
			}
			if restored.Age != tt.meta.Age {
				t.Errorf("Age = %q, want %q", restored.Age, tt.meta.Age)
			}
			if restored.Stale != tt.meta.Stale {
				t.Errorf("Stale = %v, want %v", restored.Stale, tt.meta.Stale)
			}
			if restored.Refreshing != tt.meta.Refreshing {
				t.Errorf("Refreshing = %v, want %v", restored.Refreshing, tt.meta.Refreshing)
			}
		})
	}
}

// TestGoldenFileInfrastructure tests the golden file comparison helpers
func TestGoldenFileInfrastructure(t *testing.T) {
	// Skip if golden files don't exist yet
	if _, err := os.Stat(goldenDir); os.IsNotExist(err) {
		t.Skip("Golden file directory does not exist yet")
	}

	// Test loading a non-existent golden file
	data := loadGolden(t, "nonexistent-test-file")
	if data != nil {
		t.Error("Expected nil for non-existent file")
	}
}

// TestTreeActionResponse tests the structure of tree update responses
func TestTreeActionResponse(t *testing.T) {
	// A tree response should have the standard envelope structure
	treeResponse := MessageEnvelope{
		BlockID: "lvt-0",
		Action:  "tree",
		Data:    json.RawMessage(`{"s":["static"],"d":[{"v":"value","k":0}]}`),
	}

	data, err := json.Marshal(treeResponse)
	if err != nil {
		t.Fatalf("Failed to marshal tree response: %v", err)
	}

	// Verify it can be parsed back
	var parsed MessageEnvelope
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse tree response: %v", err)
	}

	if parsed.Action != "tree" {
		t.Errorf("Action = %q, want %q", parsed.Action, "tree")
	}

	// Verify the tree data structure
	var treeData struct {
		S []string        `json:"s"`
		D json.RawMessage `json:"d"`
	}
	if err := json.Unmarshal(parsed.Data, &treeData); err != nil {
		t.Fatalf("Failed to parse tree data: %v", err)
	}

	if len(treeData.S) != 1 || treeData.S[0] != "static" {
		t.Errorf("Static parts = %v, want [\"static\"]", treeData.S)
	}
}

// TestExprUpdateResponse tests expression update message structure
func TestExprUpdateResponse(t *testing.T) {
	exprResponse := MessageEnvelope{
		BlockID: ExpressionsBlockID,
		Action:  "expr-update",
		Data:    json.RawMessage(`{"expr-0":5,"expr-1":"hello"}`),
	}

	if exprResponse.BlockID != "__expressions__" {
		t.Errorf("BlockID = %q, want %q", exprResponse.BlockID, "__expressions__")
	}
	if exprResponse.Action != "expr-update" {
		t.Errorf("Action = %q, want %q", exprResponse.Action, "expr-update")
	}

	// Verify expression values can be parsed
	var exprValues map[string]interface{}
	if err := json.Unmarshal(exprResponse.Data, &exprValues); err != nil {
		t.Fatalf("Failed to parse expression data: %v", err)
	}

	if exprValues["expr-0"].(float64) != 5 {
		t.Errorf("expr-0 = %v, want 5", exprValues["expr-0"])
	}
	if exprValues["expr-1"].(string) != "hello" {
		t.Errorf("expr-1 = %v, want \"hello\"", exprValues["expr-1"])
	}
}

// TestGoldenToggleAction tests toggle action against golden file
func TestGoldenToggleAction(t *testing.T) {
	envelope := MessageEnvelope{
		BlockID: "lvt-0",
		Action:  "Toggle",
		Data:    json.RawMessage(`{"id":"task-1"}`),
	}
	compareGolden(t, "toggle_action", envelope)
}

// TestGoldenAddAction tests add action against golden file
func TestGoldenAddAction(t *testing.T) {
	envelope := MessageEnvelope{
		BlockID: "lvt-0",
		Action:  "Add",
		Data:    json.RawMessage(`{"id":"task-new","text":"New task","done":false}`),
	}
	compareGolden(t, "add_action", envelope)
}

// TestGoldenDeleteAction tests delete action against golden file
func TestGoldenDeleteAction(t *testing.T) {
	envelope := MessageEnvelope{
		BlockID: "lvt-0",
		Action:  "Delete",
		Data:    json.RawMessage(`{"id":"task-2"}`),
	}
	compareGolden(t, "delete_action", envelope)
}

// TestGoldenTreeResponse tests tree response against golden file
func TestGoldenTreeResponse(t *testing.T) {
	envelope := MessageEnvelope{
		BlockID: "lvt-0",
		Action:  "tree",
		Data:    json.RawMessage(`{"s":["<div>","</div>"],"d":[{"v":"Task 1","k":0},{"v":"Task 2","k":1}]}`),
	}
	compareGolden(t, "tree_response", envelope)
}

// TestGoldenExprUpdate tests expression update against golden file
func TestGoldenExprUpdate(t *testing.T) {
	envelope := MessageEnvelope{
		BlockID: ExpressionsBlockID,
		Action:  "expr-update",
		Data:    json.RawMessage(`{"total":5,"completed":2,"remaining":3}`),
	}
	compareGolden(t, "expr_update", envelope)
}

// TestGoldenExecMeta tests exec metadata response against golden file
func TestGoldenExecMeta(t *testing.T) {
	envelope := MessageEnvelope{
		BlockID: "exec-0",
		Action:  "tree",
		Data:    json.RawMessage(`{}`),
		ExecMeta: &ExecMeta{
			Status:   "success",
			Duration: 150,
			Output:   "Hello, World!",
			Command:  "echo Hello, World!",
		},
	}
	compareGolden(t, "exec_meta", envelope)
}

// TestGoldenCacheMeta tests cache metadata response against golden file
func TestGoldenCacheMeta(t *testing.T) {
	envelope := MessageEnvelope{
		BlockID: "cached-0",
		Action:  "tree",
		Data:    json.RawMessage(`{}`),
		CacheMeta: &CacheMeta{
			Cached:    true,
			Age:       "30s",
			ExpiresIn: "30s",
			Stale:     false,
			Refreshing: false,
		},
	}
	compareGolden(t, "cache_meta", envelope)
}
