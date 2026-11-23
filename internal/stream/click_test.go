package stream

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// countRuns returns a command that appends a line to a file on each run, and
// a func reading how many runs there were.
func countRuns(t *testing.T, then string) (string, func() int) {
	path := filepath.Join(t.TempDir(), "runs")
	return fmt.Sprintf("echo run >> %q; %s", path, then), func() int {
		data, _ := os.ReadFile(path)
		return strings.Count(string(data), "run")
	}
}

func TestManualInputStartsOnlyOnRestart(t *testing.T) {
	h := NewHub()
	defer h.Close()
	h.IdleStop = 100 * time.Millisecond
	cmd, runs := countRuns(t, "echo done")
	in, err := h.Input(InputSpec{Name: "send", Cmd: cmd, Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := h.Merge("response", []*Buffer{in}, 0, KeySpec{})
	sub := m.Subscribe()
	time.Sleep(300 * time.Millisecond)
	if n := runs(); n != 0 {
		t.Fatalf("attaching ran the command %d times, want 0", n)
	}

	h.Restart(m)
	waitFor(t, "the run's row", func() bool { return strings.Join(msgs(m.Snapshot()), ",") == "done" })
	waitFor(t, "the run to end", func() bool { return !m.Running() })
	if n := runs(); n != 1 {
		t.Fatalf("runs = %d, want 1", n)
	}

	// After an idle stop, a page attaching again must not send it again.
	sub.Close()
	time.Sleep(300 * time.Millisecond)
	again := m.Subscribe()
	defer again.Close()
	time.Sleep(300 * time.Millisecond)
	if n := runs(); n != 1 {
		t.Fatalf("re-attaching after an idle stop ran it again: runs = %d, want 1", n)
	}
}

func TestRestartDropsTheOldRunsRows(t *testing.T) {
	marker := fmt.Sprintf("restartmark%d", os.Getpid())
	h := NewHub()
	defer h.Close()
	in, _ := h.Input(InputSpec{Name: "send", Cmd: ": " + marker + "; echo start; sleep 0.5; echo late", Manual: true})
	m, _ := h.Merge("response", []*Buffer{in}, 0, KeySpec{})
	sub := m.Subscribe()
	defer sub.Close()

	h.Restart(m)
	waitFor(t, "the first run's start", func() bool { return len(m.Snapshot()) == 1 })
	if !m.Running() {
		t.Fatal("Running = false during the run")
	}
	h.Restart(m) // a second click mid-run
	waitFor(t, "the second run to end", func() bool { return !m.Running() })
	if got := strings.Join(msgs(m.Snapshot()), ","); got != "start,late" {
		t.Fatalf("rows = %s, want start,late from the second run only", got)
	}
	if exec.Command("pgrep", "-f", marker).Run() == nil {
		t.Fatal("a run's process outlived it")
	}
}

func TestManualRunReportsOnlyFailure(t *testing.T) {
	h := NewHub()
	defer h.Close()
	for cmd, want := range map[string]string{"true": "", "exit 3": "process exit status 3"} {
		in, _ := h.Input(InputSpec{Name: "x", Cmd: cmd, Manual: true})
		sub := in.Subscribe()
		h.Restart(in)
		waitFor(t, cmd+" to end", func() bool { return !in.Running() })
		if got := strings.Join(msgs(in.Snapshot()), ","); got != want {
			t.Errorf("%s: rows = %q, want %q", cmd, got, want)
		}
		sub.Close()
	}
}

func TestManualNeedsCmd(t *testing.T) {
	h := NewHub()
	defer h.Close()
	if _, err := h.Input(InputSpec{Name: "x", File: "/tmp/x", Manual: true}); err == nil {
		t.Fatal("a manual file input was accepted")
	}
	if _, err := h.Input(InputSpec{Name: "x", Cmd: "true", LineMax: MaxLineMax + 1}); err == nil {
		t.Fatal("line_max over the cap was accepted")
	}
}

func TestLineMaxKeepsALongLine(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in, _ := h.Input(InputSpec{Name: "x", Cmd: "head -c 200000 /dev/zero | tr '\\0' a; echo", Manual: true, Format: "lines", LineMax: 1 << 20})
	sub := in.Subscribe()
	defer sub.Close()
	h.Restart(in)
	waitFor(t, "the line", func() bool { return len(in.Snapshot()) == 1 })
	r := in.Snapshot()[0]
	if len(r["line"].(string)) != 200000 || r["truncated"] != false {
		t.Fatalf("line of %d bytes, truncated=%v; want 200000, false", len(r["line"].(string)), r["truncated"])
	}
}

func TestReplaceSwapsRowsAndPrivateValues(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "in", "api", 10)
	other := newBuffer(h, "other", "other", 10)
	m, _ := h.Merge("requests", []*Buffer{in, other}, 0, KeySpec{Paths: []string{"source", "fields.entry"}})
	req := func(entry string) Row { return Row{"fields": map[string]interface{}{"entry": entry}} }
	secret := func(v string) Private { return Private{Values: map[string]string{"token": v}} }

	h.publish(bg, other, "stdout", req("1"))
	h.replace(bg, in, "file", []Row{req("1"), req("2")}, []Private{secret("a"), {Err: "unset variable x"}})
	rows := m.Snapshot()
	if len(rows) != 3 {
		t.Fatalf("merge rows = %d, want 3", len(rows))
	}
	for _, r := range rows {
		if strings.Contains(fmt.Sprint(r), "token") {
			t.Fatalf("a private value is in a row: %v", r)
		}
	}
	if p, ok := h.Private(rows[1]); !ok || p.Values["token"] != "a" {
		t.Fatalf("Private(keyed copy) = %v, %v; want token a", p, ok)
	}
	if p, _ := h.Private(rows[2]); p.Err != "unset variable x" {
		t.Fatalf("Private(row 2).Err = %q", p.Err)
	}

	old := rows[1]
	h.replace(bg, in, "file", []Row{req("1")}, []Private{secret("b")})
	rows = m.Snapshot()
	if len(rows) != 2 || rows[0]["source"] != "other" {
		t.Fatalf("after replace: %v, want other's row and one api row", rows)
	}
	if _, ok := h.Private(old); ok {
		t.Fatal("a replaced row's private values were kept")
	}
	if p, _ := h.Private(rows[1]); p.Values["token"] != "b" {
		t.Fatalf("new private value = %v, want b", p)
	}
}

func TestInputErrReachesItsMerge(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "in", "api", 10)
	m, _ := h.Merge("requests", []*Buffer{in}, 0, KeySpec{})
	sub := m.Subscribe()
	defer sub.Close()
	h.setErr(bg, in, "line 3: bad")
	if got := m.Err(); got != "api: line 3: bad" {
		t.Fatalf("merge Err = %q", got)
	}
	if len(sub.C) != 1 {
		t.Fatal("setting an error didn't wake the merge's page")
	}
	h.setErr(bg, in, "")
	if got := m.Err(); got != "" {
		t.Fatalf("cleared Err = %q", got)
	}
}
