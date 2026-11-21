package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/livetemplate/tinkerdown/internal/config"
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
