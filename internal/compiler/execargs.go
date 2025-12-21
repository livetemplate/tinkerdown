package compiler

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ExecArg represents a parsed command-line argument
type ExecArg struct {
	Name        string `json:"name"`        // "name" or "arg1" for positional
	Label       string `json:"label"`       // Display label (custom or auto-generated)
	Type        string `json:"type"`        // "string", "number", "bool"
	Default     string `json:"default"`     // Default value as string
	Position    int    `json:"position"`    // -1 for named flags, 0+ for positional
	Description string `json:"description"` // From --help introspection
}

// ParseExecCommand parses a command string and extracts the executable and arguments.
// Supports:
//   - Named args: --flag:type value or --flag value (type inferred)
//   - Positional args: label:type:value or label:value or just value
//
// All type hints are stripped from the returned args - they're only used to determine
// the input type for form generation.
func ParseExecCommand(cmd string) (executable string, args []ExecArg, err error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", nil, nil
	}

	executable = parts[0]
	parts = parts[1:]

	positionalIndex := 0

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if strings.HasPrefix(part, "--") {
			// Named flag: --flag:type value or --flag value
			flagPart := strings.TrimPrefix(part, "--")

			// Check for type hint in flag name
			var flagName, typeHint string
			if colonIdx := strings.Index(flagPart, ":"); colonIdx != -1 {
				flagName = flagPart[:colonIdx]
				typeHint = flagPart[colonIdx+1:]
			} else {
				flagName = flagPart
			}

			// Get the value (next part)
			var value string
			if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "--") {
				i++
				value = parts[i]
			}

			// Determine type
			argType := typeHint
			if argType == "" {
				argType = InferType(value)
			}

			args = append(args, ExecArg{
				Name:     flagName,
				Label:    flagName,
				Type:     argType,
				Default:  value,
				Position: -1, // Named flag
			})
		} else {
			// Positional argument: label:type:value or label:value or value
			var name, label, typeHint, value string

			colonCount := strings.Count(part, ":")
			switch colonCount {
			case 2:
				// label:type:value
				parts := strings.SplitN(part, ":", 3)
				label = parts[0]
				typeHint = parts[1]
				value = parts[2]
				name = label
			case 1:
				// label:value (type inferred)
				parts := strings.SplitN(part, ":", 2)
				label = parts[0]
				value = parts[1]
				name = label
			default:
				// Just value
				value = part
				name = positionalArgName(positionalIndex)
				label = positionalArgLabel(positionalIndex)
			}

			// Determine type
			argType := typeHint
			if argType == "" {
				argType = InferType(value)
			}

			args = append(args, ExecArg{
				Name:     name,
				Label:    label,
				Type:     argType,
				Default:  value,
				Position: positionalIndex,
			})
			positionalIndex++
		}
	}

	return executable, args, nil
}

// InferType infers the type of a value based on its content.
// Returns "number", "bool", or "string".
func InferType(value string) string {
	// Check for boolean
	lower := strings.ToLower(value)
	if lower == "true" || lower == "false" {
		return "bool"
	}

	// Check for number (integer or float, including negative)
	numRegex := regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	if numRegex.MatchString(value) {
		return "number"
	}

	return "string"
}

// positionalArgName returns the internal name for a positional argument
func positionalArgName(index int) string {
	return "arg" + strconv.Itoa(index+1)
}

// positionalArgLabel returns the display label for a positional argument
func positionalArgLabel(index int) string {
	return "Arg " + strconv.Itoa(index+1)
}

// IntrospectScript runs the script with --help and parses flag descriptions.
// Returns a map of flag name → description.
// If --help fails or returns no useful info, returns an empty map (no error).
func IntrospectScript(executable, workDir string) map[string]string {
	descriptions := make(map[string]string)

	// Run with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, "--help")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		// Swallow error - --help might not be supported
		return descriptions
	}

	// Parse the output for flag descriptions
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Try to match common --help patterns
		desc := parseHelpLine(line)
		for flag, description := range desc {
			descriptions[flag] = description
		}
	}

	return descriptions
}

// parseHelpLine attempts to extract flag name and description from a help line.
// Supports common patterns:
//   - --flag  Description text
//   - --flag VALUE  Description text
//   - -f, --flag  Description text
//   - --flag=VALUE  Description text
func parseHelpLine(line string) map[string]string {
	result := make(map[string]string)

	// Patterns ordered from most specific to least specific
	patterns := []*regexp.Regexp{
		// -f, --flag VALUE  Description (short + long with placeholder)
		regexp.MustCompile(`^\s*-\w,\s*--([a-zA-Z][-a-zA-Z0-9]*)(?:\s+\S+|\s*=\S+)?\s{2,}(.+)$`),
		// --flag=VALUE  Description
		regexp.MustCompile(`^\s*--([a-zA-Z][-a-zA-Z0-9]*)=\S+\s{2,}(.+)$`),
		// --flag VALUE  Description (placeholder after space)
		regexp.MustCompile(`^\s*--([a-zA-Z][-a-zA-Z0-9]*)\s+[A-Z][A-Z0-9_]*\s{2,}(.+)$`),
		// --flag  Description (at least 2 spaces before description, no placeholder)
		regexp.MustCompile(`^\s*--([a-zA-Z][-a-zA-Z0-9]*)\s{2,}(.+)$`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			flag := matches[1]
			description := strings.TrimSpace(matches[2])
			result[flag] = description
			break
		}
	}

	return result
}

// BuildCommand reconstructs the command from executable, args, and current values.
// This strips all type hints and uses the actual values.
func BuildCommand(executable string, args []ExecArg, argValues map[string]string) []string {
	cmdParts := []string{executable}

	// First add positional args in order
	positionalArgs := make([]ExecArg, 0)
	namedArgs := make([]ExecArg, 0)

	for _, arg := range args {
		if arg.Position >= 0 {
			positionalArgs = append(positionalArgs, arg)
		} else {
			namedArgs = append(namedArgs, arg)
		}
	}

	// Sort positional args by position (they should already be in order, but be safe)
	for _, arg := range positionalArgs {
		val := argValues[arg.Name]
		if val == "" {
			val = arg.Default
		}
		cmdParts = append(cmdParts, val)
	}

	// Add named args
	for _, arg := range namedArgs {
		val := argValues[arg.Name]
		if val == "" {
			val = arg.Default
		}
		cmdParts = append(cmdParts, "--"+arg.Name, val)
	}

	return cmdParts
}
