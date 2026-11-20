package config

import (
	"os"
	"path/filepath"
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
