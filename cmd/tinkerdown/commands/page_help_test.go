package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// A page shaped like onefile's http and containers apps: a ["*"] merge, a
// merge a button starts, a merge a link opens, and buttons.
const takesPage = "---\ntitle: T\nsources:\n" +
	"  requests: {type: merge, inputs: [\"*\"], key: [fields.entry], accepts: {kinds: [http], min: 1, hint: \"your .http file \\x1b[31m<script>\"}}\n" +
	"  response: {type: merge, inputs: [send], on: send, params: {entry: {in: requests.fields.entry}}}\n" +
	"  logview: {type: merge, inputs: [logs], params: {name: {in: requests.fields.name}}}\n" +
	"---\n\n```lvt\n<main lvt-source=\"requests\">{{range .Data}}<li data-key=\"{{.key}}\">" +
	"<button name=\"send\" data-entry=\"{{.fields.entry}}\">Send</button>" +
	"<button name=\"stop\" data-name=\"{{.fields.name}}\">Stop</button>" +
	"<button name=\"{{.fields.verb}}\">Dynamic</button>" +
	"</li>{{end}}<button name=\"Pause\">Pause</button></main>\n```\n"

func TestDescribeTakes(t *testing.T) {
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
	var out strings.Builder
	describeTakes(&out, cfg, srv, "github.com/o/r/http")
	got := out.String()
	for _, want := range []string{
		"github.com/o/r/http takes:",
		"--source http:PATH\n      1 or more, shown in requests",
		"--source send=…\n      run by the send button, shown in response",
		"--source logs=…\n      run for the row you open, shown in logview",
		"--action stop=COMMAND",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, bad := range []string{"\x1b", "--action send", "--action Pause", "--action {{"} {
		if strings.Contains(got, bad) {
			t.Errorf("must not print %q:\n%s", bad, got)
		}
	}
	if !strings.Contains(got, `<script>`) {
		t.Errorf("the hint prints as text, quoted:\n%s", got)
	}
}

// serve <page> --help loads the page without reading the flags it would run,
// so it needs no --allow-exec and starts nothing.
func TestServePageHelp(t *testing.T) {
	defer config.SetAllowExec(false)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(takesPage), 0o644)
	if err := ServeCommand([]string{dir, "--source", "send=curl x", "--help"}); err != nil {
		t.Fatalf("serve <page> --help: %v", err)
	}
	if err := ServeCommand([]string{dir, "--help", "--", "docker", "logs"}); err != nil {
		t.Fatalf("serve <page> --help -- cmd: %v", err)
	}
}
