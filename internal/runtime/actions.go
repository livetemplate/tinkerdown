package runtime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/livetemplate/tinkerdown/internal/source"
)

// buildCommandString rebuilds the command string from executable and current arg values
func buildCommandString(origCmd string, args []Arg) string {
	// Get executable from original command
	parts := strings.Fields(origCmd)
	if len(parts) == 0 {
		return origCmd
	}
	executable := parts[0]

	// Build new command with current arg values
	cmdParts := []string{executable}
	for _, arg := range args {
		cmdParts = append(cmdParts, "--"+arg.Name, arg.Value)
	}
	return strings.Join(cmdParts, " ")
}

// runExec handles the Run action for exec sources
func (s *GenericState) runExec(data map[string]interface{}) error {
	if s.sourceType != "exec" {
		return fmt.Errorf("Run action only valid for exec sources")
	}

	execSrc, ok := s.source.(*source.ExecSource)
	if !ok {
		return fmt.Errorf("invalid exec source")
	}

	s.Status = "running"

	ctx := context.Background()
	var result []map[string]interface{}
	var err error

	// Check if form data was submitted
	if len(data) > 0 {
		// Update Args with submitted values
		for i := range s.Args {
			if val, ok := data[s.Args[i].Name]; ok {
				valStr := fmt.Sprintf("%v", val)
				// Convert checkbox "on" to "true"
				if valStr == "on" {
					valStr = "true"
				}
				s.Args[i].Value = valStr
			}
		}

		// Build args map from submitted data
		argsMap := make(map[string]string)
		for k, v := range data {
			argsMap[k] = fmt.Sprintf("%v", v)
		}

		// Update the Command string to show current argument values
		s.Command = buildCommandString(s.sourceCfg.Cmd, s.Args)

		// Execute with custom arguments
		result, err = execSrc.FetchWithArgs(ctx, argsMap)
	} else {
		// No form data - use default command
		result, err = execSrc.Fetch(ctx)
	}

	if err != nil {
		s.Status = "error"
		s.Error = err.Error()
		return err
	}

	s.Data = result
	s.Status = "success"
	s.Error = ""
	return nil
}

// handleWriteAction handles Add, Toggle, Delete, Update actions for writable sources
func (s *GenericState) handleWriteAction(action string, data map[string]interface{}) error {
	writable, ok := s.source.(source.WritableSource)
	if !ok {
		return fmt.Errorf("source %q does not support write operations", s.sourceName)
	}

	if writable.IsReadonly() {
		return fmt.Errorf("source %q is read-only", s.sourceName)
	}

	// Delegate to the source's WriteItem
	ctx := context.Background()
	if err := writable.WriteItem(ctx, strings.ToLower(action), data); err != nil {
		s.Error = err.Error()
		return err
	}

	// Refresh data after write
	return s.refresh()
}

// handleDatatableAction handles Sort, NextPage, PrevPage actions
func (s *GenericState) handleDatatableAction(action string, data map[string]interface{}) error {
	// Parse action pattern: Sort_<id>, NextPage_<id>, PrevPage_<id>
	// or just Sort, NextPage, PrevPage (without suffix)
	actionLower := strings.ToLower(action)

	var baseAction string
	if strings.HasPrefix(actionLower, "sort") {
		baseAction = "sort"
	} else if strings.HasPrefix(actionLower, "nextpage") {
		baseAction = "nextpage"
	} else if strings.HasPrefix(actionLower, "prevpage") {
		baseAction = "prevpage"
	} else {
		return fmt.Errorf("unknown datatable action: %s", action)
	}

	switch baseAction {
	case "sort":
		// Get column from data
		column, ok := data["column"].(string)
		if !ok {
			// Try to get from action suffix
			if strings.Contains(action, "_") {
				parts := strings.SplitN(action, "_", 2)
				if len(parts) == 2 {
					column = parts[1]
				}
			}
		}
		if column == "" {
			return fmt.Errorf("sort requires column parameter")
		}
		return s.sortData(column)

	case "nextpage":
		// For now, pagination is handled client-side or via datatable component
		// This is a placeholder for future implementation
		return nil

	case "prevpage":
		return nil

	default:
		return fmt.Errorf("unknown datatable action: %s", baseAction)
	}
}

// sortData sorts the data by the given column
func (s *GenericState) sortData(column string) error {
	if len(s.Data) == 0 {
		return nil
	}

	// Check if column exists
	if _, ok := s.Data[0][column]; !ok {
		return fmt.Errorf("column %q not found in data", column)
	}

	// Toggle between ascending and descending based on current order
	ascending := true
	if len(s.Data) >= 2 {
		first := s.Data[0][column]
		last := s.Data[len(s.Data)-1][column]
		ascending = compareValues(first, last) > 0 // If already desc, make it asc
	}

	// Use sort.Slice for O(n log n) performance
	sort.Slice(s.Data, func(i, j int) bool {
		a := s.Data[i][column]
		b := s.Data[j][column]
		cmp := compareValues(a, b)
		if ascending {
			return cmp < 0
		}
		return cmp > 0
	})

	return nil
}

// compareValues compares two interface{} values for sorting
func compareValues(a, b interface{}) int {
	// Handle nil
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Try string comparison
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if aOk && bOk {
		return strings.Compare(aStr, bStr)
	}

	// Try numeric comparison
	aNum := toFloat64(a)
	bNum := toFloat64(b)
	if aNum < bNum {
		return -1
	}
	if aNum > bNum {
		return 1
	}
	return 0
}

// toFloat64 converts an interface{} to float64
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	default:
		return 0
	}
}
