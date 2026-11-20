package stream

import (
	"context"
	"fmt"
	"io"
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
			b.system("warn", "process "+status)
			return
		}
		if time.Since(started) >= healthyRun {
			backoff = minBackoff
		}
		b.system("warn", fmt.Sprintf("process %s, restarting in %s", status, backoff))
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func runOnce(ctx context.Context, spec InputSpec, b *Buffer) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", spec.Cmd)
	cmd.Dir = spec.Dir
	cmd.Env = os.Environ()
	for k, v := range spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	// Own process group, so shutdown kills the whole pipeline, not only sh.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second

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
	wg.Go(func() { pump(stdout, spec.Format, "stdout", b) })
	wg.Go(func() { pump(stderr, spec.Format, "stderr", b) })
	wg.Wait() // both pipes hit EOF before Wait closes them
	return cmd.Wait()
}

// pump copies r into a line splitter and publishes each read's lines as one
// batch, so a burst costs one lock and one wake-up.
func pump(r io.Reader, format, stream string, b *Buffer) {
	var batch []Row
	split := &lineSplitter{emit: func(line string, truncated bool) {
		batch = append(batch, newRow(line, truncated, format, b.hub.now()))
	}}
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		split.Write(buf[:n])
		if err != nil {
			split.Flush()
		}
		b.hub.publish(b, stream, batch...)
		batch = batch[:0]
		if err != nil {
			return
		}
	}
}
