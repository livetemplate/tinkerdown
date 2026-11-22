package stream

import (
	"context"
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
	m, err := h.Merge("m", []*Buffer{a, c}, 10, KeySpec{})
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
	m, _ := h.Merge("all", []*Buffer{a, c}, 0, KeySpec{})
	if _, err := h.Merge("nested", []*Buffer{m}, 0, KeySpec{}); err == nil {
		t.Fatal("merge of a merge: want error")
	}
	if m2, _ := h.Merge("all", []*Buffer{a, c}, 0, KeySpec{}); m2 != m {
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
	m, _ := h.Merge("all", []*Buffer{in}, 0, KeySpec{})
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
		"<-1>crafted":                              "",
		"<+3>crafted":                              "",
		"<999>out of range":                        "",
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

func TestPanickingInputDoesNotCrash(t *testing.T) {
	h := NewHub()
	defer h.Close()
	b := newBuffer(h, "k", "boom", 10)
	b.run = func(ctx context.Context) { panic("parser bug") }
	b.Subscribe()
	waitFor(t, "a system row reporting the panic", func() bool {
		for _, r := range b.Snapshot() {
			if r["stream"] == "system" && strings.Contains(r["msg"].(string), "parser bug") {
				return true
			}
		}
		return false
	})
}

// event is a parsed go-test-like JSON row: fields hold Package, Test, Action
// and Output, as newRow would leave them.
func event(pkg, test, action, output string) Row {
	f := map[string]interface{}{"Package": pkg, "Action": action}
	if test != "" {
		f["Test"] = test
	}
	if output != "" {
		f["Output"] = output
	}
	return Row{"msg": action, "fields": f}
}

func TestKeyedMergeKeepsOneRowPerKey(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "kin", "go", 100)
	m, err := h.Merge("tests", []*Buffer{in}, 3, KeySpec{Paths: []string{"fields.Package", "fields.Test"}, Collect: "fields.Output"})
	if err != nil {
		t.Fatal(err)
	}
	h.publish(in, "stdout", event("api", "TestA", "run", ""))
	h.publish(in, "stdout", event("api", "TestB", "run", ""))
	h.publish(in, "stdout", event("api", "TestA", "output", "a says hi\n"))
	h.publish(in, "stdout", event("api", "TestA", "pass", ""))
	h.publish(in, "stdout", event("api", "", "pass", "")) // the package's own row

	rows := m.Snapshot()
	if got := strings.Join(msgs(rows), ","); got != "pass,run,pass" {
		t.Fatalf("keyed rows = %s, want TestA (pass), TestB (run), package (pass) in first-seen order", got)
	}
	if rows[0]["collected"] != "a says hi\n" || rows[1]["collected"] != "" {
		t.Fatalf("collected = %q, %q", rows[0]["collected"], rows[1]["collected"])
	}
	firstKey := rows[0]["key"]
	h.publish(in, "stdout", event("api", "TestA", "output", "again\n"))
	if rows = m.Snapshot(); rows[0]["key"] != firstKey || rows[0]["collected"] != "a says hi\nagain\n" {
		t.Fatalf("update in place: key %v (was %v), collected %q", rows[0]["key"], firstKey, rows[0]["collected"])
	}

	// Past buffer (3 keys), the key updated least recently goes: TestB.
	h.publish(in, "stdout", event("api", "TestC", "run", ""))
	if got := strings.Join(keysTests(m.Snapshot()), ","); got != "TestA,,TestC" {
		t.Fatalf("after eviction = %s, want TestA, the package row, TestC", got)
	}
}

func TestKeyedCollectKeepsTheEnd(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "kin2", "go", 10)
	m, _ := h.Merge("t", []*Buffer{in}, 10, KeySpec{Paths: []string{"fields.Test"}, Collect: "fields.Output"})
	h.publish(in, "stdout", event("p", "T", "output", strings.Repeat("x", MaxCollected)))
	h.publish(in, "stdout", event("p", "T", "output", "END"))
	c := m.Snapshot()[0]["collected"].(string)
	if len(c) != MaxCollected || !strings.HasSuffix(c, "END") {
		t.Fatalf("collected is %d bytes ending %q", len(c), c[len(c)-3:])
	}
}

func keysTests(rows []Row) []string {
	var out []string
	for _, r := range rows {
		test, _ := FieldPath(r, "fields.Test").(string)
		out = append(out, test)
	}
	return out
}

// A template reads .fields.NAME on every row, so plain-text and system rows
// need a fields map too, or rendering fails on a nil pointer.
func TestEveryRowHasFields(t *testing.T) {
	plain := newRow("just text", false, "auto", time.Now())
	sys := makeRow(time.Now(), "warn", "process exited", "process exited", false, nil)
	for name, r := range map[string]Row{"plain": plain, "system": sys} {
		if f, ok := r["fields"].(map[string]interface{}); !ok || f == nil {
			t.Errorf("%s row: fields = %#v, want an empty map", name, r["fields"])
		}
	}
}

// A test's next run must not append to its last run's output.
func TestKeyedCollectReset(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "kin3", "go", 10)
	m, _ := h.Merge("t", []*Buffer{in}, 10, KeySpec{Paths: []string{"fields.Test"}, Collect: "fields.Output", Reset: Match{Path: "fields.Action", Values: []string{"run", "start"}}})
	for _, run := range []string{"first\n", "second\n"} {
		h.publish(in, "stdout", event("p", "T", "run", ""))
		h.publish(in, "stdout", event("p", "T", "output", run))
		h.publish(in, "stdout", event("p", "T", "fail", ""))
	}
	if got := m.Snapshot()[0]["collected"]; got != "second\n" {
		t.Fatalf("collected = %q, want only the latest run", got)
	}
}

func TestKeyedCollectResetAlternatives(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "kin4", "go", 10)
	m, _ := h.Merge("t", []*Buffer{in}, 10, KeySpec{Paths: []string{"fields.Test"}, Collect: "fields.Output", Reset: Match{Path: "fields.Action", Values: []string{"run", "start"}}})
	for _, run := range []string{"FAIL one\n", "FAIL two\n"} {
		h.publish(in, "stdout", event("p", "", "start", "")) // a package row: no run event
		h.publish(in, "stdout", event("p", "", "output", run))
	}
	if got := m.Snapshot()[0]["collected"]; got != "FAIL two\n" {
		t.Fatalf("package collected = %q, want only the latest run", got)
	}
}

// A key is hex whatever its fields hold, so it is safe as a data-key: a test
// named TestParseValue/"hello" once broke the client's row lookup.
func TestKeyedRowKeyIsHex(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "kin5", "go", 10)
	m, _ := h.Merge("t", []*Buffer{in}, 10, KeySpec{Paths: []string{"fields.Test"}})
	h.publish(in, "stdout", event("p", `TestParseValue/"hello"`, "pass", ""))
	h.publish(in, "stdout", event("p", `TestParseValue/'world'`, "pass", ""))
	rows := m.Snapshot()
	for _, r := range rows {
		if k := r["key"].(string); strings.Trim(k, "0123456789abcdef") != "" || k == "" {
			t.Errorf("key %q is not hex", k)
		}
	}
	if rows[0]["key"] == rows[1]["key"] {
		t.Error("different tests share a key")
	}
}

// A drop row removes its key's row and is not stored; a later row for the key
// starts a new row at the end.
func TestKeyedDrop(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "kin6", "docker", 10)
	m, _ := h.Merge("c", []*Buffer{in}, 10, KeySpec{Paths: []string{"fields.name"}, Drop: Match{Path: "fields.state", Values: []string{"removed", "gone"}}})
	ctr := func(name, state string) Row {
		return Row{"fields": map[string]interface{}{"name": name, "state": state}}
	}
	names := func() string {
		var out []string
		for _, r := range m.Snapshot() {
			out = append(out, FieldPath(r, "fields.name").(string)+":"+FieldPath(r, "fields.state").(string))
		}
		return strings.Join(out, ",")
	}
	h.publish(in, "stdout", ctr("web", "running"), ctr("db", "running"), ctr("cache", "running"))
	h.publish(in, "stdout", ctr("web", "removed"), ctr("nope", "gone"))
	if got := names(); got != "db:running,cache:running" {
		t.Fatalf("after drop: %s", got)
	}
	h.publish(in, "stdout", ctr("web", "created"), ctr("db", "exited"))
	if got := names(); got != "db:exited,cache:running,web:created" {
		t.Fatalf("after re-add: %s", got)
	}
	// The index still points at the right rows after a removal mid-list.
	h.publish(in, "stdout", ctr("cache", "removed"), ctr("web", "running"))
	if got := names(); got != "db:exited,web:running" {
		t.Fatalf("after second drop: %s", got)
	}
}

// A merge created after the drop rows were published skips them while seeding.
func TestKeyedDropSeed(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "kin7", "docker", 10)
	h.publish(in, "stdout", Row{"fields": map[string]interface{}{"name": "a", "state": "running"}}, Row{"fields": map[string]interface{}{"name": "a", "state": "removed"}})
	m, _ := h.Merge("c", []*Buffer{in}, 10, KeySpec{Paths: []string{"fields.name"}, Drop: Match{Path: "fields.state", Values: []string{"removed"}}})
	if n := len(m.Snapshot()); n != 0 {
		t.Fatalf("rows = %d, want 0", n)
	}
}

func TestParseMatch(t *testing.T) {
	m, ok := ParseMatch("fields.state=removed|dead")
	if !ok || m.Path != "fields.state" || len(m.Values) != 2 || m.String() != "fields.state=removed|dead" {
		t.Fatalf("ParseMatch = %+v, %v", m, ok)
	}
	for _, bad := range []string{"", "fields.state", "=x", "fields.state=", "fields.state=a||b", "fields.state=a|"} {
		if _, ok := ParseMatch(bad); ok {
			t.Errorf("ParseMatch(%q) ok, want refused", bad)
		}
	}
	if (Match{}).Matches(Row{"x": "y"}) {
		t.Error("zero Match matched a row")
	}
}

// One subscription wakes on every buffer attached to it, holds each while
// attached, and releases a detached one.
func TestSubscriptionAttachDetach(t *testing.T) {
	marker := fmt.Sprintf("attachmark%d", os.Getpid())
	h := NewHub()
	defer h.Close()
	h.IdleStop = time.Hour
	running := func() bool { return exec.Command("pgrep", "-f", marker).Run() == nil }

	gate := newBuffer(h, "gate1", "gate", 10)
	in, _ := h.Input(InputSpec{Name: "w", Cmd: ": " + marker + "; while true; do echo hi; sleep 0.1; done", IdleStop: 200 * time.Millisecond})

	sub := h.NewSubscription()
	sub.Attach(gate)
	h.publish(gate, "stdout", Row{"msg": "x"})
	select {
	case <-sub.C:
	case <-time.After(2 * time.Second):
		t.Fatal("no wake-up from the attached gate")
	}
	if running() {
		t.Fatal("input runs before it is attached")
	}
	sub.Attach(in)
	sub.Attach(in) // twice: holds once
	waitFor(t, "input to start on attach", running)
	sub.Detach(in)
	waitFor(t, "input to stop after its own IdleStop, not the hub's", func() bool { return !running() })

	sub.Attach(in)
	waitFor(t, "input to restart on attach", running)
	sub.Close()
	sub.Attach(in) // after Close: nothing
	waitFor(t, "input to stop after Close", func() bool { return !running() })
}

// Equal specs that differ only in environment get separate buffers.
func TestInputEnvGetsOwnBuffer(t *testing.T) {
	h := NewHub()
	defer h.Close()
	a, _ := h.Input(InputSpec{Name: "logs", Cmd: "true", Env: map[string]string{"TD_name": "a"}})
	b, _ := h.Input(InputSpec{Name: "logs", Cmd: "true", Env: map[string]string{"TD_name": "b"}})
	a2, _ := h.Input(InputSpec{Name: "logs", Cmd: "true", Env: map[string]string{"TD_name": "a"}})
	if a == b || a != a2 {
		t.Fatalf("buffers: a=%p b=%p a again=%p; want a per value, shared for equal values", a, b, a2)
	}
}

func TestKeyedSort(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "ksort", "runs", 10)
	order, _ := ParseOrder("-fields.t")
	m, _ := h.Merge("runs", []*Buffer{in}, 10, KeySpec{Paths: []string{"fields.id"}, Sort: order})
	run := func(id string, t interface{}) Row {
		f := map[string]interface{}{"id": id}
		if t != nil {
			f["t"] = t
		}
		return Row{"fields": f}
	}
	ids := func() string { return keyedIDs(m.Snapshot()) }
	h.publish(in, "stdout", run("a", "2"), run("b", "1"), run("c", "3"))
	if got := ids(); got != "c,a,b" {
		t.Fatalf("newest first: %s", got)
	}
	h.publish(in, "stdout", run("d", "5"), run("e", "3"), run("x", nil))
	if got := ids(); got != "d,c,e,a,b,x" {
		t.Fatalf("a new row first, a tie after the earlier row, a missing value last: %s", got)
	}
	h.publish(in, "stdout", run("a", "2"), run("b", "9"))
	if got := ids(); got != "b,d,c,e,a,x" {
		t.Fatalf("an unchanged value stays, a changed one moves: %s", got)
	}
	// The index follows the moves: updating rows lands on the right ones.
	h.publish(in, "stdout", run("e", "0"), run("c", "3"))
	if got := ids(); got != "b,d,c,a,e,x" {
		t.Fatalf("after more moves: %s", got)
	}
}

func TestKeyedSortNumbersAndEviction(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "ksortn", "jobs", 10)
	order, _ := ParseOrder("fields.n")
	m, _ := h.Merge("jobs", []*Buffer{in}, 3, KeySpec{Paths: []string{"fields.id"}, Sort: order})
	job := func(id string, n float64) Row { return Row{"fields": map[string]interface{}{"id": id, "n": n}} }
	ids := func() string { return keyedIDs(m.Snapshot()) }
	h.publish(in, "stdout", job("ten", 10), job("nine", 9), job("hundred", 100))
	if got := ids(); got != "nine,ten,hundred" {
		t.Fatalf("numbers compare as numbers: %s", got)
	}
	// Full: the least recently updated key (ten, after nine and hundred are
	// refreshed) makes room, and the order holds.
	h.publish(in, "stdout", job("nine", 9), job("hundred", 100), job("one", 1))
	if got := ids(); got != "one,nine,hundred" {
		t.Fatalf("after eviction: %s", got)
	}
	h.publish(in, "stdout", job("nine", 50))
	if got := ids(); got != "one,nine,hundred" {
		t.Fatalf("a move within the same place: %s", got)
	}
	h.publish(in, "stdout", job("one", 500))
	if got := ids(); got != "nine,hundred,one" {
		t.Fatalf("a move to the end: %s", got)
	}
}

// keyedIDs lists the rows' fields.id, in order.
func keyedIDs(rows []Row) string {
	var out []string
	for _, r := range rows {
		out = append(out, FieldPath(r, "fields.id").(string))
	}
	return strings.Join(out, ",")
}

// Numeric text, as logfmt gives, sorts as numbers, the way a where-clause
// filter compares it; numbers come before other text.
func TestKeyedSortNumericText(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in := newBuffer(h, "ksortt", "jobs", 10)
	order, _ := ParseOrder("fields.ms")
	m, _ := h.Merge("jobs", []*Buffer{in}, 10, KeySpec{Paths: []string{"fields.id"}, Sort: order})
	job := func(id string, ms interface{}) Row { return Row{"fields": map[string]interface{}{"id": id, "ms": ms}} }
	h.publish(in, "stdout", job("ten", "10"), job("nine", "9"), job("slow", "n/a"), job("five", 5.0), job("fast", "fast"))
	if got := keyedIDs(m.Snapshot()); got != "five,nine,ten,fast,slow" {
		t.Fatalf("numeric text as numbers, then text: %s", got)
	}
}

func TestParseOrder(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Order
		ok   bool
	}{
		{"fields.createdAt", Order{Path: "fields.createdAt"}, true},
		{"-fields.createdAt", Order{Path: "fields.createdAt", Desc: true}, true},
		{"-", Order{}, false},
		{"", Order{}, false},
	} {
		got, ok := ParseOrder(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseOrder(%q) = %+v, %v; want %+v, %v", c.in, got, ok, c.want, c.ok)
		}
		if ok && got.String() != c.in {
			t.Errorf("Order(%q).String() = %q", c.in, got.String())
		}
	}
}
