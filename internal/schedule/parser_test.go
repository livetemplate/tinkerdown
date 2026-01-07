package schedule

import (
	"testing"
	"time"
)

func TestParseRelativeDates(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	tests := []struct {
		input    string
		wantType TokenType
	}{
		{"@today", TokenRelative},
		{"@tomorrow", TokenRelative},
		{"@yesterday", TokenRelative},
		{"@TODAY", TokenRelative}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != tt.wantType {
				t.Errorf("ParseToken(%q) type = %v, want %v", tt.input, token.Type, tt.wantType)
			}
			if token.Date == nil {
				t.Errorf("ParseToken(%q) date is nil", tt.input)
			}
		})
	}
}

func TestParseWeekdays(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	weekdays := []string{
		"@monday", "@tuesday", "@wednesday", "@thursday",
		"@friday", "@saturday", "@sunday",
		"@mon", "@tue", "@wed", "@thu", "@fri", "@sat", "@sun",
	}

	for _, input := range weekdays {
		t.Run(input, func(t *testing.T) {
			token, err := p.ParseToken(input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", input, err)
			}
			if token.Type != TokenWeekday {
				t.Errorf("ParseToken(%q) type = %v, want TokenWeekday", input, token.Type)
			}
			if token.Date == nil {
				t.Errorf("ParseToken(%q) date is nil", input)
			}
		})
	}
}

func TestParseISODate(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	tests := []struct {
		input string
		year  int
		month time.Month
		day   int
	}{
		{"@2024-03-15", 2024, time.March, 15},
		{"@2025-12-31", 2025, time.December, 31},
		{"@2024-01-01", 2024, time.January, 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != TokenISODate {
				t.Errorf("ParseToken(%q) type = %v, want TokenISODate", tt.input, token.Type)
			}
			if token.Date == nil {
				t.Fatalf("ParseToken(%q) date is nil", tt.input)
			}
			if token.Date.Year() != tt.year || token.Date.Month() != tt.month || token.Date.Day() != tt.day {
				t.Errorf("ParseToken(%q) date = %v, want %d-%02d-%02d",
					tt.input, token.Date, tt.year, tt.month, tt.day)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	tests := []struct {
		input  string
		hour   int
		minute int
	}{
		{"@9am", 9, 0},
		{"@9:30am", 9, 30},
		{"@12pm", 12, 0},
		{"@12:30pm", 12, 30},
		{"@9pm", 21, 0},
		{"@9:45pm", 21, 45},
		{"@12am", 0, 0},
		{"@14:00", 14, 0},
		{"@9:00", 9, 0},
		{"@23:59", 23, 59},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != TokenTime {
				t.Errorf("ParseToken(%q) type = %v, want TokenTime", tt.input, token.Type)
			}
			if token.Time == nil {
				t.Fatalf("ParseToken(%q) time is nil", tt.input)
			}
			if token.Time.Hour != tt.hour || token.Time.Minute != tt.minute {
				t.Errorf("ParseToken(%q) time = %02d:%02d, want %02d:%02d",
					tt.input, token.Time.Hour, token.Time.Minute, tt.hour, tt.minute)
			}
		})
	}
}

func TestParseOffset(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	tests := []struct {
		input    string
		duration time.Duration
	}{
		{"@in:2hours", 2 * time.Hour},
		{"@in:30min", 30 * time.Minute},
		{"@in:1day", 24 * time.Hour},
		{"@in:2days", 48 * time.Hour},
		{"@in:1week", 7 * 24 * time.Hour},
		{"@in:1h", time.Hour},
		{"@in:30m", 30 * time.Minute},
		{"@in:7d", 7 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != TokenOffset {
				t.Errorf("ParseToken(%q) type = %v, want TokenOffset", tt.input, token.Type)
			}
			if token.Offset == nil {
				t.Fatalf("ParseToken(%q) offset is nil", tt.input)
			}
			if token.Offset.Duration != tt.duration {
				t.Errorf("ParseToken(%q) duration = %v, want %v",
					tt.input, token.Offset.Duration, tt.duration)
			}
		})
	}
}

func TestParseDaily(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	tests := []struct {
		input  string
		hour   int
		minute int
	}{
		{"@daily:9am", 9, 0},
		{"@daily:9:30am", 9, 30},
		{"@daily:14:00", 14, 0},
		{"@daily:11pm", 23, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != TokenDaily {
				t.Errorf("ParseToken(%q) type = %v, want TokenDaily", tt.input, token.Type)
			}
			if token.Recurring == nil || token.Recurring.Time == nil {
				t.Fatalf("ParseToken(%q) recurring/time is nil", tt.input)
			}
			if token.Recurring.Time.Hour != tt.hour || token.Recurring.Time.Minute != tt.minute {
				t.Errorf("ParseToken(%q) time = %02d:%02d, want %02d:%02d",
					tt.input, token.Recurring.Time.Hour, token.Recurring.Time.Minute, tt.hour, tt.minute)
			}
		})
	}
}

func TestParseWeekly(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	tests := []struct {
		input string
		days  []time.Weekday
	}{
		{"@weekly:mon", []time.Weekday{time.Monday}},
		{"@weekly:mon,wed", []time.Weekday{time.Monday, time.Wednesday}},
		{"@weekly:mon,wed,fri", []time.Weekday{time.Monday, time.Wednesday, time.Friday}},
		{"@weekly:saturday,sunday", []time.Weekday{time.Saturday, time.Sunday}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != TokenWeekly {
				t.Errorf("ParseToken(%q) type = %v, want TokenWeekly", tt.input, token.Type)
			}
			if token.Recurring == nil {
				t.Fatalf("ParseToken(%q) recurring is nil", tt.input)
			}
			if len(token.Recurring.Days) != len(tt.days) {
				t.Errorf("ParseToken(%q) days count = %d, want %d",
					tt.input, len(token.Recurring.Days), len(tt.days))
			}
		})
	}
}

func TestParseWeeklyWithTime(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	token, err := p.ParseToken("@weekly:mon,wed:9am")
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if token.Type != TokenWeekly {
		t.Errorf("type = %v, want TokenWeekly", token.Type)
	}
	if token.Recurring == nil {
		t.Fatal("recurring is nil")
	}
	if len(token.Recurring.Days) != 2 {
		t.Errorf("days count = %d, want 2", len(token.Recurring.Days))
	}
	if token.Recurring.Time == nil {
		t.Fatal("time is nil")
	}
	if token.Recurring.Time.Hour != 9 {
		t.Errorf("hour = %d, want 9", token.Recurring.Time.Hour)
	}
}

func TestParseMonthly(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	tests := []struct {
		input string
		day   int
	}{
		{"@monthly:1st", 1},
		{"@monthly:2nd", 2},
		{"@monthly:3rd", 3},
		{"@monthly:4th", 4},
		{"@monthly:15", 15},
		{"@monthly:31", 31},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != TokenMonthly {
				t.Errorf("ParseToken(%q) type = %v, want TokenMonthly", tt.input, token.Type)
			}
			if token.Recurring == nil {
				t.Fatalf("ParseToken(%q) recurring is nil", tt.input)
			}
			if token.Recurring.DayOfMonth != tt.day {
				t.Errorf("ParseToken(%q) day = %d, want %d",
					tt.input, token.Recurring.DayOfMonth, tt.day)
			}
		})
	}
}

func TestParseYearly(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	tests := []struct {
		input string
		month time.Month
		day   int
	}{
		{"@yearly:mar-15", time.March, 15},
		{"@yearly:jan-1", time.January, 1},
		{"@yearly:dec-25", time.December, 25},
		{"@yearly:february-14", time.February, 14},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != TokenYearly {
				t.Errorf("ParseToken(%q) type = %v, want TokenYearly", tt.input, token.Type)
			}
			if token.Recurring == nil {
				t.Fatalf("ParseToken(%q) recurring is nil", tt.input)
			}
			if token.Recurring.Month != tt.month || token.Recurring.Day != tt.day {
				t.Errorf("ParseToken(%q) date = %v-%d, want %v-%d",
					tt.input, token.Recurring.Month, token.Recurring.Day, tt.month, tt.day)
			}
		})
	}
}

func TestParseTextSkipsCodeBlocks(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	content := `
# Schedule test

This has a schedule @daily:9am that should be found.

` + "```" + `go
// This @weekly:mon should be skipped
func example() {}
` + "```" + `

But this @monthly:15 should be found.
`

	tokens, err := p.ParseText(content)
	if err != nil {
		t.Fatalf("ParseText error: %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("got %d tokens, want 2", len(tokens))
	}

	// Verify we found the right tokens
	foundDaily := false
	foundMonthly := false
	for _, token := range tokens {
		if token.Type == TokenDaily {
			foundDaily = true
		}
		if token.Type == TokenMonthly {
			foundMonthly = true
		}
	}

	if !foundDaily {
		t.Error("did not find daily token")
	}
	if !foundMonthly {
		t.Error("did not find monthly token")
	}
}

func TestParseTextSkipsInlineCode(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	content := "Use `@daily:9am` for daily schedules. See @weekly:mon for weekly."

	tokens, err := p.ParseText(content)
	if err != nil {
		t.Fatalf("ParseText error: %v", err)
	}

	if len(tokens) != 1 {
		t.Errorf("got %d tokens, want 1", len(tokens))
	}

	if len(tokens) > 0 && tokens[0].Type != TokenWeekly {
		t.Errorf("got type %v, want TokenWeekly", tokens[0].Type)
	}
}

func TestParseEscapedToken(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	content := `Use \@daily:9am for escaped tokens. But @weekly:mon should work.`

	tokens, err := p.ParseText(content)
	if err != nil {
		t.Fatalf("ParseText error: %v", err)
	}

	if len(tokens) != 1 {
		t.Errorf("got %d tokens, want 1 (escaped should be skipped)", len(tokens))
	}
}

func TestParseWarningsOnInvalidTokens(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	content := `Valid @daily:9am and invalid @badtoken here.`

	tokens, err := p.ParseText(content)
	if err != nil {
		t.Fatalf("ParseText error: %v", err)
	}

	if len(tokens) != 1 {
		t.Errorf("got %d tokens, want 1", len(tokens))
	}

	if len(p.Warnings) != 1 {
		t.Errorf("got %d warnings, want 1", len(p.Warnings))
	}
}

func TestNextOccurrenceDaily(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	token, _ := p.ParseToken("@daily:9am")

	// Set "now" to 8am on Jan 15, 2024
	now := time.Date(2024, time.January, 15, 8, 0, 0, 0, loc)
	next := token.NextOccurrence(now, loc)

	// Should be 9am same day
	want := time.Date(2024, time.January, 15, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}

	// Set "now" to 10am (after 9am)
	now = time.Date(2024, time.January, 15, 10, 0, 0, 0, loc)
	next = token.NextOccurrence(now, loc)

	// Should be 9am next day
	want = time.Date(2024, time.January, 16, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextOccurrenceWeekly(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	token, _ := p.ParseToken("@weekly:mon,wed:9am")

	// Monday Jan 15, 2024 at 8am
	now := time.Date(2024, time.January, 15, 8, 0, 0, 0, loc)
	next := token.NextOccurrence(now, loc)

	// Should be 9am same Monday
	want := time.Date(2024, time.January, 15, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}

	// Monday Jan 15, 2024 at 10am (after 9am)
	now = time.Date(2024, time.January, 15, 10, 0, 0, 0, loc)
	next = token.NextOccurrence(now, loc)

	// Should be Wednesday Jan 17 at 9am
	want = time.Date(2024, time.January, 17, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextOccurrenceMonthly(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	token, _ := p.ParseToken("@monthly:15:9am")

	// Jan 10 at 8am
	now := time.Date(2024, time.January, 10, 8, 0, 0, 0, loc)
	next := token.NextOccurrence(now, loc)

	// Should be Jan 15 at 9am
	want := time.Date(2024, time.January, 15, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}

	// Jan 15 at 10am (after scheduled time)
	now = time.Date(2024, time.January, 15, 10, 0, 0, 0, loc)
	next = token.NextOccurrence(now, loc)

	// Should be Feb 15 at 9am
	want = time.Date(2024, time.February, 15, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextOccurrenceYearly(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	token, _ := p.ParseToken("@yearly:mar-15:9am")

	// Jan 10, 2024
	now := time.Date(2024, time.January, 10, 8, 0, 0, 0, loc)
	next := token.NextOccurrence(now, loc)

	// Should be Mar 15, 2024 at 9am
	want := time.Date(2024, time.March, 15, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}

	// Apr 1, 2024 (after Mar 15)
	now = time.Date(2024, time.April, 1, 8, 0, 0, 0, loc)
	next = token.NextOccurrence(now, loc)

	// Should be Mar 15, 2025 at 9am
	want = time.Date(2025, time.March, 15, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextOccurrenceOffset(t *testing.T) {
	loc := time.UTC
	p := NewParser(loc)

	token, _ := p.ParseToken("@in:2hours")

	now := time.Date(2024, time.January, 15, 10, 30, 0, 0, loc)
	next := token.NextOccurrence(now, loc)

	want := time.Date(2024, time.January, 15, 12, 30, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}
