package runtime

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTimestamp(t *testing.T) {
	resolver := NewDefaultResolver("alice")

	before := time.Now().Add(-time.Second) // Allow 1 second tolerance
	result, err := resolver.Resolve("{{timestamp}}")
	require.NoError(t, err)
	after := time.Now().Add(time.Second) // Allow 1 second tolerance

	// Parse the result as RFC3339
	parsed, err := time.Parse(time.RFC3339, result.(string))
	require.NoError(t, err)

	// Verify the timestamp is within reasonable bounds
	assert.True(t, parsed.After(before), "timestamp should be > before-1s")
	assert.True(t, parsed.Before(after), "timestamp should be < after+1s")
}

func TestResolveToday(t *testing.T) {
	resolver := NewDefaultResolver("")

	result, err := resolver.Resolve("{{today}}")
	require.NoError(t, err)

	// Should be in YYYY-MM-DD format
	expected := time.Now().Format("2006-01-02")
	assert.Equal(t, expected, result)
}

func TestResolveUnix(t *testing.T) {
	resolver := NewDefaultResolver("")

	before := time.Now().Unix()
	result, err := resolver.Resolve("{{unix}}")
	require.NoError(t, err)
	after := time.Now().Unix()

	// The result is a string representation of the unix timestamp
	assert.NotEmpty(t, result)

	// Parse the unix timestamp string to int64 and verify it's within range
	resultStr, ok := result.(string)
	require.True(t, ok, "result should be a string")

	var unix int64
	_, parseErr := fmt.Sscanf(resultStr, "%d", &unix)
	require.NoError(t, parseErr, "should parse unix timestamp as int64")

	// The unix value should be between before and after
	assert.GreaterOrEqual(t, unix, before, "unix timestamp should be >= before")
	assert.LessOrEqual(t, unix, after, "unix timestamp should be <= after")
}

func TestResolveOperator(t *testing.T) {
	resolver := NewDefaultResolver("alice")

	result, err := resolver.Resolve("Created by {{.operator}}")
	require.NoError(t, err)
	assert.Equal(t, "Created by alice", result)
}

func TestResolveEmptyOperator(t *testing.T) {
	resolver := NewDefaultResolver("")

	result, err := resolver.Resolve("{{.operator}}")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestResolveNonTemplate(t *testing.T) {
	resolver := NewDefaultResolver("alice")

	// Plain strings without template syntax should be returned unchanged
	result, err := resolver.Resolve("Hello World")
	require.NoError(t, err)
	assert.Equal(t, "Hello World", result)
}

func TestResolveEmptyString(t *testing.T) {
	resolver := NewDefaultResolver("alice")

	result, err := resolver.Resolve("")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestResolveMixedContent(t *testing.T) {
	resolver := NewDefaultResolver("alice")

	result, err := resolver.Resolve("User: {{.operator}}, Date: {{today}}")
	require.NoError(t, err)

	expected := "User: alice, Date: " + time.Now().Format("2006-01-02")
	assert.Equal(t, expected, result)
}

func TestResolveInvalidTemplate(t *testing.T) {
	resolver := NewDefaultResolver("alice")

	_, err := resolver.Resolve("{{invalid}}")
	assert.Error(t, err)
}

func TestResolveMap(t *testing.T) {
	resolver := NewDefaultResolver("alice")

	input := map[string]interface{}{
		"text":       "Buy groceries",
		"priority":   "high",
		"created_by": "{{.operator}}",
		"created_at": "{{timestamp}}",
		"count":      42, // Non-string value should be preserved
	}

	result, err := resolver.ResolveMap(input)
	require.NoError(t, err)

	assert.Equal(t, "Buy groceries", result["text"])
	assert.Equal(t, "high", result["priority"])
	assert.Equal(t, "alice", result["created_by"])
	assert.Equal(t, 42, result["count"])

	// Verify created_at is a valid timestamp
	createdAt, ok := result["created_at"].(string)
	require.True(t, ok)
	_, err = time.Parse(time.RFC3339, createdAt)
	require.NoError(t, err)
}

func TestResolveMapWithNoTemplates(t *testing.T) {
	resolver := NewDefaultResolver("alice")

	input := map[string]interface{}{
		"text":     "Buy groceries",
		"priority": "high",
	}

	result, err := resolver.ResolveMap(input)
	require.NoError(t, err)

	assert.Equal(t, input, result)
}

func TestFormatDate(t *testing.T) {
	now := time.Now()
	result := formatDate(now, "2006-01-02")
	assert.Equal(t, now.Format("2006-01-02"), result)
}

func TestMathFunctions(t *testing.T) {
	resolver := NewDefaultResolver("")

	// Test add
	result, err := resolver.Resolve("{{add 1 2}}")
	require.NoError(t, err)
	assert.Equal(t, "3", result)

	// Test sub
	result, err = resolver.Resolve("{{sub 5 3}}")
	require.NoError(t, err)
	assert.Equal(t, "2", result)
}

func TestOperatorInFormData(t *testing.T) {
	// Simulates using {{.operator}} in form field values
	resolver := NewDefaultResolver("admin")

	// Form field with operator placeholder
	input := map[string]interface{}{
		"title":      "New Task",
		"created_by": "{{.operator}}",
	}

	result, err := resolver.ResolveMap(input)
	require.NoError(t, err)

	assert.Equal(t, "New Task", result["title"])
	assert.Equal(t, "admin", result["created_by"])
}
