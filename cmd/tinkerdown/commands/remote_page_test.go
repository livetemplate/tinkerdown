package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/livetemplate/tinkerdown"
)

func TestParsePageArg(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "logs.md")
	os.WriteFile(page, []byte("# Logs\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)

	cases := []struct {
		in                        string
		wantDir, single, url, pin string
		wantErr                   string
	}{
		{in: dir, wantDir: dir},
		{in: page, wantDir: dir, single: "logs.md"},
		{in: "file://" + page, wantDir: dir, single: "logs.md"},
		{in: "file://localhost" + page, wantDir: dir, single: "logs.md"},
		{in: "file://evil.example" + page, wantErr: "must name this machine"},
		{in: filepath.Join(dir, "notes.txt"), wantErr: "directory or a .md file"},
		{in: "https://example.com/a/index.md", url: "https://example.com/a/index.md"},
		{in: "https://example.com/i.md#sha256=ABC", url: "https://example.com/i.md", pin: "abc"},
		{in: "https://example.com/i.md#top", wantErr: "#sha256=HEX"},
		{in: "http://localhost:9000/i.md", url: "http://localhost:9000/i.md"},
		{in: "http://example.com/i.md", wantErr: "use https"},
	}
	for _, c := range cases {
		got, err := parsePageArg(c.in)
		if c.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("%s: got %v, want an error containing %q", c.in, err, c.wantErr)
			}
			continue
		}
		if err != nil || got.dir != c.wantDir || got.single != c.single || got.url != c.url || got.pin != c.pin {
			t.Errorf("%s: got %+v (err %v)", c.in, got, err)
		}
	}
}

func TestFetchRemotePage(t *testing.T) {
	const body = "---\ntitle: Logs\n---\n# Logs\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/page.md", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
	mux.HandleFunc("/huge.md", func(w http.ResponseWriter, r *http.Request) { w.Write(make([]byte, maxRemotePage+1)) })
	mux.HandleFunc("/to-http", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/page.md", http.StatusFound)
	})
	mux.HandleFunc("/to-private", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://169.254.169.254/latest/meta-data", http.StatusFound)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	dir, sum, err := fetchRemotePage(pageArg{url: ts.URL + "/page.md"})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	got, _ := os.ReadFile(filepath.Join(dir, "page.md"))
	info, _ := os.Stat(dir)
	if string(got) != body || len(sum) != 64 || info.Mode().Perm() != 0o700 {
		t.Fatalf("saved %q, sum %q, dir mode %v", got, sum, info.Mode().Perm())
	}
	if _, _, err := fetchRemotePage(pageArg{url: ts.URL + "/page.md", pin: sum}); err != nil {
		t.Errorf("matching pin refused: %v", err)
	}

	for name, p := range map[string]pageArg{
		"pin mismatch":                  {url: ts.URL + "/page.md", pin: strings.Repeat("0", 64)},
		"over 1 MiB":                    {url: ts.URL + "/huge.md"},
		"redirect to http":              {url: ts.URL + "/to-http"},
		"redirect to a private address": {url: ts.URL + "/to-private"},
		"not found":                     {url: ts.URL + "/missing.md"},
	} {
		if dir, _, err := fetchRemotePage(p); err == nil {
			os.RemoveAll(dir)
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestCheckRemotePage(t *testing.T) {
	parse := func(src string) *tinkerdown.Page {
		t.Helper()
		f := filepath.Join(t.TempDir(), "page.md")
		os.WriteFile(f, []byte(src), 0o644)
		p, err := tinkerdown.ParseFile(f)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	ok := "---\ntitle: Logs\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n    accepts: {kinds: [file], min: 1, hint: logs}\n---\n\n## [All] | [*] source\n\n```lvt\n<ol lvt-source=\"all\">{{range .Data}}<li data-key=\"{{.seq}}\">{{.msg}}</li>{{end}}</ol>\n```\n"
	if err := checkRemotePage(parse(ok), ok); err != nil {
		t.Fatalf("a merge-only page with accepts: %v", err)
	}
	cases := map[string]string{
		"a stream source":  "---\nsources:\n  x:\n    type: stream\n    cmd: id\n---\n# X\n",
		"a json source":    "---\nsources:\n  x:\n    type: json\n    file: ~/.aws/credentials\n---\n# X\n",
		"an action":        "---\nactions:\n  wipe:\n    kind: sql\n    statement: DELETE FROM t\n---\n# X\n",
		"a big buffer":     "---\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n    buffer: 100000\n---\n# X\n",
		"a script":         "# X\n\n<script>fetch('https://evil.example')</script>\n",
		"an event handler": "# X\n\n<img src=x onerror=\"alert(1)\">\n",
		"a javascript URL": "# X\n\n<a href=\"javascript:alert(1)\">x</a>\n",
		"a meta refresh":   "# X\n\n<meta http-equiv=\"refresh\" content=\"0;url=https://evil.example\">\n",
	}
	for name, src := range cases {
		if err := checkRemotePage(parse(src), src); err == nil {
			t.Errorf("%s: want a refusal", name)
		} else if !strings.Contains(err.Error(), "tinkerdown serve page.md") {
			t.Errorf("%s: the refusal should say how to run a reviewed copy: %v", name, err)
		}
	}
}
