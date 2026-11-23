package stream

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestParseRequestLine(t *testing.T) {
	for line, want := range map[string][3]string{
		"GET https://api.test/users":           {"GET", "https://api.test/users", "ok"},
		"POST {{host}}/users HTTP/1.1":         {"POST", "{{host}}/users", "ok"},
		"delete https://api.test/users/1":      {"DELETE", "https://api.test/users/1", "ok"},
		"https://api.test/health":              {"GET", "https://api.test/health", "ok"},
		"http://localhost:8080/ HTTP/2":        {"GET", "http://localhost:8080/", "ok"},
		"{{host}}/users":                       {"GET", "{{host}}/users", "ok"},
		"OPTIONS * HTTP/1.1":                   {"OPTIONS", "*", "ok"},
		"Content-Type: application/json":       {"", "", ""},
		"hello world":                          {"", "", ""},
		"GET":                                  {"", "", ""},
		"FETCH https://api.test/":              {"", "", ""},
		"GET https://api.test/ HTTP/1.1 extra": {"", "", ""},
	} {
		m, u, ok := parseRequestLine(line)
		got := [3]string{m, u, ""}
		if ok {
			got[2] = "ok"
		}
		if got != want {
			t.Errorf("parseRequestLine(%q) = %q, want %q", line, got, want)
		}
	}
}

const sampleHTTP = `@host = https://api.test
# a comment
// another

### List users
GET {{host}}/users
Accept: application/json

### Create user
POST {{host}}/users HTTP/1.1
Content-Type: application/json
# a comment between headers
Authorization: Bearer {{$processEnv token}}

{
  "name": "Ada"
}

###
# @name health
{{host}}/health
`

func TestParseHTTPFile(t *testing.T) {
	reqs, vars, err := ParseHTTPFile(sampleHTTP)
	if err != nil {
		t.Fatal(err)
	}
	if vars["host"] != "https://api.test" {
		t.Fatalf("vars = %v", vars)
	}
	want := []HTTPRequest{
		{Name: "List users", Method: "GET", URL: "{{host}}/users", Headers: "Accept: application/json"},
		{Name: "Create user", Method: "POST", URL: "{{host}}/users", Headers: "Content-Type: application/json\nAuthorization: Bearer {{$processEnv token}}", Body: "{\n  \"name\": \"Ada\"\n}"},
		{Name: "health", Method: "GET", URL: "{{host}}/health"},
	}
	if !reflect.DeepEqual(reqs, want) {
		t.Fatalf("requests:\n%#v\nwant:\n%#v", reqs, want)
	}
}

func TestParseHTTPFileErrorNamesTheLine(t *testing.T) {
	_, _, err := ParseHTTPFile("### a\nGET https://x\nnot a header\n")
	if err == nil || !strings.HasPrefix(err.Error(), "line 3:") {
		t.Fatalf("err = %v, want one naming line 3", err)
	}
	_, _, err = ParseHTTPFile("hello there\n")
	if err == nil || !strings.HasPrefix(err.Error(), "line 1:") {
		t.Fatalf("err = %v, want one naming line 1", err)
	}
}

func TestFillReadsTheEnvironmentOnlyByName(t *testing.T) {
	t.Setenv("TD_TEST_TOKEN", "s3cret")
	t.Setenv("TD_TEST_RAW", "{{host}}")
	got, err := Fill("{{host}}/x?t={{ $processEnv TD_TEST_TOKEN }}", map[string]string{"host": "https://h"})
	if got != "https://h/x?t=s3cret" || err != nil {
		t.Fatalf("Fill = %q, %v", got, err)
	}
	for in, want := range map[string]string{
		"{{TD_TEST_TOKEN}}":              "unset variable TD_TEST_TOKEN", // never the environment
		"{{$processEnv nope_unset_var}}": "unset environment variable nope_unset_var",
	} {
		if _, err := Fill(in, nil); err == nil || err.Error() != want {
			t.Errorf("Fill(%s) err = %v, want %s", in, err, want)
		}
	}
	// An environment value is used as it is, never filled.
	if got, _ := Fill("{{$processEnv TD_TEST_RAW}}", map[string]string{"host": "h"}); got != "{{host}}" {
		t.Fatalf("an environment value was filled: %q", got)
	}
}

func TestFillFillsVariableValues(t *testing.T) {
	t.Setenv("TD_TEST_TOKEN", "s3cret")
	vars := map[string]string{"token": "{{$processEnv TD_TEST_TOKEN}}", "auth": "Bearer {{token}}"}
	if got, err := Fill("Authorization: {{auth}}", vars); got != "Authorization: Bearer s3cret" || err != nil {
		t.Fatalf("Fill = %q, %v", got, err)
	}
	for _, cyc := range []map[string]string{{"a": "{{a}}"}, {"a": "{{b}}", "b": "{{a}}"}} {
		if _, err := Fill("{{a}}", cyc); err == nil || !strings.Contains(err.Error(), "refers to itself") {
			t.Errorf("cycle %v: err = %v", cyc, err)
		}
	}
	if _, err := Fill("{{a}}", map[string]string{"a": "{{gone}}"}); err == nil || err.Error() != "unset variable gone" {
		t.Errorf("an unset variable inside a value: err = %v", err)
	}
	deep := map[string]string{"v0": "x"}
	for i := 1; i <= 11; i++ {
		deep[fmt.Sprintf("v%d", i)] = fmt.Sprintf("{{v%d}}", i-1)
	}
	if _, err := Fill("{{v11}}", deep); err == nil || !strings.Contains(err.Error(), "more than 10 deep") {
		t.Errorf("a chain 11 deep: err = %v", err)
	}
}

func TestFillBoundsFanOut(t *testing.T) {
	// Ten references per level, nine levels deep: 10^9 copies of v0. With
	// "x" that's too many bytes; with "" it costs no bytes, only work.
	for _, v0 := range []string{"x", ""} {
		vars := map[string]string{"v0": v0}
		for i := 1; i <= 9; i++ {
			vars[fmt.Sprintf("v%d", i)] = strings.Repeat(fmt.Sprintf("{{v%d}}", i-1), 10)
		}
		start := time.Now()
		_, err := Fill("{{v9}}", vars)
		if v0 != "" && (err == nil || !strings.Contains(err.Error(), "over 64 MiB")) {
			t.Errorf("v0 %q: err = %v", v0, err)
		}
		if v0 == "" && err != nil {
			t.Errorf("v0 %q: err = %v", v0, err)
		}
		if d := time.Since(start); v0 == "" && d > 2*time.Second {
			t.Errorf("empty fan-out took %v: each variable must fill once", d)
		}
	}
	// The budget is the whole file's: many requests using one 1 MiB value
	// run out, and the requests after that say so.
	reqs := slices.Repeat([]HTTPRequest{{Method: "POST", URL: "https://h/", Body: "{{big}}"}}, 80)
	vf, failed := newFiller(map[string]string{"big": strings.Repeat("y", 1<<20)}), 0
	for _, q := range reqs {
		if p := fillRequest(q, vf); strings.Contains(p.Err, "over 64 MiB") {
			failed++
		}
	}
	if failed == 0 || failed == len(reqs) {
		t.Errorf("%d of %d requests ran out of budget, want some", failed, len(reqs))
	}
}

func TestVariablesAreFileWide(t *testing.T) {
	// A variable defined after the request that uses it still fills it, and
	// one after a request line, before the next ###, is body text.
	reqs, vars, err := ParseHTTPFile("### a\nGET {{host}}/a\n\n@notavar = 1\n\n###\n@host = https://h\n")
	if err != nil {
		t.Fatal(err)
	}
	if reqs[0].Body != "@notavar = 1" || vars["notavar"] != "" {
		t.Fatalf("body = %q, vars = %v", reqs[0].Body, vars)
	}
	if p := fillRequest(reqs[0], newFiller(vars)); p.Values["url"] != "https://h/a" {
		t.Fatalf("private = %v", p)
	}
}

func TestHTTPReaderFollowsTheFile(t *testing.T) {
	t.Setenv("token", "abc")
	path := filepath.Join(t.TempDir(), "api.http")
	if err := os.WriteFile(path, []byte(sampleHTTP), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewHub()
	defer h.Close()
	in, err := h.Input(InputSpec{Name: "api", HTTP: path})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := h.Merge("requests", []*Buffer{in}, 0, KeySpec{Paths: []string{"fields.entry"}})
	sub := m.Subscribe()
	defer sub.Close()
	waitFor(t, "three requests", func() bool { return len(m.Snapshot()) == 3 })

	rows := m.Snapshot()
	f := rows[1]["fields"].(map[string]interface{})
	if f["entry"] != "2" || f["name"] != "Create user" || f["url"] != "{{host}}/users" {
		t.Fatalf("row 2 fields = %v", f)
	}
	p, _ := h.Private(rows[1])
	if p.Values["url"] != "https://api.test/users" || !strings.Contains(p.Values["headers"], "Bearer abc") {
		t.Fatalf("row 2 private = %v", p)
	}
	for _, r := range rows {
		if strings.Contains(strings.ReplaceAll(fmtRow(r), "{{$processEnv token}}", ""), "abc") {
			t.Fatalf("a filled-in token is in a row: %v", r)
		}
	}

	// A bad edit keeps the rows and reports the line; fixing it clears it.
	time.Sleep(10 * time.Millisecond) // a new modification time
	os.WriteFile(path, []byte(sampleHTTP+"\n### broken\nGET https://x\nnot a header\n"), 0o644)
	waitFor(t, "the parse error", func() bool { return strings.Contains(m.Err(), "line 25:") })
	if len(m.Snapshot()) != 3 {
		t.Fatal("a parse error dropped the last good rows")
	}

	// Deleting a request removes its row.
	os.WriteFile(path, []byte("### only\nGET https://api.test/only\n"), 0o644)
	waitFor(t, "one request", func() bool { return len(m.Snapshot()) == 1 && m.Err() == "" })
}

func TestHTTPRequestWithUnsetVariableHasNoValues(t *testing.T) {
	reqs, vars, _ := ParseHTTPFile("GET https://x/{{nope_unset_var}}\n")
	if p := fillRequest(reqs[0], newFiller(vars)); p.Values != nil || p.Err != "unset variable nope_unset_var" {
		t.Fatalf("private = %v", p)
	}
}

func fmtRow(r Row) string {
	var b strings.Builder
	for k, v := range r {
		b.WriteString(k + "=")
		if m, ok := v.(map[string]interface{}); ok {
			for fk, fv := range m {
				b.WriteString(fk + ":" + strings.TrimSpace(strings.ReplaceAll(toString(fv), "\n", " ")) + ";")
			}
		} else {
			b.WriteString(toString(v))
		}
		b.WriteString(" ")
	}
	return b.String()
}

func toString(v interface{}) string {
	s, _ := v.(string)
	return s
}
