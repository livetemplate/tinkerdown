package schedule

import (
	"context"
	"sync"
	"time"
)

// Job represents a scheduled job to be executed.
type Job struct {
	ID           string
	PageID       string
	Line         string // The imperative line (e.g., "Notify @daily:9am Check email")
	Token        *Token
	NextRun      time.Time
	LastRun      *time.Time
	Enabled      bool
	Handler      JobHandler
}

// JobHandler is called when a job should execute.
type JobHandler func(job *Job) error

// Cron manages scheduled jobs and their execution timing.
type Cron struct {
	mu       sync.RWMutex
	jobs     map[string]*Job
	location *time.Location
	ticker   *time.Ticker
	done     chan struct{}
	running  bool

	// For testing: allows injecting a custom time source
	nowFunc func() time.Time
}

// NewCron creates a new cron scheduler with the specified timezone.
func NewCron(location *time.Location) *Cron {
	if location == nil {
		location = time.Local
	}
	return &Cron{
		jobs:     make(map[string]*Job),
		location: location,
		nowFunc:  time.Now,
	}
}

// SetTimeFunc sets a custom time function (for testing).
func (c *Cron) SetTimeFunc(fn func() time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nowFunc = fn
}

// now returns the current time using the configured time function.
func (c *Cron) now() time.Time {
	return c.nowFunc().In(c.location)
}

// AddJob registers a new job with the scheduler.
func (c *Cron) AddJob(job *Job) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if job.Token != nil {
		job.NextRun = job.Token.NextOccurrence(c.now(), c.location)
	}
	job.Enabled = true
	c.jobs[job.ID] = job
}

// RemoveJob removes a job from the scheduler.
func (c *Cron) RemoveJob(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.jobs, id)
}

// RemoveJobsByPage removes all jobs for a specific page.
func (c *Cron) RemoveJobsByPage(pageID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, job := range c.jobs {
		if job.PageID == pageID {
			delete(c.jobs, id)
		}
	}
}

// GetJob returns a job by ID.
func (c *Cron) GetJob(id string) *Job {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jobs[id]
}

// GetJobs returns all registered jobs.
func (c *Cron) GetJobs() []*Job {
	c.mu.RLock()
	defer c.mu.RUnlock()

	jobs := make([]*Job, 0, len(c.jobs))
	for _, job := range c.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetJobsByPage returns all jobs for a specific page.
func (c *Cron) GetJobsByPage(pageID string) []*Job {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var jobs []*Job
	for _, job := range c.jobs {
		if job.PageID == pageID {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// Start begins the cron scheduler's ticker loop.
func (c *Cron) Start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.ticker = time.NewTicker(time.Minute)
	c.done = make(chan struct{})
	c.mu.Unlock()

	go c.run(ctx)
}

// Stop halts the cron scheduler.
func (c *Cron) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	c.running = false
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.done)
}

// run is the main ticker loop.
func (c *Cron) run(ctx context.Context) {
	// Check immediately on start
	c.checkAndExecute()

	for {
		select {
		case <-ctx.Done():
			c.Stop()
			return
		case <-c.done:
			return
		case <-c.ticker.C:
			c.checkAndExecute()
		}
	}
}

// checkAndExecute checks all jobs and executes those that are due.
func (c *Cron) checkAndExecute() {
	now := c.now()

	c.mu.RLock()
	jobsToRun := make([]*Job, 0)
	for _, job := range c.jobs {
		if job.Enabled && !job.NextRun.After(now) {
			jobsToRun = append(jobsToRun, job)
		}
	}
	c.mu.RUnlock()

	for _, job := range jobsToRun {
		c.executeJob(job, now)
	}
}

// executeJob runs a single job and updates its schedule.
func (c *Cron) executeJob(job *Job, now time.Time) {
	if job.Handler != nil {
		if err := job.Handler(job); err != nil {
			// Log error but continue (graceful degradation)
			// In a real implementation, this would use a proper logger
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Update last run time
	job.LastRun = &now

	// Calculate next occurrence for recurring schedules
	if job.Token != nil {
		switch job.Token.Type {
		case TokenDaily, TokenWeekly, TokenMonthly, TokenYearly:
			// Recurring schedules - calculate next occurrence
			// Use a time slightly after now to avoid re-triggering
			nextCheckTime := now.Add(time.Minute)
			job.NextRun = job.Token.NextOccurrence(nextCheckTime, c.location)
		default:
			// One-time schedules - disable the job
			job.Enabled = false
		}
	}
}

// Tick manually triggers a check cycle (for testing).
func (c *Cron) Tick() {
	c.checkAndExecute()
}

// IsRunning returns whether the scheduler is currently running.
func (c *Cron) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// JobCount returns the number of registered jobs.
func (c *Cron) JobCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.jobs)
}

// EnableJob enables a previously disabled job.
func (c *Cron) EnableJob(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	job, ok := c.jobs[id]
	if !ok {
		return false
	}

	job.Enabled = true
	if job.Token != nil {
		job.NextRun = job.Token.NextOccurrence(c.now(), c.location)
	}
	return true
}

// DisableJob disables a job without removing it.
func (c *Cron) DisableJob(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	job, ok := c.jobs[id]
	if !ok {
		return false
	}

	job.Enabled = false
	return true
}

// UpdateJobToken updates a job's schedule token and recalculates next run.
func (c *Cron) UpdateJobToken(id string, token *Token) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	job, ok := c.jobs[id]
	if !ok {
		return false
	}

	job.Token = token
	if token != nil && job.Enabled {
		job.NextRun = token.NextOccurrence(c.now(), c.location)
	}
	return true
}
