package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestValidateStreamSources(t *testing.T) {
	cases := []struct {
		name, yaml, wantErr string
	}{
		{"valid", `
sources:
  web: {type: stream, cmd: "docker logs -f web"}
  app: {type: stream, file: app.log, format: json}
  all: {type: merge, inputs: [web, app], flush: 200ms}`, ""},
		{"both cmd and file", `
sources:
  x: {type: stream, cmd: "true", file: a.log}`, "exactly one of cmd or file"},
		{"bad format", `
sources:
  x: {type: stream, file: a.log, format: yaml}`, "format"},
		{"bad restart", `
sources:
  x: {type: stream, cmd: "true", restart: always}`, "restart"},
		{"bad flush", `
sources:
  x: {type: stream, cmd: "true", flush: soon}`, "flush"},
		{"merge without inputs", `
sources:
  all: {type: merge}`, "inputs lists no sources"},
		{"merge of unknown", `
sources:
  all: {type: merge, inputs: [nope]}`, `"nope" is not a defined source`},
		{"merge of non-stream", `
sources:
  db: {type: sqlite, table: t}
  all: {type: merge, inputs: [db]}`, "not a stream"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tinkerdown.yaml")
			os.WriteFile(path, []byte(c.yaml), 0o644)
			_, err := Load(path)
			if c.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
				t.Fatalf("error %v, want one containing %q", err, c.wantErr)
			}
		})
	}
}

func TestExpandInputs(t *testing.T) {
	sources := map[string]SourceConfig{
		"web":   {Type: "stream", Cmd: "a"},
		"app":   {Type: "stream", File: "a.log"},
		"users": {Type: "json", File: "u.json"},
	}
	lookup := func(n string) (SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	names := []string{"web", "users", "app", "web"}

	got := ExpandInputs(SourceConfig{Type: "merge", Inputs: []string{"*"}}, names, lookup)
	if want := []string{"app", "web"}; !slices.Equal(got.Inputs, want) {
		t.Fatalf("expanded to %v, want %v (streams only, sorted, no duplicates)", got.Inputs, want)
	}
	if err := ValidateStreamSource("all", got, lookup); err != nil {
		t.Fatalf("expanded merge: %v", err)
	}

	empty := ExpandInputs(SourceConfig{Type: "merge", Inputs: []string{"*"}}, nil, lookup)
	if err := ValidateStreamSource("all", empty, lookup); err != nil {
		t.Fatalf("[\"*\"] with no sources yet must be valid: %v", err)
	}
	if err := ValidateStreamSource("all", SourceConfig{Type: "merge", Inputs: []string{}}, lookup); err == nil {
		t.Fatal("an explicit empty inputs list must stay an error")
	}
	if err := ValidateStreamSource("all", SourceConfig{Type: "merge", Inputs: []string{"*"}}, lookup); err == nil {
		t.Fatal("an unexpanded [\"*\"] must be an error")
	}

	fileOnly := SourceConfig{Type: "merge", Inputs: []string{"*"}, Accepts: &SourceAccepts{Kinds: []string{"file"}}}
	if got := ExpandInputs(fileOnly, names, lookup); !slices.Equal(got.Inputs, []string{"app"}) {
		t.Fatalf("kinds [file] expanded to %v, want [app]", got.Inputs)
	}
}

func TestValidateAccepts(t *testing.T) {
	lookup := func(string) (SourceConfig, bool) { return SourceConfig{}, false }
	for name, src := range map[string]SourceConfig{
		"bad kind":        {Type: "merge", Inputs: []string{"*"}, Accepts: &SourceAccepts{Kinds: []string{"http"}}},
		"negative min":    {Type: "merge", Inputs: []string{"*"}, Accepts: &SourceAccepts{Min: -1}},
		"without [\"*\"]": {Type: "merge", Inputs: []string{"web"}, Accepts: &SourceAccepts{}},
	} {
		if err := ValidateStreamSource("all", ExpandInputs(src, nil, lookup), lookup); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
