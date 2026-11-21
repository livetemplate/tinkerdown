package source

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/stream"
)

// A merge with params and no value in the query has no rows and nothing to
// follow; a refused value is a validation error the page can show.
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
	if sub := src.Subscribe(); sub != nil {
		t.Fatal("Subscribe without a value: want nil, so nothing is followed")
	}

	_, err = NewStreamSource(hub, "logview", sources["logview"], t.TempDir(), lookup, page("name=-rf"))
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("refused value: err = %v, want a ValidationError", err)
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
