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
