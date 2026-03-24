package source

import (
	"context"
	"testing"

	"github.com/livetemplate/tinkerdown/internal/config"
)

// mockSource is a simple in-memory source for testing computed sources.
type mockComputedParent struct {
	name string
	data []map[string]interface{}
}

func (m *mockComputedParent) Name() string { return m.name }
func (m *mockComputedParent) Close() error { return nil }
func (m *mockComputedParent) Fetch(_ context.Context) ([]map[string]interface{}, error) {
	return m.data, nil
}

func newTestRegistry(sources map[string]Source) *Registry {
	return &Registry{
		sources: sources,
	}
}

func TestComputedSource_GroupByWithSum(t *testing.T) {
	parent := &mockComputedParent{
		name: "expenses",
		data: []map[string]interface{}{
			{"category": "Food", "amount": 10.0},
			{"category": "Transport", "amount": 25.0},
			{"category": "Food", "amount": 15.0},
			{"category": "Transport", "amount": 30.0},
			{"category": "Housing", "amount": 500.0},
		},
	}

	registry := newTestRegistry(map[string]Source{"expenses": parent})

	src, err := NewComputedSource("by_category", config.SourceConfig{
		Type:    "computed",
		From:    "expenses",
		GroupBy: "category",
		Aggregate: map[string]string{
			"total": "sum(amount)",
			"count": "count()",
		},
	}, registry)
	if err != nil {
		t.Fatalf("NewComputedSource failed: %v", err)
	}

	data, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(data) != 3 {
		t.Fatalf("expected 3 groups (Food, Transport, Housing), got %d", len(data))
	}

	// Build result map by category
	results := make(map[string]map[string]interface{})
	for _, row := range data {
		cat := row["category"].(string)
		results[cat] = row
	}

	// Check Food: total=25, count=2
	if results["Food"]["total"] != 25.0 {
		t.Errorf("Food total: expected 25.0, got %v", results["Food"]["total"])
	}
	if results["Food"]["count"] != 2 {
		t.Errorf("Food count: expected 2, got %v", results["Food"]["count"])
	}

	// Check Transport: total=55, count=2
	if results["Transport"]["total"] != 55.0 {
		t.Errorf("Transport total: expected 55.0, got %v", results["Transport"]["total"])
	}

	// Check Housing: total=500, count=1
	if results["Housing"]["total"] != 500.0 {
		t.Errorf("Housing total: expected 500.0, got %v", results["Housing"]["total"])
	}
	if results["Housing"]["count"] != 1 {
		t.Errorf("Housing count: expected 1, got %v", results["Housing"]["count"])
	}
}

func TestComputedSource_NoGroupBy(t *testing.T) {
	parent := &mockComputedParent{
		name: "expenses",
		data: []map[string]interface{}{
			{"amount": 10.0},
			{"amount": 20.0},
			{"amount": 30.0},
		},
	}

	registry := newTestRegistry(map[string]Source{"expenses": parent})

	src, err := NewComputedSource("totals", config.SourceConfig{
		Type: "computed",
		From: "expenses",
		Aggregate: map[string]string{
			"total":   "sum(amount)",
			"average": "avg(amount)",
			"count":   "count()",
		},
	}, registry)
	if err != nil {
		t.Fatalf("NewComputedSource failed: %v", err)
	}

	data, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(data) != 1 {
		t.Fatalf("expected 1 row (single aggregate), got %d", len(data))
	}

	row := data[0]
	if row["total"] != 60.0 {
		t.Errorf("total: expected 60.0, got %v", row["total"])
	}
	if row["average"] != 20.0 {
		t.Errorf("average: expected 20.0, got %v", row["average"])
	}
	if row["count"] != 3 {
		t.Errorf("count: expected 3, got %v", row["count"])
	}
}

func TestComputedSource_WithFilter(t *testing.T) {
	parent := &mockComputedParent{
		name: "tasks",
		data: []map[string]interface{}{
			{"title": "A", "status": "active", "points": 3.0},
			{"title": "B", "status": "done", "points": 5.0},
			{"title": "C", "status": "active", "points": 2.0},
			{"title": "D", "status": "done", "points": 8.0},
		},
	}

	registry := newTestRegistry(map[string]Source{"tasks": parent})

	src, err := NewComputedSource("active_stats", config.SourceConfig{
		Type:   "computed",
		From:   "tasks",
		Filter: "status = active",
		Aggregate: map[string]string{
			"count": "count()",
			"total": "sum(points)",
		},
	}, registry)
	if err != nil {
		t.Fatalf("NewComputedSource failed: %v", err)
	}

	data, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	row := data[0]
	if row["count"] != 2 {
		t.Errorf("count: expected 2 (active only), got %v", row["count"])
	}
	if row["total"] != 5.0 {
		t.Errorf("total: expected 5.0 (3+2), got %v", row["total"])
	}
}

func TestComputedSource_MinMax(t *testing.T) {
	parent := &mockComputedParent{
		name: "scores",
		data: []map[string]interface{}{
			{"value": 10.0},
			{"value": 50.0},
			{"value": 30.0},
		},
	}

	registry := newTestRegistry(map[string]Source{"scores": parent})

	src, err := NewComputedSource("stats", config.SourceConfig{
		Type: "computed",
		From: "scores",
		Aggregate: map[string]string{
			"min_val": "min(value)",
			"max_val": "max(value)",
		},
	}, registry)
	if err != nil {
		t.Fatal(err)
	}

	data, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	row := data[0]
	if row["min_val"] != 10.0 {
		t.Errorf("min: expected 10.0, got %v", row["min_val"])
	}
	if row["max_val"] != 50.0 {
		t.Errorf("max: expected 50.0, got %v", row["max_val"])
	}
}

func TestComputedSource_MissingParent(t *testing.T) {
	registry := newTestRegistry(map[string]Source{})

	_, err := NewComputedSource("bad", config.SourceConfig{
		Type: "computed",
		From: "nonexistent",
	}, registry)
	if err == nil {
		t.Fatal("expected error for missing parent source")
	}
}

func TestComputedSource_MissingFrom(t *testing.T) {
	registry := newTestRegistry(map[string]Source{})

	_, err := NewComputedSource("bad", config.SourceConfig{
		Type: "computed",
	}, registry)
	if err == nil {
		t.Fatal("expected error for missing 'from' field")
	}
}

func TestComputedSource_InvalidAggExpr(t *testing.T) {
	parent := &mockComputedParent{name: "x", data: nil}
	registry := newTestRegistry(map[string]Source{"x": parent})

	_, err := NewComputedSource("bad", config.SourceConfig{
		Type: "computed",
		From: "x",
		Aggregate: map[string]string{
			"bad": "invalid_func(field)",
		},
	}, registry)
	if err == nil {
		t.Fatal("expected error for invalid aggregate expression")
	}
}

func TestParseAggExpr(t *testing.T) {
	tests := []struct {
		expr    string
		fn      string
		field   string
		wantErr bool
	}{
		{"sum(amount)", "sum", "amount", false},
		{"count()", "count", "", false},
		{"avg(price)", "avg", "price", false},
		{"min(value)", "min", "value", false},
		{"max(score)", "max", "score", false},
		{"invalid(x)", "", "", true},
		{"sum", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		agg, err := parseAggExpr("out", tt.expr)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseAggExpr(%q): expected error", tt.expr)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseAggExpr(%q): unexpected error: %v", tt.expr, err)
			continue
		}
		if agg.function != tt.fn {
			t.Errorf("parseAggExpr(%q): function = %q, want %q", tt.expr, agg.function, tt.fn)
		}
		if agg.inputField != tt.field {
			t.Errorf("parseAggExpr(%q): field = %q, want %q", tt.expr, agg.inputField, tt.field)
		}
	}
}
