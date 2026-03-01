package slug

import "testing"

func TestHeading(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Todos", "todos"},
		{"My Task List", "my-task-list"},
		{"Morning Tasks!", "morning-tasks"},
		{"Hello World (v2)", "hello-world-v2"},
		{"café", "caf"},
		{"UPPER CASE", "upper-case"},
		{"already-lowercase", "already-lowercase"},
		{"  spaces  ", "spaces"},
		{"", ""},
	}

	for _, tt := range tests {
		result := Heading(tt.input)
		if result != tt.expected {
			t.Errorf("Heading(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
