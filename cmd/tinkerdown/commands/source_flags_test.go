package commands

import (
	"maps"
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
		{in: "docker logs -f api", cmd: "docker logs -f api"},
		{in: "grep a=b log.txt", cmd: "grep a=b log.txt"},
		{in: "env FOO=1 cmd", cmd: "env FOO=1 cmd"},
		{in: "file:./logs/app.log", file: "./logs/app.log"},
		{in: "a b=echo", cmd: "a b=echo"},
		{in: strings.Repeat("x", 33) + "=echo", cmd: strings.Repeat("x", 33) + "=echo"},
		{in: "=echo hi", wantErr: true},
		{in: "api=", wantErr: true},
		{in: "app=file:", wantErr: true},
		{in: "", wantErr: true},
		{in: "  ", wantErr: true},
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
		if back := sourceFlagValue(name, src); name != "" && back != c.in {
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
		{"empty value", []string{dir, "--source", ""}, "want COMMAND"},
		{"name missing before =", []string{dir, "--source", "=echo hi"}, "a name goes before"},
		{"unknown flag", []string{dir, "--verbose"}, "unknown flag --verbose"},
		{"unknown short flag", []string{dir, "-x"}, "unknown flag -x"},
		{"FOO=1 cmd names a source", []string{dir, "--source", "FOO=1 cmd"}, "--source FOO runs a command"},
		{"nothing after --", []string{dir, "--allow-exec", "--"}, "needs a command"},
		{"--port with no value", []string{dir, "--port"}, "--port needs a value"},
		{"--config with no value", []string{dir, "-c"}, "-c needs a value"},
		{"--port=N", []string{dir, "--port=8080"}, "give the value after a space: --port 8080"},
		// Port 1 is closed: the flag is refused before any fetch is tried.
		{"bad flag before the fetch", []string{"http://127.0.0.1:1/page.md", "--source", "sh x"}, "add --allow-exec"},
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
	in := strings.NewReader("=bad\nweb=docker logs -f web\napi=file:b.log\napp=file:./logs/app.log\n\nignored=file:x\n")
	var out strings.Builder
	given := promptSources(in, &out, flags, nil, false)

	if want := []string{"app=file:./logs/app.log"}; !slices.Equal(given, want) {
		t.Fatalf("given = %q, want %q\n%s", given, want, out.String())
	}
	for _, want := range []string{"a name goes before", "add --allow-exec", "api is given twice"} {
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

func TestAddFlagAction(t *testing.T) {
	flags := map[string]*config.Action{}
	if err := addFlagAction(flags, `stop=docker stop -- "$TD_name" --signal=TERM`, true); err != nil {
		t.Fatal(err)
	}
	if a := flags["stop"]; a == nil || a.Cmd != `docker stop -- "$TD_name" --signal=TERM` || a.Kind != "exec" || !a.FromFlag {
		t.Fatalf("stop = %+v", a)
	}
	for _, c := range []struct {
		v     string
		allow bool
	}{
		{"stop=docker stop x", true},      // given twice
		{"restart=docker restart", false}, // no --allow-exec
		{"pause=docker pause x", true},    // builtin
		{"Delete=rm x", true},             // builtin, any case
		{"bad name=x", true},
		{"noequals", true},
		{"empty=  ", true},
	} {
		if err := addFlagAction(flags, c.v, c.allow); err == nil {
			t.Errorf("addFlagAction(%q, %v) = nil, want an error", c.v, c.allow)
		}
	}
}

func TestCheckFlagActionsRefusesApproved(t *testing.T) {
	cfg := &config.Config{
		Actions:     map[string]*config.Action{"stop": {Kind: "exec", Cmd: "true"}},
		Generation:  &config.GenerationConfig{Actions: []string{"stop"}},
		FlagActions: map[string]*config.Action{"stop": {Kind: "exec", Cmd: "x", FromFlag: true}},
	}
	if err := checkFlagActions(cfg); err == nil {
		t.Fatal("a flag replaced an approved action")
	}
	cfg.FlagActions = map[string]*config.Action{"kill": {Kind: "exec", Cmd: "x", FromFlag: true}}
	if err := checkFlagActions(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPSourceFlagAndAbsPaths(t *testing.T) {
	name, src, err := parseSourceFlag("api=http:./api.http")
	if err != nil || name != "api" || src.HTTP != "./api.http" || src.Kind() != "http" {
		t.Fatalf("parseSourceFlag = %q, %+v, %v", name, src, err)
	}
	if got := sourceFlagValue(name, src); got != "api=http:./api.http" {
		t.Fatalf("sourceFlagValue = %q", got)
	}
	if _, _, err := parseSourceFlag("api=http:"); err == nil {
		t.Fatal("http: without a path was accepted")
	}
	wd, _ := os.Getwd()
	flags := map[string]config.SourceConfig{"api": src, "log": {Type: "stream", File: "/var/log/x"}, "cmd": {Type: "stream", Cmd: "echo ./x"}}
	actions := map[string]*config.Action{"stop": {Kind: "exec", Cmd: "docker stop x"}, "save": {Kind: "sql"}}
	rebase(flags, actions, wd)
	if flags["api"].HTTP != filepath.Join(wd, "api.http") || flags["log"].File != "/var/log/x" || flags["cmd"].Cmd != "echo ./x" {
		t.Fatalf("rebase = %+v", flags)
	}
	// Commands run in base, since a fetched page's directory holds only the page.
	if flags["cmd"].Dir != wd || flags["log"].Dir != "" || actions["stop"].Dir != wd || actions["save"].Dir != "" {
		t.Fatalf("rebase dirs: sources %+v, actions stop=%q save=%q", flags, actions["stop"].Dir, actions["save"].Dir)
	}
}

func TestRemoteConfigPathsFollowTheConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "http.yaml")
	os.WriteFile(path, []byte("sources:\n  api: {type: stream, http: ./api.http}\n  abs: {type: stream, file: /var/log/x}\n  run: {type: stream, cmd: ./run.sh}\n"+
		"actions:\n  stop: {kind: exec, cmd: ./stop.sh}\n"), 0o644)
	cfg, err := remoteConfig(path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Sources["api"].HTTP; got != filepath.Join(dir, "api.http") {
		t.Fatalf("http path = %q, want it next to the config", got)
	}
	if got := cfg.Sources["abs"].File; got != "/var/log/x" {
		t.Fatalf("absolute path changed to %q", got)
	}
	if cfg.Sources["run"].Dir != dir || cfg.Actions["stop"].Dir != dir {
		t.Fatalf("commands run in %q and %q, want the config's directory %q", cfg.Sources["run"].Dir, cfg.Actions["stop"].Dir, dir)
	}
}

// Named sources keep their names, and unnamed ones get defaultLabel's
// label, made unique. A named source goes in first even when it comes
// later, so "app" stays the operator's.
func TestFlagSourcesLabels(t *testing.T) {
	const tail = "tail -f /var/log/app.log"
	flags, err := flagSources([]string{tail, "app=file:a.log", tail}, "watchexec -e go -- go test -json ./...", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := slices.Sorted(maps.Keys(flags)), []string{"app", "app-2", "app-3", "test"}; !slices.Equal(got, want) {
		t.Fatalf("sources %q, want %q", got, want)
	}
	if flags["app"].File != "a.log" || flags["app-2"].Cmd != tail || flags["app-3"].Cmd != tail {
		t.Errorf("got %+v", flags)
	}
	if _, err := flagSources([]string{tail}, "", nil, false); err == nil || !strings.Contains(err.Error(), "--allow-exec") {
		t.Errorf("an unnamed command without --allow-exec: got %v", err)
	}
}

// An unnamed source's label skips the names the page uses, so it never
// replaces a page source or fills a role; only NAME= does. takesPage has the
// roles send (an on: button's input) and logs (a params merge's input).
func TestUnnamedLabelsSkipPageNames(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(takesPage), 0o644)
	cfg, err := config.LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := server.NewWithConfig(dir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}
	pageNames := pageSourceNames(cfg, srv)
	if got, want := slices.Sorted(maps.Keys(pageNames)), []string{"logs", "logview", "requests", "response", "send"}; !slices.Equal(got, want) {
		t.Fatalf("page names %q, want %q", got, want)
	}

	flags, err := flagSources([]string{"tail -f send.log", "logs=file:x.log"}, "docker logs -f requests", pageNames, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := slices.Sorted(maps.Keys(flags)), []string{"logs", "requests-2", "send-2"}; !slices.Equal(got, want) {
		t.Fatalf("sources %q, want %q: a label took a page name, or a NAME= was renamed", got, want)
	}

	var out strings.Builder
	if given := promptSources(strings.NewReader("tail -f logs\n\n"), &out, flags, pageNames, true); !slices.Equal(given, []string{"logs-2=tail -f logs"}) {
		t.Fatalf("prompt gave %q, want the label logs-2\n%s", given, out.String())
	}
}

func TestUniqueLabel(t *testing.T) {
	flags := map[string]config.SourceConfig{"api": {Type: "stream"}, "api-2": {Type: "stream"}}
	taken := func(l string) bool { return flags[l].Type != "" }
	if got := uniqueLabel("api", taken); got != "api-3" {
		t.Errorf("got %q, want api-3", got)
	}
	long := strings.Repeat("x", 32)
	flags[long] = config.SourceConfig{Type: "stream"}
	if got := uniqueLabel(long, taken); len(got) > 32 || !strings.HasSuffix(got, "-2") {
		t.Errorf("got %q, want a 32-character label ending in -2", got)
	}
}

// Everything after the first "--" is one command, run as typed: its own
// "--" and its quoting survive.
func TestCommandAfterDashes(t *testing.T) {
	cmd := commandAfterDashes([]string{"sh", "-c", `printf '%s|' "a b" c`})
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil || string(out) != "a b|c|" {
		t.Errorf("%s gave %q (err %v), want %q", cmd, out, err, "a b|c|")
	}
	// After "--" nothing is a name, so a leading assignment sets a variable.
	if out, err := exec.Command("sh", "-c", commandAfterDashes([]string{"FOO=1", "sh", "-c", "printf %s \"$FOO\""})).Output(); err != nil || string(out) != "1" {
		t.Errorf("-- FOO=1 cmd gave %q (err %v), want FOO set to 1", out, err)
	}
	// An empty argument stays one: sh -c '' runs nothing, where sh -c alone fails.
	if got := commandAfterDashes([]string{"sh", "-c", ""}); got != "sh -c ''" {
		t.Errorf("empty argument: got %q", got)
	}
	if got := commandAfterDashes([]string{"watchexec", "-e", "go", "--", "go", "test", "-json", "./..."}); got != "watchexec -e go -- go test -json ./..." {
		t.Errorf("got %q", got)
	}
}

// A flag after "--" belongs to the command, so serve doesn't read it.
func TestServeDashEndsFlags(t *testing.T) {
	defer config.SetAllowExec(false)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.md"), []byte("# Hi\n"), 0o644)
	err := ServeCommand([]string{dir, "--", "grep", "--verbose", "x"})
	if err == nil || !strings.Contains(err.Error(), "--allow-exec") {
		t.Errorf("got %v, want the --allow-exec refusal for the command, not an unknown flag", err)
	}
	// --help after "--" is the command's, so serve doesn't print its help.
	if err := ServeCommand([]string{dir, "--", "grep", "--help"}); err == nil || !strings.Contains(err.Error(), "--allow-exec") {
		t.Errorf("got %v, want the --allow-exec refusal, not serve's help", err)
	}
}

func TestReuseLineBeforeDashes(t *testing.T) {
	got := reuseLine([]string{"--allow-exec", ".", "--", "docker", "logs", "-f", "api"}, []string{"app=file:x.log"})
	if want := "tinkerdown serve --allow-exec . --source 'app=file:x.log' -- docker logs -f api"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestDefaultLabel(t *testing.T) {
	for cmd, want := range map[string]string{
		"docker logs -f api":                     "api",
		"tail -f /var/log/app.log":               "app",
		"watchexec -e go -- go test -json ./...": "test",
		"kubectl logs -f deploy/web":             "web",
		"journalctl -fu nginx":                   "nginx",
		`docker logs -f "api"`:                   "api",
		"./run.sh --all":                         "run",
		// A flag\'s value can win: serve can\'t know which flags take one.
		"curl -H Accept:x https://api.example.com/users": "users",
		"python -m http.server --port 80":                "80",
		"🙂 --x":                                          "cmd",
		"":                                               "cmd",
		strings.Repeat("y", 40) + " --x":                 strings.Repeat("y", 32),
	} {
		if got := defaultLabel(cmd); got != want {
			t.Errorf("defaultLabel(%q) = %q, want %q", cmd, got, want)
		}
	}
}

func TestSourceLabelFallbacks(t *testing.T) {
	for _, c := range []struct {
		src  config.SourceConfig
		want string
	}{
		{config.SourceConfig{Type: "stream", File: "/var/log/app.log"}, "app"},
		{config.SourceConfig{Type: "stream", HTTP: "./api.http"}, "api"},
		{config.SourceConfig{Type: "stream", File: "/tmp/🙂.log"}, "file"},
		{config.SourceConfig{Type: "stream", File: "/var/log/app.2024.log"}, "app2024"},
		{config.SourceConfig{Type: "stream", HTTP: "/tmp/🙂.http"}, "http"},
	} {
		if got := sourceLabel(c.src); got != c.want {
			t.Errorf("sourceLabel(%+v) = %q, want %q", c.src, got, c.want)
		}
	}
}

// --help wins over an unknown flag, so a mistyped line still shows help.
func TestServeHelpBeatsUnknownFlag(t *testing.T) {
	if err := ServeCommand([]string{"--bogus", "--help"}); err != nil {
		t.Fatalf("serve --bogus --help: %v", err)
	}
}
