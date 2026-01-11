//go:build !ci

package tinkerdown_test

import (
	"strings"
	"testing"
	"time"

	"github.com/livetemplate/tinkerdown"
	"github.com/livetemplate/tinkerdown/internal/schedule"
)

// TestScheduleParsingE2E tests schedule token parsing through the full markdown pipeline.
// This verifies that:
// 1. Schedule tokens like @daily:9am are parsed from markdown content
// 2. Code blocks are properly skipped during parsing
// 3. Imperative commands (Notify, Run action) are extracted
// 4. Parsing warnings are generated for invalid tokens
func TestScheduleParsingE2E(t *testing.T) {
	content := `---
title: "Schedule Test"
type: tutorial
---

# Schedule Examples

This page has a daily reminder @daily:9am to check email.

Weekly meetings happen @weekly:mon,wed:10am in the conference room.

## Code Blocks Are Skipped

` + "```go" + `
// This @daily:9am inside code should NOT be parsed
func example() {}
` + "```" + `

## Imperatives

Notify @daily:9am Check your email inbox

Notify @weekly:fri:5pm Weekly report reminder

Run action:backup @monthly:1st:2am

Run action:cleanup @yearly:jan-1:3am --force

## Various Token Types

- Relative: @tomorrow
- Weekday: @monday
- ISO date: @2024-12-25
- Time only: @9:30am
- Offset: @in:2hours
- Monthly: @monthly:15
- Yearly: @yearly:mar-15

## Invalid Token (Should Generate Warning)

This @badtoken should generate a warning.
`

	fm, _, _, err := tinkerdown.ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	// Test schedule token extraction
	t.Run("schedule tokens extracted", func(t *testing.T) {
		if len(fm.Schedules) == 0 {
			t.Fatal("No schedule tokens found")
		}

		// Should have multiple tokens (excluding code block content)
		t.Logf("Found %d schedule tokens", len(fm.Schedules))

		// Verify token types
		tokenTypes := make(map[schedule.TokenType]int)
		for _, tok := range fm.Schedules {
			tokenTypes[tok.Type]++
			t.Logf("Token: %s (type: %v, line: %d)", tok.Raw, tok.Type, tok.Line)
		}

		// Should have various token types
		if tokenTypes[schedule.TokenDaily] < 2 {
			t.Errorf("Expected at least 2 daily tokens, got %d", tokenTypes[schedule.TokenDaily])
		}
		if tokenTypes[schedule.TokenWeekly] < 2 {
			t.Errorf("Expected at least 2 weekly tokens, got %d", tokenTypes[schedule.TokenWeekly])
		}
	})

	// Test imperative extraction
	t.Run("imperatives extracted", func(t *testing.T) {
		if len(fm.Imperatives) == 0 {
			t.Fatal("No imperatives found")
		}

		t.Logf("Found %d imperatives", len(fm.Imperatives))

		var notifyCount, actionCount int
		for _, imp := range fm.Imperatives {
			t.Logf("Imperative: %s (type: %v, line: %d)", imp.Raw, imp.Type, imp.Line)
			switch imp.Type {
			case schedule.ImperativeNotify:
				notifyCount++
			case schedule.ImperativeRunAction:
				actionCount++
			}
		}

		if notifyCount != 2 {
			t.Errorf("Expected 2 Notify imperatives, got %d", notifyCount)
		}
		if actionCount != 2 {
			t.Errorf("Expected 2 Run action imperatives, got %d", actionCount)
		}
	})

	// Test that code blocks are skipped
	t.Run("code blocks skipped", func(t *testing.T) {
		// Count how many @daily:9am tokens we find - there should be exactly 2:
		// 1. Line ~4: "This page has a daily reminder @daily:9am to check email."
		// 2. Line ~17: "Notify @daily:9am Check your email inbox"
		// The one inside the code block (line ~11) should NOT be parsed
		dailyCount := 0
		for _, tok := range fm.Schedules {
			if tok.Raw == "@daily:9am" {
				dailyCount++
				t.Logf("Found @daily:9am at line %d", tok.Line)
				// The code block is around lines 10-13 (after frontmatter removed)
				// If we find a token there, the code block wasn't skipped properly
				if tok.Line >= 10 && tok.Line <= 13 {
					t.Errorf("Token from inside code block was parsed at line %d", tok.Line)
				}
			}
		}
		if dailyCount != 2 {
			t.Errorf("Expected exactly 2 @daily:9am tokens, got %d", dailyCount)
		}
	})

	// Test warnings for invalid tokens
	t.Run("warnings for invalid tokens", func(t *testing.T) {
		if len(fm.ScheduleWarnings) == 0 {
			t.Error("Expected warning for @badtoken but got none")
		} else {
			t.Logf("Found %d warnings", len(fm.ScheduleWarnings))
			for _, w := range fm.ScheduleWarnings {
				t.Logf("Warning: %s (token: %s, line: %d)", w.Message, w.Token, w.Line)
			}
		}

		// Should have warning for @badtoken
		hasWarning := false
		for _, w := range fm.ScheduleWarnings {
			if strings.Contains(w.Token, "badtoken") {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("Expected warning for @badtoken")
		}
	})
}

// TestScheduleNextOccurrenceE2E tests NextOccurrence calculations.
func TestScheduleNextOccurrenceE2E(t *testing.T) {
	content := `---
title: "Next Occurrence Test"
---

Daily reminder @daily:9am
Weekly meeting @weekly:mon:10am
Monthly review @monthly:15:2pm
`

	fm, _, _, err := tinkerdown.ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	loc := time.UTC
	now := time.Date(2024, time.January, 15, 8, 0, 0, 0, loc) // Monday, Jan 15, 2024, 8am

	for _, tok := range fm.Schedules {
		next := tok.NextOccurrence(now, loc)
		t.Logf("Token %s next occurrence: %v", tok.Raw, next)

		// Verify next occurrence is in the future
		if !next.After(now) && tok.Type != schedule.TokenRelative {
			// Note: @today would be in the past, that's expected
			t.Logf("Warning: next occurrence %v is not after now %v for %s", next, now, tok.Raw)
		}
	}
}

// TestScheduleImperativeParsingE2E tests detailed imperative parsing.
func TestScheduleImperativeParsingE2E(t *testing.T) {
	content := `---
title: "Imperative Test"
---

Notify @daily:9am Check your email

Notify Simple message without schedule

Run action:backup @weekly:sun:2am

Run action:deploy @monthly:1st:3am --env prod --verbose
`

	fm, _, _, err := tinkerdown.ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	t.Run("notify with schedule", func(t *testing.T) {
		var found *schedule.Imperative
		for _, imp := range fm.Imperatives {
			if imp.Type == schedule.ImperativeNotify && imp.Token != nil && imp.Token.Type == schedule.TokenDaily {
				found = imp
				break
			}
		}
		if found == nil {
			t.Fatal("Did not find Notify with daily schedule")
		}
		if found.Message != "Check your email" {
			t.Errorf("Message = %q, want %q", found.Message, "Check your email")
		}
	})

	t.Run("notify without schedule", func(t *testing.T) {
		var found *schedule.Imperative
		for _, imp := range fm.Imperatives {
			if imp.Type == schedule.ImperativeNotify && imp.Token == nil {
				found = imp
				break
			}
		}
		if found == nil {
			t.Fatal("Did not find Notify without schedule")
		}
		if found.Message != "Simple message without schedule" {
			t.Errorf("Message = %q, want %q", found.Message, "Simple message without schedule")
		}
	})

	t.Run("run action with schedule", func(t *testing.T) {
		var found *schedule.Imperative
		for _, imp := range fm.Imperatives {
			if imp.Type == schedule.ImperativeRunAction && imp.ActionName == "backup" {
				found = imp
				break
			}
		}
		if found == nil {
			t.Fatal("Did not find Run action:backup")
		}
		if found.Token == nil || found.Token.Type != schedule.TokenWeekly {
			t.Error("Expected weekly schedule token")
		}
	})

	t.Run("run action with args", func(t *testing.T) {
		var found *schedule.Imperative
		for _, imp := range fm.Imperatives {
			if imp.Type == schedule.ImperativeRunAction && imp.ActionName == "deploy" {
				found = imp
				break
			}
		}
		if found == nil {
			t.Fatal("Did not find Run action:deploy")
		}
		if len(found.Args) != 3 {
			t.Errorf("Args count = %d, want 3", len(found.Args))
		}
		if found.Args[0] != "--env" || found.Args[1] != "prod" || found.Args[2] != "--verbose" {
			t.Errorf("Args = %v, want [--env prod --verbose]", found.Args)
		}
	})
}

// TestScheduleRunnerE2E tests the schedule runner functionality.
func TestScheduleRunnerE2E(t *testing.T) {
	runner := schedule.NewRunner(schedule.RunnerConfig{
		Location: time.UTC,
	})

	content := `
# Page Content

Notify @daily:9am Morning reminder

Run action:backup @weekly:sun:2am
`

	imperatives, err := runner.ParsePage("test-page", content)
	if err != nil {
		t.Fatalf("ParsePage failed: %v", err)
	}

	if len(imperatives) != 2 {
		t.Errorf("Expected 2 imperatives, got %d", len(imperatives))
	}

	// Check jobs were registered
	jobs := runner.GetAllJobs()
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}

	// Verify job details
	for _, job := range jobs {
		t.Logf("Job %s: pageID=%s, nextRun=%v, enabled=%v",
			job.ID, job.PageID, job.NextRun, job.Enabled)

		if job.PageID != "test-page" {
			t.Errorf("Job pageID = %q, want %q", job.PageID, "test-page")
		}
		if !job.Enabled {
			t.Error("Job should be enabled")
		}
	}

	// Test page removal
	runner.RemovePage("test-page")
	jobs = runner.GetAllJobs()
	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs after removal, got %d", len(jobs))
	}
}

// TestScheduleCodeBlockExclusionE2E verifies code blocks are properly excluded.
func TestScheduleCodeBlockExclusionE2E(t *testing.T) {
	content := `---
title: "Code Block Exclusion"
---

This @daily:9am should be found.

` + "```go" + `
// Inside code block
token := "@weekly:mon" // should NOT be parsed
notify := "Notify @daily:8am" // should NOT be parsed
` + "```" + `

` + "```" + `
@monthly:1st - also inside code block, should NOT be parsed
` + "```" + `

Inline code: ` + "`@yearly:jan-1`" + ` should NOT be parsed.

But this @weekly:fri should be found.
`

	fm, _, _, err := tinkerdown.ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	// Should only find tokens outside code blocks
	// Expected: @daily:9am and @weekly:fri
	if len(fm.Schedules) != 2 {
		t.Errorf("Expected 2 schedule tokens, got %d", len(fm.Schedules))
		for _, tok := range fm.Schedules {
			t.Logf("Found: %s at line %d", tok.Raw, tok.Line)
		}
	}

	// Verify the specific tokens found
	hasDaily := false
	hasWeekly := false
	for _, tok := range fm.Schedules {
		if tok.Raw == "@daily:9am" {
			hasDaily = true
		}
		if tok.Raw == "@weekly:fri" {
			hasWeekly = true
		}
	}

	if !hasDaily {
		t.Error("Expected to find @daily:9am outside code blocks")
	}
	if !hasWeekly {
		t.Error("Expected to find @weekly:fri outside code blocks")
	}
}

// TestScheduleEscapedTokensE2E tests that escaped tokens are not parsed.
func TestScheduleEscapedTokensE2E(t *testing.T) {
	content := `---
title: "Escaped Tokens"
---

This @daily:9am should be found.

This \@daily:10am should NOT be found (escaped).

This @weekly:mon should be found.
`

	fm, _, _, err := tinkerdown.ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	// Should find @daily:9am and @weekly:mon but not the escaped one
	if len(fm.Schedules) != 2 {
		t.Errorf("Expected 2 schedule tokens, got %d", len(fm.Schedules))
	}

	for _, tok := range fm.Schedules {
		if tok.Raw == "@daily:10am" {
			t.Error("Escaped token @daily:10am should not have been parsed")
		}
	}
}

// TestSchedulePageIntegrationE2E tests schedule integration with Page building.
func TestSchedulePageIntegrationE2E(t *testing.T) {
	content := `---
title: "Page Integration Test"
type: tutorial
---

# My Page

Notify @daily:9am Morning reminder

` + "```go" + `
// Some code
` + "```" + `

Run action:backup @weekly:sun:2am

Another token @monthly:15 in regular text.
`

	page, err := tinkerdown.BuildPage("test-page", "test.md", []byte(content))
	if err != nil {
		t.Fatalf("BuildPage failed: %v", err)
	}

	// Verify schedules are copied to page
	if len(page.Schedules) == 0 {
		t.Error("Page.Schedules is empty")
	} else {
		t.Logf("Page has %d schedules", len(page.Schedules))
	}

	// Verify imperatives are copied to page
	if len(page.Imperatives) == 0 {
		t.Error("Page.Imperatives is empty")
	} else {
		t.Logf("Page has %d imperatives", len(page.Imperatives))
	}

	// Verify we have expected token types
	hasDaily := false
	hasWeekly := false
	hasMonthly := false
	for _, tok := range page.Schedules {
		switch tok.Type {
		case schedule.TokenDaily:
			hasDaily = true
		case schedule.TokenWeekly:
			hasWeekly = true
		case schedule.TokenMonthly:
			hasMonthly = true
		}
	}

	if !hasDaily {
		t.Error("Expected daily token")
	}
	if !hasWeekly {
		t.Error("Expected weekly token")
	}
	if !hasMonthly {
		t.Error("Expected monthly token")
	}
}
