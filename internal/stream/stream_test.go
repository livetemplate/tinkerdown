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

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func msgs(rows []Row) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = fmt.Sprint(r["msg"])
	}
	return out
}

func TestRingEvictsOldest(t *testing.T) {
	h := NewHub()
	defer h.Close()
	b := newBuffer(h, "k", "s", 3)
	for i := 1; i <= 5; i++ {
		h.publish(b, "stdout", Row{"msg": fmt.Sprint(i)})
	}
	if got := strings.Join(msgs(b.Snapshot()), ","); got != "3,4,5" {
		t.Fatalf("snapshot = %s, want 3,4,5", got)
	}
}

func TestMergeKeepsSeqOrderAndSeedsHistory(t *testing.T) {
	h := NewHub()
	defer h.Close()
	a, c := newBuffer(h, "ka", "a", 10), newBuffer(h, "kc", "c", 10)
	h.publish(a, "stdout", Row{"msg": "a1"})
	h.publish(c, "stdout", Row{"msg": "c1"})
	m, err := h.Merge("m", []*Buffer{a, c}, 10)
	if err != nil {
		t.Fatal(err)
	}
	h.publish(c, "stdout", Row{"msg": "c2"})
	h.publish(a, "stdout", Row{"msg": "a2"})

	rows := m.Snapshot()
	if got := strings.Join(msgs(rows), ","); got != "a1,c1,c2,a2" {
		t.Fatalf("merge = %s, want a1,c1,c2,a2", got)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i]["seq"].(int64) <= rows[i-1]["seq"].(int64) {
			t.Fatalf("seq not increasing at %d: %v", i, rows)
		}
	}
	if rows[0]["source"] != "a" || rows[1]["source"] != "c" {
		t.Fatalf("source not stamped: %v", rows[:2])
	}
}

func TestSubscriptionCoalescesWakeups(t *testing.T) {
	h := NewHub()
	defer h.Close()
	b := newBuffer(h, "k", "s", 10)
	s := b.Subscribe()
	for i := 0; i < 100; i++ {
		h.publish(b, "stdout", Row{}) // must never block
	}
	if len(s.C) != 1 {
		t.Fatalf("pending wake-ups = %d, want 1", len(s.C))
	}
	s.Close()
	h.publish(b, "stdout", Row{})
}

func TestLineSplitterTruncatesHugeLineAcrossChunks(t *testing.T) {
	var got []string
	var trunc []bool
	split := &lineSplitter{emit: func(l string, tr bool) {
		got = append(got, l)
		trunc = append(trunc, tr)
	}}
	in := "first\r\n" + strings.Repeat("x", 1<<20) + "\nlast\npart"
	for len(in) > 0 { // feed in odd-sized chunks so lines straddle writes
		n := min(len(in), 4093)
		split.Write([]byte(in[:n]))
		in = in[n:]
	}
	if len(got) != 3 || got[0] != "first" || got[2] != "last" {
		t.Fatalf("lines = %d: %q … %q", len(got), got[0], got[len(got)-1])
	}
	if len(got[1]) != MaxLineBytes || !trunc[1] || trunc[0] || trunc[2] {
		t.Fatalf("huge line: len %d truncated %v", len(got[1]), trunc)
	}
	split.Flush()
	if got[len(got)-1] != "part" {
		t.Fatalf("Flush did not emit the partial line: %q", got)
	}
}

func TestNewRowFormats(t *testing.T) {
	now := time.Date(2026, 11, 20, 17, 2, 11, 0, time.UTC)
	cases := []struct {
		name, line, format, level, msg string
		field                          string
	}{
		{"json", `{"level":"WARNING","msg":"slow query","ms":812}`, "auto", "warn", "slow query", "ms"},
		{"logfmt", `level=error msg="nil map" path=/api`, "auto", "error", "nil map", "path"},
		{"plain", `GET /api/users 200`, "auto", "", "GET /api/users 200", ""},
		{"prose with one equals stays plain", `retry count=3 reached`, "auto", "", "retry count=3 reached", ""},
		{"docker timestamp prefix", `2026-11-20T17:02:11.5Z {"level":"info","msg":"up"}`, "auto", "info", "up", ""},
		{"timestamp prefix with format json", `2026-11-20T17:02:11.5Z {"level":"error","msg":"x"}`, "json", "error", "x", ""},
		{"lines skips parsing", `{"level":"error"}`, "lines", "", `{"level":"error"}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newRow(c.line, false, c.format, now)
			if r["level"] != c.level || r["msg"] != c.msg || r["line"] != c.line {
				t.Fatalf("got level=%q msg=%q line=%q", r["level"], r["msg"], r["line"])
			}
			fields, _ := r["fields"].(map[string]interface{})
			if c.field != "" && fields[c.field] == nil {
				t.Fatalf("field %q missing from %v", c.field, fields)
			}
			if _, leaked := fields["level"]; leaked {
				t.Fatalf("level left in fields: %v", fields)
			}
		})
	}
	if r := newRow(`2026-11-20T17:02:11.5Z hi`, false, "auto", now); r["ts"] != "2026-11-20T17:02:11.5Z" {
		t.Fatalf("ts = %v, want the prefixed timestamp", r["ts"])
	}
}

func TestCmdTagsStreamsAndRestartsWithSystemRow(t *testing.T) {
	h := NewHub()
	defer h.Close()
	b, err := h.Input(InputSpec{Name: "demo", Cmd: `echo out; echo err 1>&2; exit 3`})
	if err != nil {
		t.Fatal(err)
	}
	b.Subscribe()
	waitFor(t, "restart row and second run", func() bool {
		n := 0
		for _, r := range b.Snapshot() {
			if r["msg"] == "out" {
				n++
			}
		}
		return n >= 2
	})
	var sawErr, sawSystem bool
	for _, r := range b.Snapshot() {
		if r["msg"] == "err" && r["stream"] == "stderr" {
			sawErr = true
		}
		if r["stream"] == "system" && strings.Contains(r["msg"].(string), "restarting in 1s") {
			sawSystem = true
		}
	}
	if !sawErr || !sawSystem {
		t.Fatalf("stderr row %v, system restart row %v: %v", sawErr, sawSystem, msgs(b.Snapshot()))
	}
}

func TestRestartNeverStopsAfterExit(t *testing.T) {
	h := NewHub()
	defer h.Close()
	b, _ := h.Input(InputSpec{Name: "once", Cmd: "echo hi", Restart: "never"})
	b.Subscribe()
	waitFor(t, "exit row", func() bool { return len(b.Snapshot()) == 2 })
	time.Sleep(1500 * time.Millisecond)
	if n := len(b.Snapshot()); n != 2 {
		t.Fatalf("rows = %d after exit with restart: never, want 2: %v", n, msgs(b.Snapshot()))
	}
}

func TestCloseKillsProcessGroup(t *testing.T) {
	marker := fmt.Sprintf("%d", 3000+os.Getpid()%1000)
	h := NewHub()
	b, _ := h.Input(InputSpec{Name: "pg", Cmd: "sleep " + marker + "1 & sleep " + marker + "2; echo never"})
	b.Subscribe()
	waitFor(t, "children to start", func() bool {
		return exec.Command("pgrep", "-f", "sleep "+marker+"1").Run() == nil
	})
	h.Close()
	waitFor(t, "children to die", func() bool {
		return exec.Command("pgrep", "-f", "sleep "+marker).Run() != nil
	})
}

func TestTailBacklogAppendRotateTruncate(t *testing.T) {
	old := pollInterval
	pollInterval = 20 * time.Millisecond
	defer func() { pollInterval = old }()

	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	os.WriteFile(path, []byte("l1\nl2\nl3\nl4\n"), 0o644)

	h := NewHub()
	defer h.Close()
	b, _ := h.Input(InputSpec{Name: "app", File: path, Backlog: 2})
	b.Subscribe()
	has := func(m string) func() bool {
		return func() bool {
			for _, r := range b.Snapshot() {
				if r["msg"] == m {
					return true
				}
			}
			return false
		}
	}
	waitFor(t, "backlog", has("l4"))
	if got := strings.Join(msgs(b.Snapshot()), ","); got != "l3,l4" {
		t.Fatalf("backlog = %s, want l3,l4", got)
	}

	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString("par")
	f.Sync()
	time.Sleep(100 * time.Millisecond)
	if has("par")() {
		t.Fatal("partial line published before its newline")
	}
	f.WriteString("tial\n")
	f.Close()
	waitFor(t, "appended line", has("partial"))

	os.WriteFile(path, []byte("t1\n"), 0o644) // shrink in place
	waitFor(t, "line after truncation", has("t1"))

	os.Rename(path, path+".1")
	os.WriteFile(path, []byte("r1\n"), 0o644)
	waitFor(t, "line after rotation", has("r1"))

	var system []string
	for _, r := range b.Snapshot() {
		if r["stream"] == "system" {
			system = append(system, r["msg"].(string))
		}
	}
	if len(system) != 2 || !strings.Contains(system[0], "truncated") || !strings.Contains(system[1], "rotated") {
		t.Fatalf("system rows = %q", system)
	}
}

func TestInputRejectsBothOrNeither(t *testing.T) {
	h := NewHub()
	defer h.Close()
	for name, spec := range map[string]InputSpec{
		"no cmd or file":    {Name: "x"},
		"both cmd and file": {Name: "x", Cmd: "true", File: "/tmp/x"},
		"unknown format":    {Name: "x", Cmd: "true", Format: "yaml"},
		"unknown restart":   {Name: "x", Cmd: "true", Restart: "always"},
	} {
		if _, err := h.Input(spec); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestInputKeyFollowsSpec(t *testing.T) {
	h := NewHub()
	defer h.Close()
	a, _ := h.Input(InputSpec{Name: "w", Cmd: "true", Env: map[string]string{"A": "1", "B": "2"}})
	b, _ := h.Input(InputSpec{Name: "w", Cmd: "true", Format: "auto", Env: map[string]string{"B": "2", "A": "1"}})
	c, _ := h.Input(InputSpec{Name: "w", Cmd: "false"})
	if a != b {
		t.Fatal("equal specs (after defaults) got different buffers")
	}
	if a == c {
		t.Fatal("different specs share a buffer")
	}
	m, _ := h.Merge("all", []*Buffer{a, c}, 0)
	if _, err := h.Merge("nested", []*Buffer{m}, 0); err == nil {
		t.Fatal("merge of a merge: want error")
	}
	if m2, _ := h.Merge("all", []*Buffer{a, c}, 0); m2 != m {
		t.Fatal("same merge got a new buffer")
	}
}

func TestIdleInputStopsAndRestarts(t *testing.T) {
	marker := fmt.Sprintf("idlemark%d", os.Getpid())
	h := NewHub()
	defer h.Close()
	h.IdleStop = 200 * time.Millisecond
	running := func() bool { return exec.Command("pgrep", "-f", marker).Run() == nil }

	in, _ := h.Input(InputSpec{Name: "w", Cmd: ": " + marker + "; while true; do echo hi; sleep 0.1; done"})
	m, _ := h.Merge("all", []*Buffer{in}, 0)
	sub := m.Subscribe() // a merge subscriber holds its input
	waitFor(t, "input to start through the merge", running)

	sub.Close()
	sub.Close() // idempotent: must not release twice
	waitFor(t, "idle input to stop", func() bool { return !running() })
	if n := len(in.Snapshot()); n != 0 {
		t.Fatalf("stopped input kept %d rows", n)
	}

	again := in.Subscribe()
	defer again.Close()
	waitFor(t, "input to restart on a new subscriber", running)

	// A subscriber that returns within IdleStop keeps the same process.
	again.Close()
	keep := in.Subscribe()
	defer keep.Close()
	time.Sleep(400 * time.Millisecond)
	if !running() {
		t.Fatal("input stopped although a subscriber came back within IdleStop")
	}
}

func TestDetectLevel(t *testing.T) {
	cases := map[string]string{
		"ERROR disk full":                          "error",
		"[warn] slow query":                        "warn",
		"WARN: retrying":                           "warn",
		"Info: started":                            "info",
		"2026/11/20 19:02:11 ERROR db down":        "error",
		"2026-11-20 19:02:11,123 DEBUG cache miss": "debug",
		"<3>kernel: oops":                          "error",
		"<12>user notice":                          "warn",
		"E1120 19:02:11.123 server.go:42] failed":  "error",
		"I1120 19:02:11.123 server.go:42] started": "info",
		"panic: runtime error: index out of range": "fatal",
		"fatal error: all goroutines are asleep":   "fatal",
		"no errors found":                          "",
		"retrying after error":                     "",
		"GET /api/users 200":                       "",
		"":                                         "",
		"Error rate is 0.1%":                       "error",
	}
	for line, want := range cases {
		if got := normalizeLevel(detectLevel(line)); got != want {
			t.Errorf("detectLevel(%q) = %q, want %q", line, got, want)
		}
	}
	// format: auto uses it for plain lines; other formats don't guess.
	if r := newRow("ERROR boom", false, "auto", time.Now()); r["level"] != "error" {
		t.Errorf("auto plain line level = %v", r["level"])
	}
	if r := newRow("ERROR boom", false, "lines", time.Now()); r["level"] != "" {
		t.Errorf("lines format guessed level %v", r["level"])
	}
}
