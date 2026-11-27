package stream

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// listFixture is a list input whose command prints the lines of a file and
// counts its runs in another file, outside any watched folder.
type listFixture struct {
	h         *Hub
	list      string
	runCount  func() int
	in, merge *Buffer
	sub       *Subscription
}

func newList(t *testing.T, h *Hub, every time.Duration, lines string) *listFixture {
	t.Helper()
	f := &listFixture{h: h, list: filepath.Join(t.TempDir(), "list.txt")}
	f.write(t, lines)
	cmd, runs := countRuns(t, fmt.Sprintf("cat %q", f.list))
	f.runCount = runs
	in, err := h.Input(InputSpec{
		Name:    "files",
		Cmd:     cmd,
		Format:  "lines",
		Every:   every,
		Pattern: `^(?P<state>\S+) (?P<name>.+)$`,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.in = in
	if f.merge, err = h.Merge("m", []*Buffer{in}, 0, KeySpec{Paths: []string{"fields.name"}}); err != nil {
		t.Fatal(err)
	}
	f.sub = f.merge.Subscribe()
	t.Cleanup(f.sub.Close)
	return f
}

func (f *listFixture) write(t *testing.T, lines string) {
	t.Helper()
	if err := os.WriteFile(f.list, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f *listFixture) names() string {
	var out []string
	for _, r := range f.merge.Snapshot() {
		fields := r["fields"].(map[string]interface{})
		out = append(out, fmt.Sprint(fields["state"], " ", fields["name"]))
	}
	return strings.Join(out, ",")
}

func TestListRunReplacesRows(t *testing.T) {
	h := NewHub()
	defer h.Close()
	f := newList(t, h, 100*time.Millisecond, "M a.txt\n?? b.txt\n")
	waitFor(t, "first run's rows", func() bool { return f.names() == "M a.txt,?? b.txt" })

	// b.txt leaves, c.txt arrives, and a.txt changes state in place.
	f.write(t, "A a.txt\n?? c.txt\n")
	waitFor(t, "second run's rows", func() bool { return f.names() == "A a.txt,?? c.txt" })

	f.write(t, "")
	waitFor(t, "an empty list", func() bool { return f.names() == "" })
}

func TestListUnchangedRunWakesNothing(t *testing.T) {
	h := NewHub()
	defer h.Close()
	f := newList(t, h, 50*time.Millisecond, "M a.txt\n")
	waitFor(t, "first rows", func() bool { return f.names() == "M a.txt" })
	before := f.runCount()
	for range len(f.sub.C) {
		<-f.sub.C
	}
	waitFor(t, "three more runs", func() bool { return f.runCount() >= before+3 })
	select {
	case <-f.sub.C:
		t.Fatal("a run that printed the same lines woke the page")
	default:
	}
}

func TestListFailedRunKeepsRowsAndSetsError(t *testing.T) {
	h := NewHub()
	defer h.Close()
	f := newList(t, h, 50*time.Millisecond, "M a.txt\n")
	waitFor(t, "first rows", func() bool { return f.names() == "M a.txt" })

	os.Remove(f.list) // cat now fails
	waitFor(t, "the error", func() bool { return strings.Contains(f.merge.Err(), "No such file") })
	if got := f.names(); got != "M a.txt" {
		t.Fatalf("rows after a failed run = %q, want the last good rows", got)
	}

	f.write(t, "M a.txt\n") // the same lines as before the failure
	waitFor(t, "the error to clear", func() bool { return f.merge.Err() == "" })
	if got := f.names(); got != "M a.txt" {
		t.Fatalf("rows = %q", got)
	}
}

func TestRerunListsRunsAtOnce(t *testing.T) {
	h := NewHub()
	defer h.Close()
	f := newList(t, h, time.Hour, "M a.txt\n")
	waitFor(t, "first run", func() bool { return f.runCount() == 1 })
	f.write(t, "M b.txt\n")
	h.RerunLists()
	waitFor(t, "the rerun's rows", func() bool { return f.names() == "M b.txt" })
}

func TestClickEndRerunsLists(t *testing.T) {
	h := NewHub()
	defer h.Close()
	f := newList(t, h, time.Hour, "M a.txt\n")
	waitFor(t, "first run", func() bool { return f.runCount() == 1 })
	click, err := h.Input(InputSpec{Name: "commit", Cmd: fmt.Sprintf("printf 'A c.txt\\n' > %q", f.list), Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	sub := click.Subscribe()
	defer sub.Close()
	h.Restart(click)
	waitFor(t, "the list to rerun after the click's command", func() bool { return f.names() == "A c.txt" })
}

func TestListStopsWithoutSubscribers(t *testing.T) {
	h := NewHub()
	h.IdleStop = 50 * time.Millisecond
	defer h.Close()
	f := newList(t, h, 20*time.Millisecond, "M a.txt\n")
	waitFor(t, "a few runs", func() bool { return f.runCount() >= 3 })
	f.sub.Close()
	waitFor(t, "the input to stop", func() bool { return !f.in.Running() })
	n := f.runCount()
	time.Sleep(150 * time.Millisecond)
	if f.runCount() != n {
		t.Fatal("the list kept running with no page")
	}
}

func TestWatchDirRunsOnFileChanges(t *testing.T) {
	dir := t.TempDir()
	h := NewHub()
	h.WatchDir = dir
	defer h.Close()
	f := newList(t, h, time.Hour, "M a.txt\n")
	// The first run, then one more once the watch is up, for a change made
	// during the walk.
	waitFor(t, "the first run and the one after the walk", func() bool { return f.runCount() == 2 })
	time.Sleep(50 * time.Millisecond)

	os.WriteFile(filepath.Join(dir, "x"), []byte("1"), 0o644)
	waitFor(t, "a run after a write", func() bool { return f.runCount() == 3 })

	// A folder made after the watch started is watched too.
	sub := filepath.Join(dir, "new", ".hidden")
	os.MkdirAll(sub, 0o755)
	waitFor(t, "a run after the mkdir", func() bool { return f.runCount() == 4 })
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(filepath.Join(sub, "y"), []byte("1"), 0o644)
	waitFor(t, "a run after a write in the new folder", func() bool { return f.runCount() == 5 })

	// A burst of writes, all inside WatchQuiet, starts one run. They go
	// back to back: a slow CI runner under -race once spread 50 writes with
	// 1 ms sleeps past a 50 ms window and counted two runs.
	for i := range 50 {
		os.WriteFile(filepath.Join(dir, "x"), []byte(fmt.Sprint(i)), 0o644)
	}
	waitFor(t, "a run after the burst", func() bool { return f.runCount() == 6 })
	time.Sleep(2 * WatchQuiet)
	if n := f.runCount(); n != 6 {
		t.Fatalf("runs after a burst of 50 writes = %d, want 6", n)
	}
}

func TestWatchDirFallsBackToTimer(t *testing.T) {
	h := NewHub()
	h.WatchDir = filepath.Join(t.TempDir(), "missing")
	defer h.Close()
	f := newList(t, h, 30*time.Millisecond, "M a.txt\n")
	waitFor(t, "runs on the timer", func() bool { return f.runCount() >= 3 })
}

func TestPatternFields(t *testing.T) {
	spec, err := InputSpec{Name: "s", Cmd: "true", Pattern: `^(?P<x>.)(?P<y>.) (?P<path>.+)$`}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	r := spec.row(" M sp ace.txt", false, time.Now())
	fields := r["fields"].(map[string]interface{})
	if fields["x"] != " " || fields["y"] != "M" || fields["path"] != "sp ace.txt" {
		t.Fatalf("fields = %#v", fields)
	}
	// A line that doesn't match keeps what its format parsed.
	r = spec.row("a=1 b=2", false, time.Now())
	if f := r["fields"].(map[string]interface{}); f["a"] != "1" || f["path"] != nil {
		t.Fatalf("fields of a line the pattern misses = %#v", f)
	}
}

func TestSpecRejectsBadEveryAndPattern(t *testing.T) {
	for _, spec := range []InputSpec{
		{Name: "f", File: "/tmp/x", Every: time.Second},
		{Name: "m", Cmd: "true", Manual: true, Every: time.Second},
		{Name: "n", Cmd: "true", Restart: "never", Every: time.Second},
		{Name: "p", Cmd: "true", Pattern: `(.+)`},
		{Name: "q", Cmd: "true", Pattern: `(?P<a>`},
	} {
		if _, err := spec.Normalize(); err == nil {
			t.Errorf("%s: Normalize accepted %+v", spec.Name, spec)
		}
	}
}

func TestWatchDirTurnsTheTimerOff(t *testing.T) {
	h := NewHub()
	h.WatchDir = t.TempDir()
	defer h.Close()
	f := newList(t, h, 20*time.Millisecond, "M a.txt\n")
	waitFor(t, "the first run and the one after the walk", func() bool { return f.runCount() == 2 })
	time.Sleep(200 * time.Millisecond) // ten timer periods
	if n := f.runCount(); n != 2 {
		t.Fatalf("runs with --watch-dir and no file change = %d, want 2", n)
	}
}

func TestListRunsOnlyWhileAPageShowsIt(t *testing.T) {
	h := NewHub()
	defer h.Close()
	cmd, runs := countRuns(t, "true")
	in, err := h.Input(InputSpec{Name: "files", Cmd: cmd, Every: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if n := runs(); n != 0 {
		t.Fatalf("runs before any page subscribed = %d, want 0", n)
	}
	sub := in.Subscribe()
	defer sub.Close()
	waitFor(t, "runs once a page subscribes", func() bool { return runs() >= 2 })
}

func TestListRunTimesOut(t *testing.T) {
	old := ListRunTimeout
	ListRunTimeout = 300 * time.Millisecond
	defer func() { ListRunTimeout = old }()
	h := NewHub()
	defer h.Close()
	in, err := h.Input(InputSpec{Name: "stuck", Cmd: "sleep 10", Every: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	sub := in.Subscribe()
	defer sub.Close()
	waitFor(t, "the timeout", func() bool { return strings.Contains(in.Err(), "stopped after 300ms") })
}
