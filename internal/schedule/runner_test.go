package schedule

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestParseNotifyImperative(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	tests := []struct {
		line    string
		wantMsg string
		hasToken bool
	}{
		{"Notify @daily:9am Check your email", "Check your email", true},
		{"Notify @weekly:mon,fri Meeting reminder", "Meeting reminder", true},
		{"Notify Simple message without schedule", "Simple message without schedule", false},
		{"Notify @in:2hours Time to stretch", "Time to stretch", true},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			imp := r.parseNotifyLine(tt.line, 1)
			if imp == nil {
				t.Fatal("parseNotifyLine returned nil")
			}
			if imp.Type != ImperativeNotify {
				t.Errorf("type = %v, want ImperativeNotify", imp.Type)
			}
			if imp.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", imp.Message, tt.wantMsg)
			}
			if tt.hasToken && imp.Token == nil {
				t.Error("expected token but got nil")
			}
			if !tt.hasToken && imp.Token != nil {
				t.Error("expected no token but got one")
			}
		})
	}
}

func TestParseRunActionImperative(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	tests := []struct {
		line       string
		wantAction string
		wantArgs   []string
		hasToken   bool
	}{
		{"Run action:backup @daily:2am", "backup", nil, true},
		{"Run action:sync @weekly:mon,fri:9am --force", "sync", []string{"--force"}, true},
		{"Run action:cleanup", "cleanup", nil, false},
		{"Run action:deploy @monthly:1st:3am --env prod", "deploy", []string{"--env", "prod"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			imp := r.parseRunActionLine(tt.line, 1)
			if imp == nil {
				t.Fatal("parseRunActionLine returned nil")
			}
			if imp.Type != ImperativeRunAction {
				t.Errorf("type = %v, want ImperativeRunAction", imp.Type)
			}
			if imp.ActionName != tt.wantAction {
				t.Errorf("action = %q, want %q", imp.ActionName, tt.wantAction)
			}
			if tt.hasToken && imp.Token == nil {
				t.Error("expected token but got nil")
			}
			if !tt.hasToken && imp.Token != nil {
				t.Error("expected no token but got one")
			}
			if len(imp.Args) != len(tt.wantArgs) {
				t.Errorf("args count = %d, want %d", len(imp.Args), len(tt.wantArgs))
			}
		})
	}
}

func TestParsePage(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	content := `
# My Page

Some regular content here.

Notify @daily:9am Check your email

More content.

` + "```" + `go
// This should be skipped
Notify @weekly:mon Should not be parsed
` + "```" + `

Run action:backup @weekly:sun:2am

Regular line
`

	imperatives, err := r.ParsePage("test-page", content)
	if err != nil {
		t.Fatalf("ParsePage error: %v", err)
	}

	if len(imperatives) != 2 {
		t.Errorf("got %d imperatives, want 2", len(imperatives))
	}

	// Check jobs were registered
	jobs := r.GetJobsForPage("test-page")
	if len(jobs) != 2 {
		t.Errorf("got %d jobs, want 2", len(jobs))
	}
}

func TestIsImperativeLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"Notify @daily:9am Test", true},
		{"Run action:backup", true},
		{"  Notify with leading spaces", true},
		{"\tRun action:test", true},
		{"Regular line", false},
		{"notify lowercase", false},
		{"run action lowercase", false},
		{"Something Notify", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := IsImperativeLine(tt.line)
			if got != tt.want {
				t.Errorf("IsImperativeLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestExtractImperatives(t *testing.T) {
	content := `
Notify @daily:9am First
Regular line
Run action:test @weekly:mon
Another line
Notify Simple message
`
	imperatives := ExtractImperatives(content, time.UTC)

	if len(imperatives) != 3 {
		t.Errorf("got %d imperatives, want 3", len(imperatives))
	}

	// Verify types
	if imperatives[0].Type != ImperativeNotify {
		t.Error("first should be Notify")
	}
	if imperatives[1].Type != ImperativeRunAction {
		t.Error("second should be RunAction")
	}
	if imperatives[2].Type != ImperativeNotify {
		t.Error("third should be Notify")
	}
}

func TestRunnerHandlers(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	var mu sync.Mutex
	notifications := make([]string, 0)
	actions := make([]string, 0)

	r.SetNotificationHandler(func(pageID, message string) error {
		mu.Lock()
		notifications = append(notifications, message)
		mu.Unlock()
		return nil
	})

	r.SetActionHandler(func(pageID, actionName string, args []string, message string) error {
		mu.Lock()
		actions = append(actions, actionName)
		mu.Unlock()
		return nil
	})

	// Set a fixed time
	fixedTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	r.SetTimeFunc(func() time.Time { return fixedTime })

	content := `
Notify @daily:9am Morning reminder
Run action:backup @daily:9am
`

	_, err := r.ParsePage("test-page", content)
	if err != nil {
		t.Fatalf("ParsePage error: %v", err)
	}

	// Trigger manually
	r.Tick()

	mu.Lock()
	defer mu.Unlock()

	if len(notifications) != 1 {
		t.Errorf("got %d notifications, want 1", len(notifications))
	}
	if len(actions) != 1 {
		t.Errorf("got %d actions, want 1", len(actions))
	}
	if len(notifications) > 0 && notifications[0] != "Morning reminder" {
		t.Errorf("notification = %q, want %q", notifications[0], "Morning reminder")
	}
	if len(actions) > 0 && actions[0] != "backup" {
		t.Errorf("action = %q, want %q", actions[0], "backup")
	}
}

func TestRemovePage(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	content := `Notify @daily:9am Test`
	r.ParsePage("page1", content)
	r.ParsePage("page2", content)

	if len(r.GetAllJobs()) != 2 {
		t.Errorf("got %d jobs, want 2", len(r.GetAllJobs()))
	}

	r.RemovePage("page1")

	jobs := r.GetAllJobs()
	if len(jobs) != 1 {
		t.Errorf("got %d jobs after removal, want 1", len(jobs))
	}
	if jobs[0].PageID != "page2" {
		t.Errorf("remaining job pageID = %q, want %q", jobs[0].PageID, "page2")
	}
}

func TestTokenPattern(t *testing.T) {
	pattern := TokenPattern()

	tests := []struct {
		input string
		want  bool
	}{
		{"@today", true},
		{"@tomorrow", true},
		{"@monday", true},
		{"@2024-03-15", true},
		{"@9am", true},
		{"@9:30pm", true},
		{"@in:2hours", true},
		{"@daily:9am", true},
		{"@weekly:mon,wed", true},
		{"@monthly:1st", true},
		{"@yearly:mar-15", true},
		{"@invalid", false},
		{"regular text", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pattern.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("pattern.MatchString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCronScheduler(t *testing.T) {
	cron := NewCron(time.UTC)

	var mu sync.Mutex
	executed := make([]string, 0)

	// Set fixed time
	fixedTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	cron.SetTimeFunc(func() time.Time { return fixedTime })

	job := &Job{
		ID:     "test-job",
		PageID: "test-page",
		Line:   "Test job",
		NextRun: fixedTime,
		Handler: func(j *Job) error {
			mu.Lock()
			executed = append(executed, j.ID)
			mu.Unlock()
			return nil
		},
	}

	cron.AddJob(job)

	if cron.JobCount() != 1 {
		t.Errorf("job count = %d, want 1", cron.JobCount())
	}

	// Manually tick
	cron.Tick()

	mu.Lock()
	defer mu.Unlock()

	if len(executed) != 1 {
		t.Errorf("executed count = %d, want 1", len(executed))
	}
	if len(executed) > 0 && executed[0] != "test-job" {
		t.Errorf("executed[0] = %q, want %q", executed[0], "test-job")
	}
}

func TestCronStartStop(t *testing.T) {
	cron := NewCron(time.UTC)

	if cron.IsRunning() {
		t.Error("cron should not be running initially")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cron.Start(ctx)

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	if !cron.IsRunning() {
		t.Error("cron should be running after Start")
	}

	cancel()
	cron.Stop()

	// Give it a moment to stop
	time.Sleep(10 * time.Millisecond)

	if cron.IsRunning() {
		t.Error("cron should not be running after Stop")
	}
}

func TestCronDisableEnableJob(t *testing.T) {
	cron := NewCron(time.UTC)

	job := &Job{
		ID:      "test-job",
		PageID:  "test-page",
		Enabled: true,
		NextRun: time.Now(),
	}

	cron.AddJob(job)

	// Disable
	ok := cron.DisableJob("test-job")
	if !ok {
		t.Error("DisableJob failed")
	}

	j := cron.GetJob("test-job")
	if j.Enabled {
		t.Error("job should be disabled")
	}

	// Enable
	ok = cron.EnableJob("test-job")
	if !ok {
		t.Error("EnableJob failed")
	}

	j = cron.GetJob("test-job")
	if !j.Enabled {
		t.Error("job should be enabled")
	}
}

func TestRecurringJobReschedule(t *testing.T) {
	cron := NewCron(time.UTC)
	p := NewParser(time.UTC)

	fixedTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	cron.SetTimeFunc(func() time.Time { return fixedTime })

	token, _ := p.ParseToken("@daily:9am")

	job := &Job{
		ID:      "recurring-job",
		PageID:  "test-page",
		Token:   token,
		NextRun: fixedTime,
		Handler: func(j *Job) error { return nil },
	}

	cron.AddJob(job)

	// Execute
	cron.Tick()

	// Check next run was rescheduled
	j := cron.GetJob("recurring-job")
	if j == nil {
		t.Fatal("job not found")
	}

	// Next run should be tomorrow at 9am
	expectedNext := time.Date(2024, time.January, 16, 9, 0, 0, 0, time.UTC)
	if !j.NextRun.Equal(expectedNext) {
		t.Errorf("next run = %v, want %v", j.NextRun, expectedNext)
	}
}

func TestOneTimeJobDisabled(t *testing.T) {
	cron := NewCron(time.UTC)
	p := NewParser(time.UTC)

	fixedTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	cron.SetTimeFunc(func() time.Time { return fixedTime })

	token, _ := p.ParseToken("@in:1hour")

	job := &Job{
		ID:      "one-time-job",
		PageID:  "test-page",
		Token:   token,
		NextRun: fixedTime,
		Handler: func(j *Job) error { return nil },
	}

	cron.AddJob(job)

	// Advance time to when the job is scheduled (now + 1 hour)
	scheduledTime := fixedTime.Add(time.Hour)
	cron.SetTimeFunc(func() time.Time { return scheduledTime })

	// Execute
	cron.Tick()

	// Job should be disabled (one-time)
	j := cron.GetJob("one-time-job")
	if j == nil {
		t.Fatal("job not found")
	}
	if j.Enabled {
		t.Error("one-time job should be disabled after execution")
	}
}

func TestFilterTokenParsing(t *testing.T) {
	p := NewParser(time.UTC)

	tests := []struct {
		input    string
		wantType TokenType
		isFilter bool
	}{
		{"@weekdays", TokenWeekdays, true},
		{"@weekends", TokenWeekends, true},
		{"@WEEKDAYS", TokenWeekdays, true},
		{"@Weekends", TokenWeekends, true},
		{"@daily:9am", TokenDaily, false},
		{"@weekly:mon", TokenWeekly, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			token, err := p.ParseToken(tt.input)
			if err != nil {
				t.Fatalf("ParseToken(%q) error: %v", tt.input, err)
			}
			if token.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", token.Type, tt.wantType)
			}
			if token.IsFilterToken() != tt.isFilter {
				t.Errorf("IsFilterToken() = %v, want %v", token.IsFilterToken(), tt.isFilter)
			}
		})
	}
}

func TestPassesFilter(t *testing.T) {
	p := NewParser(time.UTC)

	// 2024-01-15 is a Monday
	monday := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	// 2024-01-20 is a Saturday
	saturday := time.Date(2024, time.January, 20, 9, 0, 0, 0, time.UTC)
	// 2024-01-21 is a Sunday
	sunday := time.Date(2024, time.January, 21, 9, 0, 0, 0, time.UTC)

	weekdaysToken, _ := p.ParseToken("@weekdays")
	weekendsToken, _ := p.ParseToken("@weekends")

	tests := []struct {
		name   string
		token  *Token
		time   time.Time
		passes bool
	}{
		{"weekdays passes Monday", weekdaysToken, monday, true},
		{"weekdays fails Saturday", weekdaysToken, saturday, false},
		{"weekdays fails Sunday", weekdaysToken, sunday, false},
		{"weekends passes Saturday", weekendsToken, saturday, true},
		{"weekends passes Sunday", weekendsToken, sunday, true},
		{"weekends fails Monday", weekendsToken, monday, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.PassesFilter(tt.time)
			if got != tt.passes {
				t.Errorf("PassesFilter(%v) = %v, want %v", tt.time.Weekday(), got, tt.passes)
			}
		})
	}
}

func TestNextOccurrenceWithFilters(t *testing.T) {
	p := NewParser(time.UTC)

	// 2024-01-15 is a Monday at 8:59am - before the schedule time
	mondayBefore := time.Date(2024, time.January, 15, 8, 59, 0, 0, time.UTC)

	dailyToken, _ := p.ParseToken("@daily:9am")
	weekdaysFilter, _ := p.ParseToken("@weekdays")
	weekendsFilter, _ := p.ParseToken("@weekends")

	// Test @daily:9am @weekdays from Monday before 9am
	// Should return Monday Jan 15 at 9am (same day since it passes weekday filter)
	next := NextOccurrenceWithFilters(dailyToken, []*Token{weekdaysFilter}, mondayBefore, time.UTC)
	expectedWeekday := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expectedWeekday) {
		t.Errorf("NextOccurrenceWithFilters(@daily:9am @weekdays from Monday before 9am) = %v, want %v", next, expectedWeekday)
	}

	// Test @daily:9am @weekends from Monday before 9am
	// Should skip weekdays and return Saturday Jan 20 at 9am
	nextWeekend := NextOccurrenceWithFilters(dailyToken, []*Token{weekendsFilter}, mondayBefore, time.UTC)
	expectedWeekend := time.Date(2024, time.January, 20, 9, 0, 0, 0, time.UTC)
	if !nextWeekend.Equal(expectedWeekend) {
		t.Errorf("NextOccurrenceWithFilters(@daily:9am @weekends from Monday) = %v, want %v", nextWeekend, expectedWeekend)
	}

	// Test with no filters - should behave same as NextOccurrence
	nextNoFilter := NextOccurrenceWithFilters(dailyToken, nil, mondayBefore, time.UTC)
	expectedNoFilter := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	if !nextNoFilter.Equal(expectedNoFilter) {
		t.Errorf("NextOccurrenceWithFilters(@daily:9am no filters from Monday before 9am) = %v, want %v", nextNoFilter, expectedNoFilter)
	}
}

func TestParseNotifyWithFilterTokens(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	tests := []struct {
		line           string
		wantFilters    int
		wantMsg        string
		hasSchedule    bool
	}{
		{"Notify @daily:9am @weekdays Standup", 1, "Standup", true},
		{"Notify @daily:9am @weekdays @weekends Test", 2, "Test", true},
		{"Notify @weekdays Only filter", 1, "Only filter", false}, // @weekdays is parsed as filter, no schedule
		{"Notify @daily:9am No filters", 0, "No filters", true},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			imp := r.parseNotifyLine(tt.line, 1)
			if imp == nil {
				t.Fatal("parseNotifyLine returned nil")
			}
			if len(imp.FilterTokens) != tt.wantFilters {
				t.Errorf("FilterTokens count = %d, want %d", len(imp.FilterTokens), tt.wantFilters)
			}
			if imp.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", imp.Message, tt.wantMsg)
			}
			if tt.hasSchedule && imp.Token == nil {
				t.Error("expected schedule token but got nil")
			}
			if !tt.hasSchedule && imp.Token != nil {
				t.Errorf("expected no schedule token but got %v", imp.Token.Type)
			}
		})
	}
}

func TestParseBlockquoteMessage(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	tests := []struct {
		name         string
		lines        []string
		startIdx     int
		wantMessage  string
		wantConsumed int
	}{
		{
			name:         "single line blockquote",
			lines:        []string{"> Hello world"},
			startIdx:     0,
			wantMessage:  "Hello world",
			wantConsumed: 1,
		},
		{
			name:         "multi-line blockquote",
			lines:        []string{"> Line 1", "> Line 2", "> Line 3"},
			startIdx:     0,
			wantMessage:  "Line 1\nLine 2\nLine 3",
			wantConsumed: 3,
		},
		{
			name:         "blockquote with empty line after",
			lines:        []string{"> Hello", "", "Regular line"},
			startIdx:     0,
			wantMessage:  "Hello",
			wantConsumed: 1,
		},
		{
			name:         "blockquote with leading empty lines",
			lines:        []string{"", "", "> Hello"},
			startIdx:     0,
			wantMessage:  "Hello",
			wantConsumed: 3,
		},
		{
			name:         "no blockquote",
			lines:        []string{"Regular line", "Another line"},
			startIdx:     0,
			wantMessage:  "",
			wantConsumed: 0,
		},
		{
			name:         "blockquote terminated by non-blockquote",
			lines:        []string{"> Part 1", "> Part 2", "Not a blockquote"},
			startIdx:     0,
			wantMessage:  "Part 1\nPart 2",
			wantConsumed: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, consumed := r.parseBlockquoteMessage(tt.lines, tt.startIdx)
			if msg != tt.wantMessage {
				t.Errorf("message = %q, want %q", msg, tt.wantMessage)
			}
			if consumed != tt.wantConsumed {
				t.Errorf("consumed = %d, want %d", consumed, tt.wantConsumed)
			}
		})
	}
}

func TestParsePageWithBlockquotes(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	content := `
Notify @daily:9am
> This is a blockquote message
> spanning multiple lines

Run action:test @weekly:mon
> Action message here

Regular content
`

	imperatives, err := r.ParsePage("test-page", content)
	if err != nil {
		t.Fatalf("ParsePage error: %v", err)
	}

	if len(imperatives) != 2 {
		t.Errorf("got %d imperatives, want 2", len(imperatives))
		return
	}

	// Check first imperative (Notify) has blockquote message
	if imperatives[0].Type != ImperativeNotify {
		t.Error("first imperative should be Notify")
	}
	wantMsg1 := "This is a blockquote message\nspanning multiple lines"
	if imperatives[0].Message != wantMsg1 {
		t.Errorf("first message = %q, want %q", imperatives[0].Message, wantMsg1)
	}

	// Check second imperative (Run) has blockquote message
	if imperatives[1].Type != ImperativeRunAction {
		t.Error("second imperative should be RunAction")
	}
	wantMsg2 := "Action message here"
	if imperatives[1].Message != wantMsg2 {
		t.Errorf("second message = %q, want %q", imperatives[1].Message, wantMsg2)
	}
}

func TestBlockquoteNotReParsed(t *testing.T) {
	r := NewRunner(RunnerConfig{Location: time.UTC})

	// This test ensures blockquote lines are consumed and not re-parsed
	// as separate imperatives if they happen to look like imperatives
	content := `
Notify @daily:9am First notification
> Notify @weekly:mon This should NOT be parsed as separate imperative

Notify @daily:10am Second notification
`

	imperatives, err := r.ParsePage("test-page", content)
	if err != nil {
		t.Fatalf("ParsePage error: %v", err)
	}

	// Should only have 2 imperatives, not 3
	if len(imperatives) != 2 {
		t.Errorf("got %d imperatives, want 2 (blockquote content should not be re-parsed)", len(imperatives))
	}

	// First imperative should have the blockquote as its message
	if len(imperatives) > 0 {
		wantMsg := "Notify @weekly:mon This should NOT be parsed as separate imperative"
		if imperatives[0].Message != wantMsg {
			t.Errorf("first message = %q, want %q", imperatives[0].Message, wantMsg)
		}
	}
}
