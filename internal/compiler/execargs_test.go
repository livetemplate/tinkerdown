package compiler

import (
	"testing"
)

func TestInferType(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		// Numbers
		{"5", "number"},
		{"3.14", "number"},
		{"-10", "number"},
		{"0", "number"},
		{"-3.14", "number"},

		// Booleans
		{"true", "bool"},
		{"false", "bool"},
		{"True", "bool"},
		{"FALSE", "bool"},

		// Strings
		{"hello", "string"},
		{"file.txt", "string"},
		{"", "string"},
		{"hello world", "string"},
		{"123abc", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := InferType(tt.value)
			if result != tt.expected {
				t.Errorf("InferType(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestParseExecCommand(t *testing.T) {
	tests := []struct {
		name       string
		cmd        string
		wantExec   string
		wantArgs   []ExecArg
	}{
		{
			name:     "simple command no args",
			cmd:      "./script.sh",
			wantExec: "./script.sh",
			wantArgs: nil,
		},
		{
			name:     "named args with type hints",
			cmd:      "./script.sh --name:string adnaan --count:number 5",
			wantExec: "./script.sh",
			wantArgs: []ExecArg{
				{Name: "name", Label: "name", Type: "string", Default: "adnaan", Position: -1},
				{Name: "count", Label: "count", Type: "number", Default: "5", Position: -1},
			},
		},
		{
			name:     "named args with type inference",
			cmd:      "./script.sh --name adnaan --count 5 --verbose true",
			wantExec: "./script.sh",
			wantArgs: []ExecArg{
				{Name: "name", Label: "name", Type: "string", Default: "adnaan", Position: -1},
				{Name: "count", Label: "count", Type: "number", Default: "5", Position: -1},
				{Name: "verbose", Label: "verbose", Type: "bool", Default: "true", Position: -1},
			},
		},
		{
			name:     "positional args with labels",
			cmd:      "./script.sh input:string:file.txt output:string:result.json",
			wantExec: "./script.sh",
			wantArgs: []ExecArg{
				{Name: "input", Label: "input", Type: "string", Default: "file.txt", Position: 0},
				{Name: "output", Label: "output", Type: "string", Default: "result.json", Position: 1},
			},
		},
		{
			name:     "positional args without type hints",
			cmd:      "./script.sh input:file.txt",
			wantExec: "./script.sh",
			wantArgs: []ExecArg{
				{Name: "input", Label: "input", Type: "string", Default: "file.txt", Position: 0},
			},
		},
		{
			name:     "positional args auto-labeled",
			cmd:      "./script.sh file.txt",
			wantExec: "./script.sh",
			wantArgs: []ExecArg{
				{Name: "arg1", Label: "Arg 1", Type: "string", Default: "file.txt", Position: 0},
			},
		},
		{
			name:     "mixed positional and named",
			cmd:      "./greet.sh input:string:greeting.txt --name:string World --count 3 --uppercase false",
			wantExec: "./greet.sh",
			wantArgs: []ExecArg{
				{Name: "input", Label: "input", Type: "string", Default: "greeting.txt", Position: 0},
				{Name: "name", Label: "name", Type: "string", Default: "World", Position: -1},
				{Name: "count", Label: "count", Type: "number", Default: "3", Position: -1},
				{Name: "uppercase", Label: "uppercase", Type: "bool", Default: "false", Position: -1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExec, gotArgs, err := ParseExecCommand(tt.cmd)
			if err != nil {
				t.Fatalf("ParseExecCommand() error = %v", err)
			}

			if gotExec != tt.wantExec {
				t.Errorf("executable = %q, want %q", gotExec, tt.wantExec)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("got %d args, want %d args", len(gotArgs), len(tt.wantArgs))
			}

			for i, want := range tt.wantArgs {
				got := gotArgs[i]
				if got.Name != want.Name {
					t.Errorf("arg[%d].Name = %q, want %q", i, got.Name, want.Name)
				}
				if got.Label != want.Label {
					t.Errorf("arg[%d].Label = %q, want %q", i, got.Label, want.Label)
				}
				if got.Type != want.Type {
					t.Errorf("arg[%d].Type = %q, want %q", i, got.Type, want.Type)
				}
				if got.Default != want.Default {
					t.Errorf("arg[%d].Default = %q, want %q", i, got.Default, want.Default)
				}
				if got.Position != want.Position {
					t.Errorf("arg[%d].Position = %d, want %d", i, got.Position, want.Position)
				}
			}
		})
	}
}

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		args       []ExecArg
		argValues  map[string]string
		want       []string
	}{
		{
			name:       "simple named args",
			executable: "./script.sh",
			args: []ExecArg{
				{Name: "name", Type: "string", Default: "World", Position: -1},
				{Name: "count", Type: "number", Default: "3", Position: -1},
			},
			argValues: map[string]string{
				"name":  "Alice",
				"count": "5",
			},
			want: []string{"./script.sh", "--name", "Alice", "--count", "5"},
		},
		{
			name:       "mixed positional and named",
			executable: "./script.sh",
			args: []ExecArg{
				{Name: "input", Type: "string", Default: "in.txt", Position: 0},
				{Name: "output", Type: "string", Default: "out.txt", Position: 1},
				{Name: "verbose", Type: "bool", Default: "false", Position: -1},
			},
			argValues: map[string]string{
				"input":   "data.json",
				"output":  "result.json",
				"verbose": "true",
			},
			want: []string{"./script.sh", "data.json", "result.json", "--verbose", "true"},
		},
		{
			name:       "uses defaults when no value provided",
			executable: "./script.sh",
			args: []ExecArg{
				{Name: "name", Type: "string", Default: "Default", Position: -1},
			},
			argValues: map[string]string{},
			want:      []string{"./script.sh", "--name", "Default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCommand(tt.executable, tt.args, tt.argValues)

			if len(got) != len(tt.want) {
				t.Fatalf("BuildCommand() = %v, want %v", got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("BuildCommand()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseHelpLine(t *testing.T) {
	tests := []struct {
		line string
		want map[string]string
	}{
		{
			line: "  --name NAME      The name to greet",
			want: map[string]string{"name": "The name to greet"},
		},
		{
			line: "  --count=N        Number of repetitions",
			want: map[string]string{"count": "Number of repetitions"},
		},
		{
			line: "  -v, --verbose    Enable verbose output",
			want: map[string]string{"verbose": "Enable verbose output"},
		},
		{
			line: "  --help           Show this help",
			want: map[string]string{"help": "Show this help"},
		},
		{
			line: "This is not a flag line",
			want: map[string]string{},
		},
		{
			line: "",
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := parseHelpLine(tt.line)

			if len(got) != len(tt.want) {
				t.Fatalf("parseHelpLine(%q) = %v, want %v", tt.line, got, tt.want)
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseHelpLine(%q)[%q] = %q, want %q", tt.line, k, got[k], v)
				}
			}
		})
	}
}
