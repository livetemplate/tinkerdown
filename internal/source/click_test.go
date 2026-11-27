package source

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/stream"
)

// clickSetup serves a .http file as requests and a click merge whose command
// prints what it gets, and returns the click source, subscribed.
func clickSetup(t *testing.T, httpText string) (*StreamSource, *stream.Hub, string) {
	t.Helper()
	orig := config.IsExecAllowed()
	config.SetAllowExec(true)
	t.Cleanup(func() { config.SetAllowExec(orig) })
	hub := stream.NewHub()
	t.Cleanup(func() { hub.Close() })
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api.http"), []byte(httpText), 0o644); err != nil {
		t.Fatal(err)
	}
	runs := filepath.Join(dir, "runs")
	sources := map[string]config.SourceConfig{
		"api":      {Type: "stream", HTTP: "api.http"},
		"requests": {Type: "merge", Inputs: []string{"api"}, Key: []string{"fields.entry"}},
		"send": {Type: "stream", Format: "lines",
			Cmd: `echo run >> ` + runs + `; printf '%s %s|%s|%s\n' "$TD_method" "$TD_url" "$TD_headers" "$TD_body" | tr '\n' ' '; echo`},
		"response": {Type: "merge", Inputs: []string{"send"}, On: "send", Params: map[string]config.SourceParam{"entry": {In: "requests.fields.entry"}}},
	}
	lookup := func(n string) (config.SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	page := &Page{Query: url.Values{"entry": {"api/1"}}, Resolve: lookup, InputLookup: func(string) func(string) (config.SourceConfig, bool) { return lookup }}
	reqs, err := NewStreamSource(hub, "requests", sources["requests"], dir, lookup, nil)
	if err != nil {
		t.Fatal(err)
	}
	rsub := reqs.Subscribe()
	t.Cleanup(rsub.Close)
	src, err := NewStreamSource(hub, "response", sources["response"], dir, lookup, page)
	if err != nil {
		t.Fatal(err)
	}
	sub := src.Subscribe()
	t.Cleanup(sub.Close)
	deadline := time.Now().Add(5 * time.Second)
	for len(reqs.buf.Snapshot()) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	return src, hub, runs
}

func rowsOf(t *testing.T, src *StreamSource, n int) []map[string]interface{} {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rows, _ := src.Fetch(context.Background())
		if len(rows) == n && !src.Running() {
			return rows
		}
		time.Sleep(20 * time.Millisecond)
	}
	rows, err := src.Fetch(context.Background())
	t.Fatalf("want %d rows, got %v (err %v)", n, rows, err)
	return nil
}

func TestClickRunsWithTheRowsPrivateValues(t *testing.T) {
	t.Setenv("TD_TEST_TOKEN", "s3cret")
	src, _, runs := clickSetup(t, "@host = https://api.test\n### Create\nPOST {{host}}/users\nAuthorization: Bearer {{$processEnv TD_TEST_TOKEN}}\n\n{\"a\": 1}\n")

	// The URL holds entry=api/1, but only a click starts the run.
	time.Sleep(300 * time.Millisecond)
	if rows, _ := src.Fetch(context.Background()); len(rows) != 0 {
		t.Fatalf("rows before a click: %v", rows)
	}
	if _, err := os.Stat(runs); err == nil {
		t.Fatal("the page's URL started the command")
	}

	if err := src.Click(map[string]interface{}{"entry": "api/1"}); err != nil {
		t.Fatal(err)
	}
	rows := rowsOf(t, src, 1)
	want := `POST https://api.test/users|Authorization: Bearer s3cret|{"a": 1}`
	if got := strings.TrimSpace(rows[0]["line"].(string)); got != want {
		t.Fatalf("command got %q, want %q", got, want)
	}
	if p := src.Params(); p["entry"] != "api/1" || len(p) != 1 {
		t.Fatalf("Params() = %v; private values must not be there", p)
	}
}

func TestClickRefusals(t *testing.T) {
	src, _, runs := clickSetup(t, "### a\nGET https://api.test/{{nope_unset_var}}\n### b\nGET https://api.test/b\n")
	var ve *ValidationError
	for name, values := range map[string]map[string]interface{}{
		"not listed":     {"entry": "api/99"},
		"missing":        {},
		"option":         {"entry": "-1"},
		"unset variable": {"entry": "api/1"},
	} {
		err := src.Click(values)
		if !errors.As(err, &ve) {
			t.Errorf("%s: err = %v, want a ValidationError", name, err)
		}
		if _, ferr := src.Fetch(context.Background()); ferr == nil {
			t.Errorf("%s: Fetch didn't report the refusal", name)
		}
	}
	if _, err := os.Stat(runs); err == nil {
		t.Fatal("a refused click ran the command")
	}
	if err := src.Click(map[string]interface{}{"entry": "api/2"}); err != nil {
		t.Fatal(err)
	}
	rowsOf(t, src, 1)
	if _, err := src.Fetch(context.Background()); err != nil {
		t.Fatalf("a good click left the old refusal: %v", err)
	}
}

func TestClickValueSizeLimit(t *testing.T) {
	// The whole TD_body=… string with its NUL may be MaxClickValue bytes.
	fits := strings.Repeat("x", MaxClickValue-len("TD_body=")-1)
	src, _, _ := clickSetup(t, "### a\nPOST https://api.test/\n\n"+fits+"\n### b\nPOST https://api.test/\n\n"+fits+"y\n")
	if err := src.Click(map[string]interface{}{"entry": "api/1"}); err != nil {
		t.Fatalf("a body at the limit was refused: %v", err)
	}
	// One stdout row means the command ran: an E2BIG would be a system row.
	if r := rowsOf(t, src, 1)[0]; r["stream"] != "stdout" || !strings.Contains(r["line"].(string), "xxxx") {
		t.Fatalf("at the limit, the command didn't run: %v", r["line"])
	}
	if err := src.Click(map[string]interface{}{"entry": "api/2"}); err == nil || !strings.Contains(err.Error(), "over 128 KiB") {
		t.Fatalf("a body one byte over: err = %v", err)
	}
}

func TestClicksInALoopRunOncePerGap(t *testing.T) {
	src, _, runs := clickSetup(t, "### a\nPOST https://api.test/a\n")
	count := func() int { data, _ := os.ReadFile(runs); return strings.Count(string(data), "run") }
	start := time.Now()
	for range 20 {
		if err := src.Click(map[string]interface{}{"entry": "api/1"}); err != nil {
			t.Fatal(err)
		}
	}
	// A stalled machine can stretch the loop past a gap, which allows one
	// more run per gap.
	most := 2 + int(time.Since(start)/ClickGap)
	// The first click runs now, and the last of the rest when the gap ends.
	deadline := time.Now().Add(5 * time.Second)
	for count() < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(3 * ClickGap)
	if n := count(); n < 2 || n > most {
		t.Fatalf("20 clicks in a row ran it %d times, want 2 to %d", n, most)
	}
}

func TestAClickAfterCloseRunsNothing(t *testing.T) {
	src, _, runs := clickSetup(t, "### a\nPOST https://api.test/a\n")
	src.Close()
	if err := src.Click(map[string]interface{}{"entry": "api/1"}); err != nil {
		t.Fatalf("a click after Close: %v, want nil", err)
	}
	time.Sleep(3 * ClickGap)
	if data, _ := os.ReadFile(runs); len(data) != 0 {
		t.Fatalf("a click after Close ran: %q", data)
	}
}

func TestClosingDropsAWaitingClick(t *testing.T) {
	src, _, runs := clickSetup(t, "### a\nPOST https://api.test/a\n")
	src.Click(map[string]interface{}{"entry": "api/1"})
	src.Click(map[string]interface{}{"entry": "api/1"}) // waits for the gap
	src.Close()
	time.Sleep(3 * ClickGap)
	if data, _ := os.ReadFile(runs); strings.Count(string(data), "run") != 1 {
		t.Fatalf("runs = %q, want the first click's only", data)
	}
}

func TestClickMergeIgnoresTheQuery(t *testing.T) {
	src, _, _ := clickSetup(t, "### a\nGET https://api.test/a\n")
	if src.SetQuery(url.Values{"entry": {"api/1"}}) {
		t.Fatal("a click merge took values from the URL")
	}
}

// A typed param takes the click's value, such as a form field, under the
// value rules alone: there is no merge it must be listed in.
func TestTypedParamReachesTheCommand(t *testing.T) {
	orig := config.IsExecAllowed()
	config.SetAllowExec(true)
	t.Cleanup(func() { config.SetAllowExec(orig) })
	hub := stream.NewHub()
	t.Cleanup(func() { hub.Close() })
	dir := t.TempDir()
	runs := filepath.Join(dir, "runs")
	sources := map[string]config.SourceConfig{
		"commit":    {Type: "stream", Format: "lines", Cmd: `echo run >> ` + runs + `; printf 'committed %s\n' "$TD_message"`},
		"committed": {Type: "merge", Inputs: []string{"commit"}, On: "commit", Params: map[string]config.SourceParam{"message": {Typed: true}}},
	}
	lookup := func(n string) (config.SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	page := &Page{Resolve: lookup, InputLookup: func(string) func(string) (config.SourceConfig, bool) { return lookup }}
	src, err := NewStreamSource(hub, "committed", sources["committed"], dir, lookup, page)
	if err != nil {
		t.Fatal(err)
	}
	sub := src.Subscribe()
	t.Cleanup(sub.Close)

	for name, v := range map[string]interface{}{
		"empty":      "",
		"two lines":  "fix\nmore",
		"too long":   strings.Repeat("a", 257),
		"an option":  "--amend",
		"no message": nil,
	} {
		if err := src.Click(map[string]interface{}{"message": v}); err == nil {
			t.Errorf("%s: the click was accepted", name)
		}
	}
	if _, err := os.Stat(runs); err == nil {
		t.Fatal("a refused message ran the command")
	}
	if err := src.Click(map[string]interface{}{"message": `fix "the" typo; $(id)`}); err != nil {
		t.Fatal(err)
	}
	rows := rowsOf(t, src, 1)
	if got := rows[0]["line"]; got != `committed fix "the" typo; $(id)` {
		t.Fatalf("the command got %q", got)
	}
}
