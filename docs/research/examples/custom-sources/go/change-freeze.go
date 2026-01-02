// Change freeze checker for tinkerdown
// Checks if deploys are allowed based on calendar rules
//
// Build: go build -o change-freeze change-freeze.go
//
// Usage in markdown:
// ```yaml
// sources:
//   freeze:
//     type: exec
//     command: "./sources/change-freeze"
// ```
//
// <div lvt-if="freeze[0].frozen" class="warning">
//   ⚠️ Change freeze: {{freeze[0].reason}} (until {{freeze[0].until}})
// </div>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Input struct {
	Query map[string]interface{} `json:"query"`
	Env   map[string]string      `json:"env"`
}

type Output struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

// FreezeWindow represents a scheduled change freeze period
type FreezeWindow struct {
	Name   string
	Start  time.Time
	End    time.Time
	Reason string
}

func main() {
	// Read input (we consume it but don't need params for this source)
	var input Input
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		// Allow empty input for simple queries
		if err.Error() != "EOF" {
			fmt.Fprintf(os.Stderr, "Warning: could not parse input: %v\n", err)
		}
	}

	now := time.Now()

	// Define freeze windows (in production, fetch from calendar API)
	freezeWindows := []FreezeWindow{
		{
			Name:   "Holiday Freeze",
			Start:  time.Date(now.Year(), 12, 20, 0, 0, 0, 0, time.Local),
			End:    time.Date(now.Year()+1, 1, 3, 0, 0, 0, 0, time.Local),
			Reason: "Holiday code freeze",
		},
		{
			Name:   "Quarter End",
			Start:  time.Date(now.Year(), 3, 28, 0, 0, 0, 0, time.Local),
			End:    time.Date(now.Year(), 4, 1, 0, 0, 0, 0, time.Local),
			Reason: "Q1 close - no deploys",
		},
		// Weekend freeze (every Saturday/Sunday)
	}

	// Check if currently in a freeze window
	var activeFreeze *FreezeWindow
	for i := range freezeWindows {
		if now.After(freezeWindows[i].Start) && now.Before(freezeWindows[i].End) {
			activeFreeze = &freezeWindows[i]
			break
		}
	}

	// Check weekend freeze
	weekday := now.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	// Build result
	result := map[string]interface{}{
		"frozen":  false,
		"reason":  "",
		"until":   "",
		"weekend": isWeekend,
	}

	if activeFreeze != nil {
		result["frozen"] = true
		result["reason"] = activeFreeze.Reason
		result["until"] = activeFreeze.End.Format("Jan 2, 3:04 PM")
	} else if isWeekend {
		// Optional: treat weekends as soft freeze
		result["frozen"] = false // or true if strict
		result["reason"] = "Weekend - consider waiting for Monday"
		// Calculate next Monday
		daysUntilMonday := (8 - int(weekday)) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		monday := now.AddDate(0, 0, daysUntilMonday)
		monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 9, 0, 0, 0, time.Local)
		result["until"] = monday.Format("Jan 2, 3:04 PM")
	}

	output := Output{
		Columns: []string{"frozen", "reason", "until", "weekend"},
		Rows:    []map[string]interface{}{result},
	}

	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding output: %v\n", err)
		os.Exit(1)
	}
}
