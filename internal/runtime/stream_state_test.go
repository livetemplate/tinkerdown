package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/stream"
)

func TestParseWhereClausePicksEarliestOperator(t *testing.T) {
	cases := []struct{ in, field, op string }{
		{"msg ~ k=v", "msg", "~"},
		{"path = a~b", "path", "="},
		{"n >= 3", "n", ">="},
		{"status != done", "status", "!="},
		{"level = error", "level", "="},
	}
	for _, c := range cases {
		w, err := parseWhereClause(c.in)
		if err != nil || w.Field != c.field || w.Operator != c.op {
			t.Errorf("%q: got field %q op %q err %v, want %q %q", c.in, w.Field, w.Operator, err, c.field, c.op)
		}
	}
}

func TestFilterSubstringAndConjunction(t *testing.T) {
	s := &GenericState{Data: []map[string]interface{}{
		{"source": "web", "line": "GET /api/Users 200"},
		{"source": "web", "line": "POST /login 500"},
		{"source": "db", "line": "slow query on users"},
	}}
	s.handleFilter(map[string]interface{}{"filter": "source = web and line ~ users"})
	got := s.GetFilteredData()
	if len(got) != 1 || got[0]["line"] != "GET /api/Users 200" {
		t.Fatalf("got %v, want the one web row mentioning users (case-insensitive)", got)
	}
}

func TestFilterMessagesUpdateOnlyTheirPart(t *testing.T) {
	s := &GenericState{Data: []map[string]interface{}{
		{"source": "web", "level": "error"},
		{"source": "web", "level": "info"},
		{"source": "db", "level": "error"},
	}}
	s.handleFilter(map[string]interface{}{"filter": "source = web"}) // a tab
	s.handleFilter(map[string]interface{}{"f.level": "error"})       // the level select
	if got := s.GetFilteredData(); len(got) != 1 {
		t.Fatalf("tab + level: got %d rows, want 1", len(got))
	}
	s.handleFilter(map[string]interface{}{"filter": ""}) // the All tab keeps the level filter
	if got := s.GetFilteredData(); len(got) != 2 {
		t.Fatalf("All tab + level: got %d rows, want 2", len(got))
	}
	s.handleFilter(map[string]interface{}{"f.level": ""}) // "any level" clears it
	if got := s.GetFilteredData(); len(got) != 3 {
		t.Fatalf("no filters: got %d rows, want 3", len(got))
	}
	s.handleFilter(map[string]interface{}{}) // legacy: an empty Filter clears the expression
	if s.activeFilter != "" {
		t.Fatalf("empty Filter kept %q", s.activeFilter)
	}
}

func TestStreamControlsOnAFileStream(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	os.WriteFile(path, []byte("one\ntwo\n"), 0o644)
	hub := stream.NewHub()
	defer hub.Close()

	s, err := NewGenericStateWithMetadata("app", config.SourceConfig{Type: "stream", File: "app.log", Flush: "50ms"}, dir, "", nil, WithStreams(hub, nil))
	if err != nil {
		t.Fatal(err)
	}
	if s.FlushInterval() != 50*time.Millisecond {
		t.Fatalf("flush = %v", s.FlushInterval())
	}
	sub := s.Subscribe()
	defer sub.Close()
	waitRows := func(n int) {
		t.Helper()
		for i := 0; i < 250; i++ {
			s.HandleAction("Refresh", nil)
			if len(s.GetFilteredData()) == n {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("rows = %d, want %d", len(s.GetFilteredData()), n)
	}
	waitRows(2)

	if err := s.HandleAction("Pause", nil); err != nil || !s.IsPaused() {
		t.Fatalf("Pause: err %v paused %v", err, s.IsPaused())
	}
	if err := s.HandleAction("Clear", nil); err != nil {
		t.Fatal(err)
	}
	if got := len(s.GetFilteredData()); got != 0 {
		t.Fatalf("after Clear: %d rows visible, want 0", got)
	}
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	fmt.Fprintln(f, "three")
	f.Close()
	if err := s.HandleAction("Resume", nil); err != nil || s.IsPaused() {
		t.Fatalf("Resume: err %v paused %v", err, s.IsPaused())
	}
	waitRows(1) // only the row after Clear
	if s.GetFilteredData()[0]["msg"] != "three" {
		t.Fatalf("visible row = %v", s.GetFilteredData()[0])
	}
}

func TestStreamControlsRejectedForOtherSources(t *testing.T) {
	s := &GenericState{sourceType: "json"}
	if err := s.HandleAction("Pause", nil); err == nil {
		t.Fatal("Pause on a json source: want error")
	}
	for _, a := range []string{"pause", "Resume", "CLEAR"} {
		if !IsBuiltinAction(a) {
			t.Errorf("IsBuiltinAction(%q) = false", a)
		}
	}
}

func TestStreamSourceNeedsHubAndExecFlag(t *testing.T) {
	if _, err := NewGenericStateWithMetadata("x", config.SourceConfig{Type: "stream", File: "a.log"}, t.TempDir(), "", nil); err == nil {
		t.Fatal("stream without a hub: want error")
	}
	hub := stream.NewHub()
	defer hub.Close()
	config.SetAllowExec(false)
	if _, err := NewGenericStateWithMetadata("x", config.SourceConfig{Type: "stream", Cmd: "echo hi"}, t.TempDir(), "", nil, WithStreams(hub, nil)); err == nil {
		t.Fatal("cmd stream without --allow-exec: want error")
	}
}
