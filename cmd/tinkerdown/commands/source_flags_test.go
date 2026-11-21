package commands

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
	"golang.org/x/term"
)

func TestParseSourceFlag(t *testing.T) {
	cases := []struct {
		in, name, cmd, file string
		wantErr             bool
	}{
		{in: "api=docker logs -f api", name: "api", cmd: "docker logs -f api"},
		{in: "q=grep a=b log.txt", name: "q", cmd: "grep a=b log.txt"},
		{in: "app=file:./logs/app.log", name: "app", file: "./logs/app.log"},
		{in: "=echo hi", wantErr: true},
		{in: "api", wantErr: true},
		{in: "api=", wantErr: true},
		{in: "app=file:", wantErr: true},
		{in: "a b=echo", wantErr: true},
		{in: "a\"=echo", wantErr: true},
		{in: strings.Repeat("x", 33) + "=echo", wantErr: true},
	}
	for _, c := range cases {
		name, src, err := parseSourceFlag(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q: want an error", c.in)
			}
			continue
		}
		if err != nil || name != c.name || src.Cmd != c.cmd || src.File != c.file || src.Type != "stream" {
			t.Errorf("%q: got name %q cmd %q file %q err %v", c.in, name, src.Cmd, src.File, err)
		}
		if back := sourceFlagValue(name, src); back != c.in {
			t.Errorf("%q: sourceFlagValue gave %q", c.in, back)
		}
	}
}

// The reuse line serve prints must run the same values when pasted back.
func TestShellQuoteRoundTrips(t *testing.T) {
	for _, v := range []string{"api=docker logs -f api", `it's "quoted" $HOME`, "a'b'c"} {
		out, err := exec.Command("sh", "-c", "printf %s "+shellQuote(v)).Output()
		if err != nil || string(out) != v {
			t.Errorf("%q came back as %q (err %v)", v, out, err)
		}
	}
}

func TestServeHelpExitsWithoutServing(t *testing.T) {
	if err := ServeCommand([]string{"--help"}); err != nil {
		t.Fatalf("serve --help: %v", err)
	}
	if err := ServeCommand([]string{"-h"}); err != nil {
		t.Fatalf("serve -h: %v", err)
	}
	// Every example in the help must be a valid --source value.
	for _, line := range strings.Split(serveUsage, "\n") {
		_, ex, ok := strings.Cut(line, "--source '")
		if !ok {
			continue
		}
		ex = strings.TrimSuffix(strings.TrimSpace(ex), "'")
		if _, _, err := parseSourceFlag(ex); err != nil {
			t.Errorf("help example %q: %v", ex, err)
		}
	}
}

// serve refuses bad --source flags before it starts listening.
func TestServeRefusesBadSourceFlags(t *testing.T) {
	defer config.SetAllowExec(false)
	dir := t.TempDir()
	page := "---\ntitle: Logs\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n    accepts:\n      kinds: [file]\n---\n\n# Logs\n"
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644)
	empty := t.TempDir()
	os.WriteFile(filepath.Join(empty, "index.md"), []byte("# Hi\n"), 0o644)

	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"command without --allow-exec", []string{dir, "--source", "api=echo hi"}, "--allow-exec"},
		{"kind the page doesn't accept", []string{dir, "--allow-exec", "--source", "api=echo hi"}, "no page takes it"},
		{"no page takes sources", []string{empty, "--source", "app=file:a.log"}, "no page takes it"},
		{"name given twice", []string{dir, "--source", "a=file:x", "--source", "a=file:y"}, "twice"},
		{"bad format", []string{dir, "--source", "noequals"}, "NAME=COMMAND"},
	}
	for _, c := range cases {
		err := ServeCommand(c.args)
		if err == nil || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: got %v, want an error containing %q", c.name, err, c.wantErr)
		}
	}
}

func TestPromptSources(t *testing.T) {
	flags := map[string]config.SourceConfig{"api": {Type: "stream", File: "a.log"}}
	in := strings.NewReader("bad\nweb=docker logs -f web\napi=file:b.log\napp=file:./logs/app.log\n\nignored=file:x\n")
	var out strings.Builder
	given := promptSources(in, &out, flags, false)

	if want := []string{"app=file:./logs/app.log"}; !slices.Equal(given, want) {
		t.Fatalf("given = %q, want %q\n%s", given, want, out.String())
	}
	for _, want := range []string{"NAME=COMMAND", "add --allow-exec", "api is given twice"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("prompt output missing %q:\n%s", want, out.String())
		}
	}
	if _, ok := flags["ignored"]; ok {
		t.Error("lines after the empty line must not be read")
	}

	// The reuse line runs the same values when pasted into a shell.
	line := reuseLine([]string{"--allow-exec", "."}, []string{"q=grep 'a b' log"})
	out2, err := exec.Command("sh", "-c", "printf '%s\\n' "+strings.TrimPrefix(line, "tinkerdown serve ")).Output()
	if err != nil || string(out2) != "--allow-exec\n.\n--source\nq=grep 'a b' log\n" {
		t.Fatalf("reuse line %q splits into %q (err %v)", line, out2, err)
	}
}

func TestDescribeNeedsCleansPageText(t *testing.T) {
	var out strings.Builder
	describeNeeds(&out, []sourceNeed{{page: "index.md", min: 2, hint: "Logs\x1b[2J\x1b]52;c;ZXZpbA==\x07 to follow " + strings.Repeat("x", 200)}})
	got := out.String()
	if strings.ContainsAny(got, "\x1b\x07") {
		t.Fatalf("escape sequences reached the terminal: %q", got)
	}
	if !strings.Contains(got, "index.md needs at least 2 sources. The page says: \"Logs[2J]52;c;ZXZpbA== to follow") || len(got) > 220 {
		t.Fatalf("unexpected message: %q", got)
	}
}

// Without a terminal (go test's stdin), serve says what to pass and starts
// serving the empty state instead of waiting for input.
func TestServeWithoutTerminalDoesNotPrompt(t *testing.T) {
	stdinIsTerminal = func() bool { return false }
	defer func() { stdinIsTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) } }()
	dir := t.TempDir()
	page := "---\ntitle: Logs\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n---\n\n# Logs\n"
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644)
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	go ServeCommand([]string{dir, "--port", strconv.Itoa(port)})
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get("http://localhost:" + strconv.Itoa(port) + "/"); err == nil {
			resp.Body.Close()
			return // serving: nothing waited on stdin
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("serve did not start listening; it may be waiting on stdin")
}

// checkFlagSources decides from resolution, as the page will see it. An
// approved ["*"] merge takes only tinkerdown.yaml sources, so a flag it would
// seem to take is unused, and refused.
func TestCheckFlagSources(t *testing.T) {
	serveDir := func(yaml, page string) (*config.Config, *server.Server) {
		dir := t.TempDir()
		if yaml != "" {
			os.WriteFile(filepath.Join(dir, "tinkerdown.yaml"), []byte(yaml), 0o644)
		}
		os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644)
		cfg, err := config.LoadFromDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		srv := server.NewWithConfig(dir, cfg)
		if err := srv.Discover(); err != nil {
			t.Fatal(err)
		}
		return cfg, srv
	}
	wildcardPage := "---\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n---\n# Logs\n"
	approvedYAML := "sources:\n  web:\n    type: stream\n    file: web.log\n  all:\n    type: merge\n    inputs: [\"*\"]\ngeneration:\n  sources: [all, web]\n"
	flag := map[string]config.SourceConfig{"api": {Type: "stream", File: "api.log"}}

	cases := []struct {
		name, yaml, page string
		flags            map[string]config.SourceConfig
		wantErr          string
	}{
		{"taken by a [\"*\"] merge", "", wildcardPage, flag, ""},
		{"replaces a page source by name", "", "---\nsources:\n  api:\n    type: stream\n    file: x.log\n---\n# A\n", flag, ""},
		{"no page takes it", "", "# Hi\n", flag, "no page takes it"},
		{"approved [\"*\"] merge takes only manifest sources", approvedYAML, "# Hi\n", flag, "no page takes it"},
		{"approved name", approvedYAML, "# Hi\n", map[string]config.SourceConfig{"web": {Type: "stream", File: "x.log"}}, "approved source"},
	}
	for _, c := range cases {
		cfg, srv := serveDir(c.yaml, c.page)
		cfg.FlagSources = c.flags
		err := checkFlagSources(cfg, srv)
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
			t.Errorf("%s: got %v, want an error containing %q", c.name, err, c.wantErr)
		}
	}
}
