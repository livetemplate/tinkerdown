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
  x: {type: stream, cmd: "true", file: a.log}`, "exactly one of cmd, file, http or sql"},
		{"both sql and cmd", `
sources:
  x: {type: stream, cmd: "true", sql: q.sql}`, "exactly one of cmd, file, http or sql"},
		{"valid sql", `
sources:
  q: {type: stream, sql: ./queries/}
  queries: {type: merge, inputs: [q], key: [fields.entry]}`, ""},
		{"on a built-in action", `
sources:
  q: {type: stream, sql: q.sql}
  queries: {type: merge, inputs: [q], key: [fields.entry]}
  run: {type: stream, cmd: "true"}
  result: {type: merge, inputs: [run], on: run, params: {entry: {in: queries.fields.entry}}}`, `on "run" is a built-in action`},
		{"sql named to start with -", `
sources:
  -q: {type: stream, sql: q.sql}`, "can't start with -"},
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
		"bad kind":        {Type: "merge", Inputs: []string{"*"}, Accepts: &SourceAccepts{Kinds: []string{"ftp"}}},
		"negative min":    {Type: "merge", Inputs: []string{"*"}, Accepts: &SourceAccepts{Min: -1}},
		"without [\"*\"]": {Type: "merge", Inputs: []string{"web"}, Accepts: &SourceAccepts{}},
	} {
		if err := ValidateStreamSource("all", ExpandInputs(src, nil, lookup), lookup); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestValidateKeyAndCollect(t *testing.T) {
	lookup := func(string) (SourceConfig, bool) { return SourceConfig{Type: "stream", File: "a.log"}, true }
	for _, ok := range []SourceConfig{
		{Type: "merge", Inputs: []string{"a"}, Key: []string{"fields.Package", "fields.Test"}, Collect: "fields.Output", CollectReset: "fields.Action=run"},
		{Type: "merge", Inputs: []string{"a"}, Key: []string{"fields.name"}, Drop: "fields.state=removed|dead"},
		{Type: "merge", Inputs: []string{"a"}, Key: []string{"fields.id"}, Sort: "-fields.createdAt"},
		{Type: "merge", Inputs: []string{"a"}, Key: []string{"fields.id"}, Sort: "seq"},
	} {
		if err := ValidateStreamSource("t", ok, lookup); err != nil {
			t.Fatalf("valid keyed merge: %v", err)
		}
	}
	for name, src := range map[string]SourceConfig{
		"key on a stream":   {Type: "stream", File: "a.log", Key: []string{"msg"}},
		"collect, no key":   {Type: "merge", Inputs: []string{"a"}, Collect: "fields.Output"},
		"bad path":          {Type: "merge", Inputs: []string{"a"}, Key: []string{"fields.a.b"}},
		"other nested root": {Type: "merge", Inputs: []string{"a"}, Key: []string{"meta.id"}},
		"reset without =":   {Type: "merge", Inputs: []string{"a"}, Key: []string{"x"}, Collect: "y", CollectReset: "fields.Action"},
		"reset, no collect": {Type: "merge", Inputs: []string{"a"}, Key: []string{"x"}, CollectReset: "fields.Action=run"},
		"drop on a stream":  {Type: "stream", File: "a.log", Drop: "fields.state=removed"},
		"reset on a stream": {Type: "stream", File: "a.log", CollectReset: "fields.Action=run"},
		"drop, no key":      {Type: "merge", Inputs: []string{"a"}, Drop: "fields.state=removed"},
		"drop without =":    {Type: "merge", Inputs: []string{"a"}, Key: []string{"x"}, Drop: "fields.state"},
		"drop, empty value": {Type: "merge", Inputs: []string{"a"}, Key: []string{"x"}, Drop: "fields.state="},
		"drop, bad path":    {Type: "merge", Inputs: []string{"a"}, Key: []string{"x"}, Drop: "a.b.c=removed"},
		"sort, no key":      {Type: "merge", Inputs: []string{"a"}, Sort: "-fields.createdAt"},
		"sort on a stream":  {Type: "stream", File: "a.log", Sort: "seq"},
		"sort, bad path":    {Type: "merge", Inputs: []string{"a"}, Key: []string{"x"}, Sort: "-a.b.c"},
		"sort, only -":      {Type: "merge", Inputs: []string{"a"}, Key: []string{"x"}, Sort: "-"},
		"sort, double -":    {Type: "merge", Inputs: []string{"a"}, Key: []string{"x"}, Sort: "--fields.t"},
	} {
		if err := ValidateStreamSource("t", src, lookup); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestKeySpecFromConfig(t *testing.T) {
	k := SourceConfig{Key: []string{"fields.Test"}, Collect: "fields.Output", CollectReset: "fields.Action=run", Drop: "fields.Action=skip|gone", Sort: "-fields.Elapsed"}.KeySpec()
	if k.Reset.String() != "fields.Action=run" || k.Drop.String() != "fields.Action=skip|gone" || k.Sort.String() != "-fields.Elapsed" || k.Collect != "fields.Output" || len(k.Paths) != 1 {
		t.Fatalf("KeySpec = %+v", k)
	}
}

func TestValidateParams(t *testing.T) {
	sources := map[string]SourceConfig{
		"logs":       {Type: "stream", Cmd: `docker logs -f -- "$TD_name"`},
		"containers": {Type: "merge", Inputs: []string{"logs"}, Key: []string{"fields.name"}},
		"gated":      {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"x": {In: "containers.fields.name"}}},
		"astream":    {Type: "stream", File: "a.log"},
		"logview":    {Type: "merge", Inputs: []string{"logs"}},
	}
	lookup := func(n string) (SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	ok := SourceConfig{Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"name": {In: "containers.fields.name"}}}
	if err := ValidateStreamSource("logview", ok, lookup); err != nil {
		t.Fatalf("valid params merge: %v", err)
	}
	for name, src := range map[string]SourceConfig{
		"params on a stream":   {Type: "stream", File: "a.log", Params: map[string]SourceParam{"name": {In: "containers.fields.name"}}},
		"bad param name":       {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"my-name": {In: "containers.fields.name"}}},
		"no in":                {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"name": {}}},
		"in without a path":    {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"name": {In: "containers"}}},
		"in, bad path":         {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"name": {In: "containers.a.b.c"}}},
		"in, unknown merge":    {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"name": {In: "nope.fields.name"}}},
		"in, not a merge":      {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"name": {In: "astream.msg"}}},
		"in, gated merge":      {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"name": {In: "gated.fields.name"}}},
		"in, itself":           {Type: "merge", Inputs: []string{"logs"}, Params: map[string]SourceParam{"name": {In: "logview.fields.name"}}},
		"wildcard with params": {Type: "merge", Inputs: []string{"*"}, Params: map[string]SourceParam{"name": {In: "containers.fields.name"}}},
	} {
		if err := ValidateStreamSource("logview", src, lookup); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestValidateOnAndLineMax(t *testing.T) {
	sources := map[string]SourceConfig{
		"api":      {Type: "stream", HTTP: "api.http"},
		"send":     {Type: "stream", Cmd: `curl -- "$TD_url"`},
		"requests": {Type: "merge", Inputs: []string{"api"}, Key: []string{"fields.entry"}},
	}
	lookup := func(n string) (SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	params := map[string]SourceParam{"entry": {In: "requests.fields.entry"}}
	ok := SourceConfig{Type: "merge", Inputs: []string{"send"}, On: "send", LineMax: "1MiB", Params: params}
	if err := ValidateStreamSource("response", ok, lookup); err != nil {
		t.Fatalf("valid click merge: %v", err)
	}
	if err := ValidateStreamSource("api", sources["api"], lookup); err != nil {
		t.Fatalf("http stream: %v", err)
	}
	for name, src := range map[string]SourceConfig{
		"on without params":   {Type: "merge", Inputs: []string{"send"}, On: "send"},
		"on, bad name":        {Type: "merge", Inputs: []string{"send"}, On: "Send!", Params: params},
		"on, input not a cmd": {Type: "merge", Inputs: []string{"api"}, On: "send", Params: params},
		"on on a stream":      {Type: "stream", Cmd: "true", On: "send"},
		"line_max on stream":  {Type: "stream", Cmd: "true", LineMax: "1MiB"},
		"line_max, bad unit":  {Type: "merge", Inputs: []string{"send"}, LineMax: "1MB"},
		"line_max, too big":   {Type: "merge", Inputs: []string{"send"}, LineMax: "17MiB"},
		"line_max, zero":      {Type: "merge", Inputs: []string{"send"}, LineMax: "0"},
		"http and cmd":        {Type: "stream", Cmd: "true", HTTP: "a.http"},
	} {
		if err := ValidateStreamSource("x", src, lookup); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	for v, want := range map[string]int{"": 0, "65536": 65536, "64KiB": 65536, "1MiB": 1 << 20} {
		if got, err := ParseLineMax(v); err != nil || got != want {
			t.Errorf("ParseLineMax(%q) = %d, %v; want %d", v, got, err, want)
		}
	}
	if k := sources["api"].Kind(); k != "http" {
		t.Errorf("Kind() = %q, want http", k)
	}
	acc := SourceConfig{Type: "merge", Inputs: []string{"*"}, Accepts: &SourceAccepts{Kinds: []string{"http"}}}
	if err := validateAccepts("requests", acc); err != nil {
		t.Errorf("accepts http: %v", err)
	}
	if k := (SourceConfig{Type: "stream", SQL: "q.sql"}).Kind(); k != "sql" {
		t.Errorf("Kind() = %q, want sql", k)
	}
	acc.Accepts.Kinds = []string{"sql", "csv"}
	if err := validateAccepts("queries", acc); err == nil || !strings.Contains(err.Error(), `"csv" is not one of cmd, file, http, sql`) {
		t.Errorf("accepts csv: %v", err)
	}
}

func TestValidateEveryAndPattern(t *testing.T) {
	sources := map[string]SourceConfig{
		"status": {Type: "stream", Cmd: "git status"},
		"log":    {Type: "stream", File: "app.log"},
		"deploy": {Type: "stream", Cmd: "./deploy.sh", Restart: "never"},
		"diff":   {Type: "stream", Cmd: `git diff -- "$TD_path"`},
		"files":  {Type: "merge", Inputs: []string{"status"}, Key: []string{"fields.path"}},
	}
	lookup := func(n string) (SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	params := map[string]SourceParam{"path": {In: "files.fields.path"}}
	for name, src := range map[string]SourceConfig{
		"every and pattern":  {Type: "merge", Inputs: []string{"status"}, Every: "2s", Key: []string{"fields.path"}, Pattern: `^(?P<x>.)(?P<y>.) (?P<path>.+)$`},
		"every with params":  {Type: "merge", Inputs: []string{"diff"}, Every: "2s", Params: params},
		"pattern on a file":  {Type: "merge", Inputs: []string{"log"}, Pattern: `^(?P<level>\w+) `},
		"the shortest every": {Type: "merge", Inputs: []string{"status"}, Every: "100ms"},
	} {
		if err := ValidateStreamSource("x", src, lookup); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, src := range map[string]SourceConfig{
		"every with on":          {Type: "merge", Inputs: []string{"diff"}, Every: "2s", On: "show", Params: params},
		"every under 100ms":      {Type: "merge", Inputs: []string{"status"}, Every: "50ms"},
		"every not a duration":   {Type: "merge", Inputs: []string{"status"}, Every: "often"},
		"every over a file":      {Type: "merge", Inputs: []string{"log"}, Every: "2s"},
		"every over run-once":    {Type: "merge", Inputs: []string{"deploy"}, Every: "2s"},
		"every on a stream":      {Type: "stream", Cmd: "true", Every: "2s"},
		"pattern on a stream":    {Type: "stream", Cmd: "true", Pattern: `(?P<a>.)`},
		"pattern without a name": {Type: "merge", Inputs: []string{"status"}, Pattern: `(.+)`},
		"pattern that fails":     {Type: "merge", Inputs: []string{"status"}, Pattern: `(?P<a>`},
	} {
		if err := ValidateStreamSource("x", src, lookup); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestValidateTypedParams(t *testing.T) {
	sources := map[string]SourceConfig{
		"commit": {Type: "stream", Cmd: `git commit -m "$TD_message"`},
		"files":  {Type: "merge", Inputs: []string{"commit"}},
	}
	lookup := func(n string) (SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	typed := map[string]SourceParam{"message": {Typed: true}}
	if err := ValidateStreamSource("committed", SourceConfig{Type: "merge", Inputs: []string{"commit"}, On: "commit", Params: typed}, lookup); err != nil {
		t.Fatalf("a typed param on an on merge: %v", err)
	}
	for name, src := range map[string]SourceConfig{
		"typed without on": {Type: "merge", Inputs: []string{"commit"}, Params: typed},
		"typed with in":    {Type: "merge", Inputs: []string{"commit"}, On: "commit", Params: map[string]SourceParam{"message": {Typed: true, In: "files.fields.x"}}},
	} {
		if err := ValidateStreamSource("x", src, lookup); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if err := CheckIn(SourceParam{Typed: true}, "", lookup); err == nil {
		t.Error("CheckIn accepted a typed value as a gate")
	}
}

// A page action converted to config's Action has an empty Params map; it
// still counts as values-only, or the --action that replaces it would drop
// the page's checks.
func TestValuesOnlyIgnoresEmptyMaps(t *testing.T) {
	values := map[string]SourceParam{"path": {In: "files.fields.path"}}
	if a := (&Action{Values: values, Params: map[string]ParamDef{}, Statements: []string{}}); !a.ValuesOnly() {
		t.Fatal("empty Params and Statements made a values-only action count as running something")
	}
	if a := (&Action{Values: values, Confirm: "Sure?"}); a.ValuesOnly() {
		t.Fatal("an action with confirm counts as values-only")
	}
}
