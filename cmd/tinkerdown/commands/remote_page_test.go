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
		{in: "https://example.com/i.md#sha256=" + strings.Repeat("AB", 32), url: "https://example.com/i.md", pin: strings.Repeat("ab", 32)},
		{in: "https://example.com/i.md#top", wantErr: "#sha256=HEX"},
		{in: "https://example.com/i.md#sha256=xyz", wantErr: "64 hex characters"},
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

func TestParseShortName(t *testing.T) {
	const raw = "https://raw.githubusercontent.com/"
	pin := strings.Repeat("ab", 32)
	cases := []struct {
		in, url, pin, wantErr string
	}{
		{in: "github.com/o/r/app", url: raw + "o/r/HEAD/app/index.md"},
		{in: "github.com/o/r/deep/app@v1", url: raw + "o/r/v1/deep/app/index.md"},
		{in: "github.com/o/r/app@3f2a9c1", url: raw + "o/r/3f2a9c1/app/index.md"},
		{in: "github.com/o/r/app@main", url: raw + "o/r/main/app/index.md"},
		{in: "github.com/o/r/doc.md", url: raw + "o/r/HEAD/doc.md"},
		{in: "github.com/o/my.repo", url: raw + "o/my.repo/HEAD/index.md"},
		{in: "github.com/o/r/app@v1#sha256=" + strings.ToUpper(pin), url: raw + "o/r/v1/app/index.md", pin: pin},
		{in: "github.com/o", wantErr: "github.com/OWNER/REPO"},
		{in: "github.com/o/", wantErr: "each part after OWNER"},
		{in: "github.com/o/r/../x", wantErr: "each part after OWNER"},
		{in: "github.com/o/r/a?b", wantErr: "each part after OWNER"},
		{in: "github.com/o/r/a%2e", wantErr: "each part after OWNER"},
		{in: "github.com/o/r/a b", wantErr: "each part after OWNER"},
		{in: "github.com/-o/r", wantErr: "OWNER"},
		{in: "github.com/o/r/app@", wantErr: "@REF"},
		{in: "github.com/o/r/app@..x", wantErr: "@REF"},
		{in: "github.com/o/r/app@-x", wantErr: "@REF"},
		{in: "github.com/o/r/app#top", wantErr: "#sha256=HEX"},
	}
	for _, c := range cases {
		got, err := parsePageArg(c.in)
		if c.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("%s: got %v, want an error containing %q", c.in, err, c.wantErr)
			}
			continue
		}
		if err != nil || got.url != c.url || got.pin != c.pin {
			t.Errorf("%s: got %+v (err %v), want url %s", c.in, got, err, c.url)
			continue
		}
		name, _, _ := strings.Cut(c.in, "#")
		if got.share != name || got.shown != name+" ("+c.url+")" {
			t.Errorf("%s: share %q, shown %q: both must name the short name, and shown the URL too", c.in, got.share, got.shown)
		}
	}
}

// A GOPATH-style checkout is a real directory named github.com/..., and
// serving it must keep working.
func TestShortNameLocalPathWins(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("github.com/o/r/app", 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := parsePageArg("github.com/o/r/app")
	if err != nil || got.url != "" || got.dir != "github.com/o/r/app" {
		t.Fatalf("got %+v (err %v), want the local directory", got, err)
	}
}

func TestShortNameFetchErrorNamesBoth(t *testing.T) {
	p, err := parsePageArg("github.com/o/r/app")
	if err != nil {
		t.Fatal(err)
	}
	p.url = "http://127.0.0.1:1/index.md" // nothing listens: the fetch fails offline
	_, _, err = fetchRemotePage(p)
	if err == nil || !strings.Contains(err.Error(), "github.com/o/r/app (https://raw.githubusercontent.com/o/r/HEAD/app/index.md)") {
		t.Fatalf("fetch error %v must name the short name and the URL", err)
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
	mux.HandleFunc("/loop", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
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
	if pinned, _, err := fetchRemotePage(pageArg{url: ts.URL + "/page.md", pin: sum}); err != nil {
		t.Errorf("matching pin refused: %v", err)
	} else {
		os.RemoveAll(pinned)
	}

	for name, p := range map[string]pageArg{
		"pin mismatch":                  {url: ts.URL + "/page.md", pin: strings.Repeat("0", 64)},
		"over 1 MiB":                    {url: ts.URL + "/huge.md"},
		"redirect to http":              {url: ts.URL + "/to-http"},
		"redirect to a private address": {url: ts.URL + "/to-private"},
		"more than 3 redirects":         {url: ts.URL + "/loop"},
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
	if err := checkRemotePage(parse(ok)); err != nil {
		t.Fatalf("a merge-only page with accepts: %v", err)
	}
	withParams := "---\ntitle: Containers\nsources:\n  containers:\n    type: merge\n    inputs: [\"*\"]\n    key: [fields.name]\n    drop: fields.state=removed\n  logs:\n    type: merge\n    inputs: [logs]\n    params:\n      name: {in: containers.fields.name}\n---\n\n```lvt\n<ol lvt-source=\"logs\">{{range .Data}}<li data-key=\"{{.seq}}\">{{.msg}}</li>{{end}}</ol>\n```\n"
	if err := checkRemotePage(parse(withParams)); err != nil {
		t.Fatalf("a merge-only page with drop and params: %v", err)
	}
	cases := map[string]string{
		"a stream source":                 "---\nsources:\n  x:\n    type: stream\n    cmd: id\n---\n# X\n",
		"a json source":                   "---\nsources:\n  x:\n    type: json\n    file: ~/.aws/credentials\n---\n# X\n",
		"an action":                       "---\nactions:\n  wipe:\n    kind: sql\n    statement: DELETE FROM t\n---\n# X\n",
		"a big buffer":                    "---\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n    buffer: 100000\n---\n# X\n",
		"a script":                        "# X\n\n<script>fetch('https://evil.example')</script>\n",
		"an event handler":                "# X\n\n<img src=x onerror=\"alert(1)\">\n",
		"a javascript URL":                "# X\n\n<a href=\"javascript:alert(1)\">x</a>\n",
		"a meta refresh":                  "# X\n\n<meta http-equiv=\"refresh\" content=\"0;url=https://evil.example\">\n",
		"a description":                   "---\ndescription: x\n---\n# X\n",
		"a source repo":                   "---\nsource_repo: https://github.com/a/b\n---\n# X\n",
		"a template action in a tag name": "---\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n---\n\n```lvt\n<p lvt-source=\"all\"><{{\"meta\"}} http-equiv=\"refresh\"></p>\n```\n",
		"a template action in a raw-text element": "---\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n---\n\n```lvt\n<div lvt-source=\"all\"><textarea>{{range .Data}}{{.msg}}{{end}}</textarea></div>\n```\n",
	}
	for name, src := range cases {
		if err := checkRemotePage(parse(src)); err == nil {
			t.Errorf("%s: want a refusal", name)
		} else if !strings.Contains(err.Error(), "tinkerdown serve page.md") {
			t.Errorf("%s: the refusal should say how to run a reviewed copy: %v", name, err)
		}
	}
}

func TestRemoteMessagesHideCredentials(t *testing.T) {
	p, err := parsePageArg("https://user:s3cret@example.invalid/page.md#sha256=" + strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.shown, "s3cret") || !strings.Contains(p.url, "s3cret") {
		t.Fatalf("shown %q, url %q", p.shown, p.url)
	}
	_, _, err = fetchRemotePage(p)
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("fetch error %v must not carry the password", err)
	}
	if _, err := parsePageArg("http://user:s3cret@example.com/p.md"); err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("refusal %v must not carry the password", err)
	}
}
