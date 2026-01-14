package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ImperativeType indicates the type of imperative command.
type ImperativeType int

const (
	ImperativeNotify ImperativeType = iota
	ImperativeRunAction
)

// Imperative represents a parsed imperative line.
type Imperative struct {
	Type         ImperativeType
	Token        *Token   // The primary schedule token (e.g., @daily:9am)
	FilterTokens []*Token // Filter tokens (e.g., @weekdays, @weekends)
	Message      string   // For Notify: inline message, or blockquote content
	ActionName   string   // For RunAction: the action name
	Args         []string // For RunAction: optional arguments
	Line         int      // Source line number
	Raw          string   // Original line text
}

// NotificationHandler is called when a notification should be sent.
type NotificationHandler func(pageID, message string) error

// ActionHandler is called when an action should be executed.
// The message parameter contains any blockquote content associated with the imperative.
type ActionHandler func(pageID, actionName string, args []string, message string) error

// Runner manages scheduled job execution.
type Runner struct {
	mu          sync.RWMutex
	cron        *Cron
	parser      *Parser
	location    *time.Location
	imperatives map[string][]*Imperative // pageID -> imperatives
	stateDir    string                   // Directory for persistence

	// Handlers for execution
	onNotify NotificationHandler
	onAction ActionHandler
}

// RunnerConfig configures a new Runner.
type RunnerConfig struct {
	Location *time.Location
	StateDir string // Directory for state persistence (optional)
}

// NewRunner creates a new schedule runner.
func NewRunner(cfg RunnerConfig) *Runner {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}

	return &Runner{
		cron:        NewCron(cfg.Location),
		parser:      NewParser(cfg.Location),
		location:    cfg.Location,
		imperatives: make(map[string][]*Imperative),
		stateDir:    cfg.StateDir,
	}
}

// SetNotificationHandler sets the callback for notifications.
func (r *Runner) SetNotificationHandler(h NotificationHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onNotify = h
}

// SetActionHandler sets the callback for action execution.
func (r *Runner) SetActionHandler(h ActionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onAction = h
}

// ParsePage extracts and registers imperative schedules from page content.
func (r *Runner) ParsePage(pageID, content string) ([]*Imperative, error) {
	imperatives, err := r.parseImperatives(content)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.imperatives[pageID] = imperatives
	r.mu.Unlock()

	// Register jobs with cron
	r.cron.RemoveJobsByPage(pageID)

	for i, imp := range imperatives {
		if imp.Token == nil {
			continue // No schedule token
		}

		jobID := fmt.Sprintf("%s:%d", pageID, i)
		job := &Job{
			ID:      jobID,
			PageID:  pageID,
			Line:    imp.Raw,
			Token:   imp.Token,
			Handler: r.createJobHandler(pageID, imp),
		}
		r.cron.AddJob(job)
	}

	return imperatives, nil
}

// parseImperatives extracts imperative lines from content.
func (r *Runner) parseImperatives(content string) ([]*Imperative, error) {
	var imperatives []*Imperative

	lines := strings.Split(content, "\n")
	inCodeBlock := false

	for lineNum, line := range lines {
		// Skip code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Check for Notify imperative
		if strings.HasPrefix(trimmed, "Notify ") {
			imp := r.parseNotifyLine(trimmed, lineNum+1)
			if imp != nil {
				imperatives = append(imperatives, imp)
			}
		}

		// Check for Run action imperative
		if strings.HasPrefix(trimmed, "Run action:") {
			imp := r.parseRunActionLine(trimmed, lineNum+1)
			if imp != nil {
				imperatives = append(imperatives, imp)
			}
		}
	}

	return imperatives, nil
}

// parseNotifyLine parses a "Notify @schedule message" line.
func (r *Runner) parseNotifyLine(line string, lineNum int) *Imperative {
	// Format: Notify @schedule Message here
	// Example: Notify @daily:9am Check your email

	rest := strings.TrimPrefix(line, "Notify ")
	rest = strings.TrimSpace(rest)

	if len(rest) == 0 {
		return nil
	}

	// Find the schedule token
	var token *Token
	var message string

	// Look for @ token at the start
	if strings.HasPrefix(rest, "@") {
		// Extract token (up to first space)
		parts := strings.SplitN(rest, " ", 2)
		tokenStr := parts[0]

		var err error
		token, err = r.parser.ParseToken(tokenStr)
		if err != nil {
			// Add warning but continue
			r.parser.Warnings = append(r.parser.Warnings, ParseWarning{
				Message: err.Error(),
				Line:    lineNum,
				Token:   tokenStr,
			})
		}

		if len(parts) > 1 {
			message = strings.TrimSpace(parts[1])

			// Check for additional schedule tokens in the message
			for _, word := range strings.Fields(message) {
				if strings.HasPrefix(word, "@") {
					// Check if it looks like a schedule token
					if _, parseErr := r.parser.ParseToken(word); parseErr == nil {
						r.parser.Warnings = append(r.parser.Warnings, ParseWarning{
							Message: "multiple schedule tokens in imperative; only first will be used",
							Line:    lineNum,
							Token:   word,
						})
					}
				}
			}
		}
	} else {
		// No schedule token, just a message
		message = rest
	}

	return &Imperative{
		Type:    ImperativeNotify,
		Token:   token,
		Message: message,
		Line:    lineNum,
		Raw:     line,
	}
}

// parseRunActionLine parses a "Run action:name @schedule" line.
func (r *Runner) parseRunActionLine(line string, lineNum int) *Imperative {
	// Format: Run action:name @schedule [args...]
	// Example: Run action:backup @daily:2am
	// Example: Run action:sync @weekly:mon,fri:9am --force

	rest := strings.TrimPrefix(line, "Run action:")
	rest = strings.TrimSpace(rest)

	if len(rest) == 0 {
		return nil
	}

	// Extract action name (first word)
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return nil
	}

	actionName := parts[0]
	var token *Token
	var args []string
	tokenCount := 0

	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "@") {
			tokenCount++
			if token == nil {
				// This is the first schedule token - use it
				var err error
				token, err = r.parser.ParseToken(part)
				if err != nil {
					r.parser.Warnings = append(r.parser.Warnings, ParseWarning{
						Message: err.Error(),
						Line:    lineNum,
						Token:   part,
					})
				}
			} else {
				// Multiple schedule tokens - warn and treat as argument
				r.parser.Warnings = append(r.parser.Warnings, ParseWarning{
					Message: "multiple schedule tokens in imperative; only first will be used",
					Line:    lineNum,
					Token:   part,
				})
				args = append(args, part)
			}
		} else {
			// This is an argument
			args = append(args, part)
		}
	}
	_ = tokenCount // Used for warning above

	return &Imperative{
		Type:       ImperativeRunAction,
		Token:      token,
		ActionName: actionName,
		Args:       args,
		Line:       lineNum,
		Raw:        line,
	}
}

// createJobHandler creates a handler function for a job.
func (r *Runner) createJobHandler(pageID string, imp *Imperative) JobHandler {
	return func(job *Job) error {
		r.mu.RLock()
		notifyHandler := r.onNotify
		actionHandler := r.onAction
		r.mu.RUnlock()

		switch imp.Type {
		case ImperativeNotify:
			if notifyHandler != nil {
				return notifyHandler(pageID, imp.Message)
			}
		case ImperativeRunAction:
			if actionHandler != nil {
				return actionHandler(pageID, imp.ActionName, imp.Args, imp.Message)
			}
		}
		return nil
	}
}

// Start begins the schedule runner.
func (r *Runner) Start(ctx context.Context) error {
	// Load persisted state if available
	if r.stateDir != "" {
		if err := r.loadState(); err != nil {
			// Non-fatal: log and continue
			// In production, use proper logging
		}
	}

	r.cron.Start(ctx)
	return nil
}

// Stop halts the schedule runner.
func (r *Runner) Stop() error {
	r.cron.Stop()

	// Persist state if configured
	if r.stateDir != "" {
		if err := r.saveState(); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
	}

	return nil
}

// RemovePage removes all schedules for a page.
func (r *Runner) RemovePage(pageID string) {
	r.mu.Lock()
	delete(r.imperatives, pageID)
	r.mu.Unlock()

	r.cron.RemoveJobsByPage(pageID)
}

// GetImperatives returns all imperatives for a page.
func (r *Runner) GetImperatives(pageID string) []*Imperative {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.imperatives[pageID]
}

// GetAllJobs returns all scheduled jobs.
func (r *Runner) GetAllJobs() []*Job {
	return r.cron.GetJobs()
}

// GetJobsForPage returns all jobs for a specific page.
func (r *Runner) GetJobsForPage(pageID string) []*Job {
	return r.cron.GetJobsByPage(pageID)
}

// AddJob adds a pre-configured job to the scheduler.
// This is useful when jobs are created externally (e.g., from parsed Page imperatives).
func (r *Runner) AddJob(job *Job) {
	r.cron.AddJob(job)
}

// GetWarnings returns any parsing warnings.
func (r *Runner) GetWarnings() []ParseWarning {
	return r.parser.Warnings
}

// SetTimeFunc sets a custom time function (for testing).
func (r *Runner) SetTimeFunc(fn func() time.Time) {
	r.cron.SetTimeFunc(fn)
}

// Tick manually triggers a schedule check (for testing).
func (r *Runner) Tick() {
	r.cron.Tick()
}

// State persistence

// persistedState represents the saved state for restarts.
type persistedState struct {
	Jobs []persistedJob `json:"jobs"`
}

type persistedJob struct {
	ID       string     `json:"id"`
	PageID   string     `json:"page_id"`
	Line     string     `json:"line"`
	NextRun  time.Time  `json:"next_run"`
	LastRun  *time.Time `json:"last_run,omitempty"`
	Enabled  bool       `json:"enabled"`
	TokenRaw string     `json:"token_raw"`
}

// saveState persists the current state to disk.
func (r *Runner) saveState() error {
	if r.stateDir == "" {
		return nil
	}

	jobs := r.cron.GetJobs()
	state := persistedState{
		Jobs: make([]persistedJob, len(jobs)),
	}

	for i, job := range jobs {
		tokenRaw := ""
		if job.Token != nil {
			tokenRaw = job.Token.Raw
		}
		state.Jobs[i] = persistedJob{
			ID:       job.ID,
			PageID:   job.PageID,
			Line:     job.Line,
			NextRun:  job.NextRun,
			LastRun:  job.LastRun,
			Enabled:  job.Enabled,
			TokenRaw: tokenRaw,
		}
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	stateFile := filepath.Join(r.stateDir, "schedule_state.json")
	if err := os.MkdirAll(r.stateDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0600)
}

// loadState restores state from disk.
func (r *Runner) loadState() error {
	if r.stateDir == "" {
		return nil
	}

	stateFile := filepath.Join(r.stateDir, "schedule_state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state file, that's OK
		}
		return err
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	// Restore jobs
	for _, pj := range state.Jobs {
		var token *Token
		if pj.TokenRaw != "" {
			token, _ = r.parser.ParseToken(pj.TokenRaw)
		}

		job := &Job{
			ID:      pj.ID,
			PageID:  pj.PageID,
			Line:    pj.Line,
			Token:   token,
			NextRun: pj.NextRun,
			LastRun: pj.LastRun,
			Enabled: pj.Enabled,
		}

		// Restore handler by re-parsing the imperative line
		if pj.Line != "" {
			imp := r.parseImperativeFromLine(pj.Line)
			if imp != nil {
				job.Handler = r.createJobHandler(pj.PageID, imp)
			}
		}

		// Recalculate next run if it's in the past
		if job.Enabled && job.Token != nil {
			now := time.Now().In(r.location)
			if job.NextRun.Before(now) {
				job.NextRun = job.Token.NextOccurrence(now, r.location)
			}
		}

		r.cron.AddJob(job)
	}

	return nil
}

// parseImperativeFromLine parses a single imperative line for handler restoration.
func (r *Runner) parseImperativeFromLine(line string) *Imperative {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "Notify ") {
		return r.parseNotifyLine(line, 0)
	}
	if strings.HasPrefix(line, "Run action:") {
		return r.parseRunActionLine(line, 0)
	}
	return nil
}

// ExtractScheduleTokens finds all schedule tokens in text (helper function).
func ExtractScheduleTokens(text string, location *time.Location) ([]*Token, []ParseWarning) {
	p := NewParser(location)
	tokens, _ := p.ParseText(text)
	return tokens, p.Warnings
}

// ParseImperativeLine parses a single imperative line (helper function).
func ParseImperativeLine(line string, location *time.Location) *Imperative {
	r := NewRunner(RunnerConfig{Location: location})
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "Notify ") {
		return r.parseNotifyLine(line, 1)
	}
	if strings.HasPrefix(line, "Run action:") {
		return r.parseRunActionLine(line, 1)
	}
	return nil
}

// IsImperativeLine checks if a line is an imperative command.
func IsImperativeLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "Notify ") || strings.HasPrefix(trimmed, "Run action:")
}

// ExtractImperatives returns all imperative lines from content.
func ExtractImperatives(content string, location *time.Location) []*Imperative {
	r := NewRunner(RunnerConfig{Location: location})
	imperatives, _ := r.parseImperatives(content)
	return imperatives
}

// TokenPattern returns a regex pattern for matching schedule tokens.
func TokenPattern() *regexp.Regexp {
	return regexp.MustCompile(`@(?:today|tomorrow|yesterday|` +
		`(?:sun|mon|tue|wed|thu|fri|sat)(?:day)?|` +
		`\d{4}-\d{2}-\d{2}|` +
		`\d{1,2}(?::\d{2})?(?:am|pm)|` +
		`\d{1,2}:\d{2}|` +
		`in:\d+(?:hours?|mins?|minutes?|days?|weeks?|h|m|d|w)|` +
		`daily:\d{1,2}(?::\d{2})?(?:am|pm)?|` +
		`weekly:[\w,]+(?::\d{1,2}(?::\d{2})?(?:am|pm)?)?|` +
		`monthly:\d{1,2}(?:st|nd|rd|th)?(?::\d{1,2}(?::\d{2})?(?:am|pm)?)?|` +
		`yearly:[\w]+-\d{1,2}(?::\d{1,2}(?::\d{2})?(?:am|pm)?)?)`)
}
