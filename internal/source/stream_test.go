package source

import (
	"context"
	"errors"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/stream"
)

// A merge with params and no value in the query has no rows, but follows its
// gate so a value set later can open; a refused value is a validation error
// the page can show.
func TestParamsMergeWithoutValue(t *testing.T) {
	hub := stream.NewHub()
	defer hub.Close()
	sources := map[string]config.SourceConfig{
		"list":    {Type: "stream", File: "list.jsonl", Format: "json"},
		"things":  {Type: "merge", Inputs: []string{"list"}, Key: []string{"fields.name"}},
		"logs":    {Type: "stream", File: "logs.txt"},
		"logview": {Type: "merge", Inputs: []string{"logs"}, Params: map[string]config.SourceParam{"name": {In: "things.fields.name"}}},
	}
	lookup := func(n string) (config.SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	page := func(q string) *Page {
		v, _ := url.ParseQuery(q)
		return &Page{Query: v, Resolve: lookup, InputLookup: func(string) func(string) (config.SourceConfig, bool) { return lookup }}
	}

	src, err := NewStreamSource(hub, "logview", sources["logview"], t.TempDir(), lookup, page(""))
	if err != nil {
		t.Fatal(err)
	}
	if rows, _ := src.Fetch(context.Background()); rows != nil {
		t.Fatalf("rows without a value: %v", rows)
	}
	if sub := src.Subscribe(); sub == nil {
		t.Fatal("Subscribe without a value: want the gate's subscription, so a later value can open")
	} else {
		sub.Close()
	}

	src, err = NewStreamSource(hub, "logview", sources["logview"], t.TempDir(), lookup, page("name=-rf"))
	if err != nil {
		t.Fatal(err)
	}
	var ve *ValidationError
	if _, err := src.Fetch(context.Background()); !errors.As(err, &ve) {
		t.Fatalf("refused value: Fetch err = %v, want a ValidationError", err)
	}

	// A listed-later value: closed now, so no rows, but the page follows the gate.
	src, err = NewStreamSource(hub, "logview", sources["logview"], t.TempDir(), lookup, page("name=web"))
	if err != nil {
		t.Fatal(err)
	}
	sub := src.Subscribe()
	if sub == nil {
		t.Fatal("a gated source must follow its gate")
	}
	defer sub.Close()
	if rows, _ := src.Fetch(context.Background()); rows != nil {
		t.Fatalf("rows while the gate is closed: %v", rows)
	}
	if src.buf != nil {
		t.Fatal("a value never listed built its buffer; any link could add hub entries")
	}

	// The page resolves the gate to a different merge than the one this
	// source was validated against: refused.
	swapped := &Page{Query: url.Values{"name": {"web"}}, InputLookup: func(string) func(string) (config.SourceConfig, bool) { return lookup },
		Resolve: func(n string) (config.SourceConfig, bool) {
			if n == "things" {
				return config.SourceConfig{Type: "merge", Inputs: []string{"logs"}}, true
			}
			return lookup(n)
		}}
	if _, err := NewStreamSource(hub, "logview", sources["logview"], t.TempDir(), lookup, swapped); !errors.As(err, &ve) {
		t.Fatalf("swapped gate: err = %v, want a ValidationError", err)
	}
}

// SetQuery switches a params merge to other values on the same page: each
// value gets its own buffer, an unchanged value keeps it, and a refused or
// missing value shows no rows.
func TestParamsMergeSetQuery(t *testing.T) {
	orig := config.IsExecAllowed()
	config.SetAllowExec(true)
	defer config.SetAllowExec(orig)
	hub := stream.NewHub()
	defer hub.Close()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "list.jsonl"), []byte(`{"name":"a"}`+"\n"+`{"name":"b"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sources := map[string]config.SourceConfig{
		"list":    {Type: "stream", File: "list.jsonl", Format: "json"},
		"things":  {Type: "merge", Inputs: []string{"list"}, Key: []string{"fields.name"}},
		"logs":    {Type: "stream", Cmd: `echo "line from $TD_name"; sleep 60`, Format: "lines", Restart: "never"},
		"logview": {Type: "merge", Inputs: []string{"logs"}, Params: map[string]config.SourceParam{"name": {In: "things.fields.name"}}},
	}
	lookup := func(n string) (config.SourceConfig, bool) { s, ok := sources[n]; return s, ok }
	q := func(s string) url.Values { v, _ := url.ParseQuery(s); return v }
	page := &Page{Query: q("name=a"), Resolve: lookup, InputLookup: func(string) func(string) (config.SourceConfig, bool) { return lookup }}
	src, err := NewStreamSource(hub, "logview", sources["logview"], dir, lookup, page)
	if err != nil {
		t.Fatal(err)
	}
	sub := src.Subscribe()
	defer sub.Close()
	waitFor := func(want string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			rows, _ := src.Fetch(context.Background())
			if len(rows) == 1 && rows[0]["msg"] == want {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		rows, err := src.Fetch(context.Background())
		t.Fatalf("want one row %q, got %v (err %v)", want, rows, err)
	}
	waitFor("line from a")
	bufA := src.buf
	if got := src.Params(); !maps.Equal(got, map[string]string{"name": "a"}) {
		t.Fatalf("Params() = %v, want name=a", got)
	}

	if src.SetQuery(q("name=a&other=1")) || src.buf != bufA || !src.attached {
		t.Fatal("an unchanged value must report no change and keep its buffer attached")
	}
	if !src.SetQuery(q("name=b")) {
		t.Fatal("a new value must report a change")
	}
	waitFor("line from b")
	if src.buf == bufA {
		t.Fatal("b must get its own buffer")
	}

	src.SetQuery(q("name=-rf"))
	if src.SetQuery(q("name=-rf")) {
		t.Fatal("the same refused value must report no change")
	}
	if got := src.Params(); got != nil {
		t.Fatalf("Params() for a refused value = %v, want nil", got)
	}
	var ve *ValidationError
	if rows, err := src.Fetch(context.Background()); !errors.As(err, &ve) || rows != nil {
		t.Fatalf("refused value: rows %v, err %v; want no rows and a ValidationError", rows, err)
	}
	src.SetQuery(q(""))
	if src.SetQuery(q("other=1")) {
		t.Fatal("a value still missing must report no change")
	}
	if rows, err := src.Fetch(context.Background()); rows != nil || err != nil {
		t.Fatalf("no value: rows %v, err %v; want neither", rows, err)
	}
	src.SetQuery(q("name=a"))
	waitFor("line from a")

	plain, err := NewStreamSource(hub, "things", sources["things"], dir, lookup, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plain.SetQuery(q("name=a")) {
		t.Error("a merge without params reported using the query")
	}
	if got := plain.Params(); got != nil {
		t.Errorf("Params() of a merge without params = %v, want nil", got)
	}
}

func TestSameMergeIgnoresWildcardExpansion(t *testing.T) {
	want := config.SourceConfig{Type: "merge", Inputs: []string{"*"}, Key: []string{"fields.name"}}
	got := config.SourceConfig{Type: "merge", Inputs: []string{"docker"}, Wildcard: true, Key: []string{"fields.name"}}
	if !sameMerge(want, got) {
		t.Error("an expanded [\"*\"] merge should match its definition")
	}
	got.Key = []string{"fields.id"}
	if sameMerge(want, got) {
		t.Error("a different key matched")
	}
}
