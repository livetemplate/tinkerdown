package stream

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	minBackoff = time.Second
	maxBackoff = 30 * time.Second
	// healthyRun is how long a command must run before its backoff resets.
	healthyRun = 30 * time.Second
)

// runCmd runs spec.Cmd under /bin/sh -c until ctx ends, publishing each
// stdout and stderr line, and restarts it with backoff when it exits unless
// spec.Restart is "never".
func runCmd(ctx context.Context, spec InputSpec, b *Buffer) {
	backoff := minBackoff
	for {
		started := time.Now()
		err := runOnce(ctx, spec, b)
		if ctx.Err() != nil {
			return
		}
		status := "exited"
		if err != nil {
			status = err.Error()
		}
		if spec.Restart == "never" {
			// A manual run is expected to end; only a failure is news.
			if !spec.Manual || err != nil {
				b.system(ctx, "warn", "process "+status)
			}
			return
		}
		if time.Since(started) >= healthyRun {
			backoff = minBackoff
		}
		b.system(ctx, "warn", fmt.Sprintf("process %s, restarting in %s", status, backoff))
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

// shellCmd builds spec.Cmd under /bin/sh -c, in its own process group, so
// ending ctx kills the whole pipeline, not only sh.
func shellCmd(ctx context.Context, spec InputSpec) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", spec.Cmd)
	cmd.Dir = spec.Dir
	cmd.Env = os.Environ()
	for k, v := range spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
	return cmd
}

func runOnce(ctx context.Context, spec InputSpec, b *Buffer) error {
	cmd := shellCmd(ctx, spec)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Go(func() { pump(ctx, stdout, spec, "stdout", b) })
	wg.Go(func() { pump(ctx, stderr, spec, "stderr", b) })
	wg.Wait() // both pipes hit EOF before Wait closes them
	return cmd.Wait()
}

// pump copies r into a line splitter and publishes each read's lines as one
// batch, so a burst costs one lock and one wake-up.
func pump(ctx context.Context, r io.Reader, spec InputSpec, stream string, b *Buffer) {
	var batch []Row
	split := &lineSplitter{max: spec.LineMax, emit: func(line string, truncated bool) {
		batch = append(batch, spec.row(line, truncated, b.hub.now()))
	}}
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		split.Write(buf[:n])
		if err != nil {
			split.Flush()
		}
		b.hub.publish(ctx, b, stream, batch...)
		batch = batch[:0]
		if err != nil {
			return
		}
	}
}

// listRun is what one run of a list input printed.
type listRun struct {
	lines     []string // stdout, one entry per line, at most spec.Buffer of them; never nil
	truncated []bool   // whether each line was cut at spec.LineMax
	stderr    string   // the end of stderr, at most MaxCollected bytes
	err       error    // why the command failed; nil when it exited 0
	spec      InputSpec
	now       time.Time // when the run ended, the time of its rows
}

// rows parses the run's lines. settle calls it only for a run that changed
// the list, since most runs print what the last one did.
func (r listRun) rows() []Row {
	out := make([]Row, len(r.lines))
	for i, line := range r.lines {
		out[i] = r.spec.row(line, r.truncated[i], r.now)
	}
	return out
}

// runList runs a list input's command until ctx ends: once, then again
// spec.Every after each run exits, or with the hub's WatchDir when a file
// there changes, and at once after RerunLists. settle shows each run.
func runList(ctx context.Context, spec InputSpec, b *Buffer) {
	var shown []string
	watching := false
	for first := true; ; first = false {
		run := collectRun(ctx, spec, b.hub.now)
		if ctx.Err() != nil {
			return
		}
		shown = b.settle(ctx, run, shown)
		if first && b.hub.WatchDir != "" {
			// After the first run, so the page's first rows don't wait for
			// the walk over a large folder.
			stop, err := b.hub.watchFiles()
			if err != nil {
				log.Printf("[stream] %s: can't watch %s, so it runs every %s instead: %v", spec.Name, b.hub.WatchDir, spec.Every, err)
			} else {
				defer stop()
				watching = true
				// One more run catches a change made during the walk, which
				// no event reports.
				select {
				case b.kick <- struct{}{}:
				default:
				}
			}
		}
		var timer <-chan time.Time
		if !watching {
			timer = time.After(spec.Every)
		}
		select {
		case <-ctx.Done():
			return
		case <-b.kick: // RerunLists, from a click or the watch
		case <-timer:
		}
	}
}

// collectRun runs spec.Cmd once and gathers what it printed.
func collectRun(ctx context.Context, spec InputSpec, now func() time.Time) listRun {
	run := listRun{lines: []string{}, spec: spec}
	split := &lineSplitter{max: spec.LineMax, emit: func(line string, truncated bool) {
		if len(run.lines) < spec.Buffer {
			run.lines = append(run.lines, line)
			run.truncated = append(run.truncated, truncated)
		}
	}}
	stderr := &tailWriter{max: MaxCollected}
	cmd := shellCmd(ctx, spec)
	cmd.Stdout, cmd.Stderr = split, stderr
	run.err = cmd.Run()
	split.Flush()
	run.stderr, run.now = string(stderr.buf), now()
	return run
}

// tailWriter keeps the last max bytes written to it, where a command prints
// why it failed.
type tailWriter struct {
	max int
	buf []byte
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	if over := len(w.buf) - w.max; over > 0 {
		w.buf = w.buf[over:]
	}
	return len(p), nil
}
