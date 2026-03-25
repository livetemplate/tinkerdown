package source

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/livetemplate/tinkerdown/internal/config"
)

// ComputedSource derives data from another source by applying group_by and
// aggregate operations. It implements the Source interface (read-only).
//
// Example config:
//
//	sources:
//	  expenses:
//	    type: sqlite
//	    db: ./expenses.db
//	    table: expenses
//	  by_category:
//	    type: computed
//	    from: expenses
//	    group_by: category
//	    aggregate:
//	      total: sum(amount)
//	      count: count()
type ComputedSource struct {
	name    string
	parent  Source
	groupBy string
	aggs    []aggDef
	filter  *filterDef
}

// aggDef defines a single aggregation: output field name + function + input field.
type aggDef struct {
	outputField string // e.g., "total"
	function    string // e.g., "sum", "count", "avg", "min", "max"
	inputField  string // e.g., "amount" (empty for count)
}

// filterDef defines a simple filter: field operator value.
type filterDef struct {
	field    string
	operator string
	value    string
}

var aggPattern = regexp.MustCompile(`^(count|sum|avg|min|max)\(([^)]*)\)$`)
var filterPattern = regexp.MustCompile(`^(\w+)\s*(=|!=|<|>|<=|>=)\s*(.+)$`)

// NewComputedSource creates a computed source that derives data from a parent source.
// The parent source must already be registered in the registry.
func NewComputedSource(name string, cfg config.SourceConfig, registry *Registry) (*ComputedSource, error) {
	if cfg.From == "" {
		return nil, fmt.Errorf("computed source %q: 'from' field is required", name)
	}

	parent, ok := registry.Get(cfg.From)
	if !ok {
		return nil, fmt.Errorf("computed source %q: parent source %q not found", name, cfg.From)
	}

	if len(cfg.Aggregate) == 0 {
		return nil, fmt.Errorf("computed source %q: 'aggregate' is required (at least one aggregation)", name)
	}

	// Parse aggregate definitions (sorted by output field for deterministic ordering)
	aggKeys := make([]string, 0, len(cfg.Aggregate))
	for k := range cfg.Aggregate {
		aggKeys = append(aggKeys, k)
	}
	sort.Strings(aggKeys)

	var aggs []aggDef
	for _, outputField := range aggKeys {
		agg, err := parseAggExpr(outputField, cfg.Aggregate[outputField])
		if err != nil {
			return nil, fmt.Errorf("computed source %q: %w", name, err)
		}
		aggs = append(aggs, agg)
	}

	// Parse filter if present
	var filter *filterDef
	if cfg.Filter != "" {
		f, err := parseFilter(cfg.Filter)
		if err != nil {
			return nil, fmt.Errorf("computed source %q: %w", name, err)
		}
		filter = f
	}

	return &ComputedSource{
		name:    name,
		parent:  parent,
		groupBy: cfg.GroupBy,
		aggs:    aggs,
		filter:  filter,
	}, nil
}

func (s *ComputedSource) Name() string { return s.name }
func (s *ComputedSource) Close() error { return nil }

// Fetch retrieves data from the parent source, applies optional filter,
// groups by the specified field, and computes aggregations per group.
func (s *ComputedSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Fetch parent data
	data, err := s.parent.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("computed source %q: parent fetch failed: %w", s.name, err)
	}

	// Apply filter if present
	if s.filter != nil {
		data = applyFilter(data, s.filter)
	}

	// If no group_by, compute a single aggregate row
	if s.groupBy == "" {
		row := make(map[string]interface{})
		for _, agg := range s.aggs {
			val, err := computeAgg(data, agg)
			if err != nil {
				return nil, err
			}
			row[agg.outputField] = val
		}
		return []map[string]interface{}{row}, nil
	}

	// Group data by field
	groups := groupData(data, s.groupBy)

	// Sort group keys for deterministic output ordering
	groupKeys := make([]string, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	sort.Strings(groupKeys)

	// Compute aggregations per group
	var results []map[string]interface{}
	for _, groupValue := range groupKeys {
		groupRows := groups[groupValue]
		row := map[string]interface{}{
			s.groupBy: groupValue,
		}
		for _, agg := range s.aggs {
			val, err := computeAgg(groupRows, agg)
			if err != nil {
				return nil, err
			}
			row[agg.outputField] = val
		}
		results = append(results, row)
	}

	return results, nil
}

// groupData groups rows by a field value.
func groupData(data []map[string]interface{}, field string) map[string][]map[string]interface{} {
	groups := make(map[string][]map[string]interface{})
	for _, row := range data {
		key := fmt.Sprintf("%v", getField(row, field))
		groups[key] = append(groups[key], row)
	}
	return groups
}

// computeAgg computes a single aggregation over a set of rows.
func computeAgg(rows []map[string]interface{}, agg aggDef) (interface{}, error) {
	switch agg.function {
	case "count":
		return len(rows), nil
	case "sum":
		return sumValues(rows, agg.inputField)
	case "avg":
		sum, err := sumValues(rows, agg.inputField)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return 0.0, nil
		}
		return sum / float64(len(rows)), nil
	case "min":
		return minValue(rows, agg.inputField)
	case "max":
		return maxValue(rows, agg.inputField)
	default:
		return nil, fmt.Errorf("unknown aggregation function %q", agg.function)
	}
}

// sumValues sums a numeric field across rows.
func sumValues(rows []map[string]interface{}, field string) (float64, error) {
	var total float64
	for _, row := range rows {
		val := getField(row, field)
		if val == nil {
			continue
		}
		f, ok := toFloat64(val)
		if !ok {
			continue
		}
		total += f
	}
	return total, nil
}

// minValue returns the minimum numeric value of a field.
func minValue(rows []map[string]interface{}, field string) (interface{}, error) {
	min := math.MaxFloat64
	found := false
	for _, row := range rows {
		val := getField(row, field)
		if val == nil {
			continue
		}
		f, ok := toFloat64(val)
		if !ok {
			continue
		}
		if f < min {
			min = f
			found = true
		}
	}
	if !found {
		return nil, nil
	}
	return min, nil
}

// maxValue returns the maximum numeric value of a field.
func maxValue(rows []map[string]interface{}, field string) (interface{}, error) {
	max := -math.MaxFloat64
	found := false
	for _, row := range rows {
		val := getField(row, field)
		if val == nil {
			continue
		}
		f, ok := toFloat64(val)
		if !ok {
			continue
		}
		if f > max {
			max = f
			found = true
		}
	}
	if !found {
		return nil, nil
	}
	return max, nil
}

// getField gets a field value from a row (case-insensitive).
func getField(row map[string]interface{}, field string) interface{} {
	if field == "" {
		return nil
	}
	if val, ok := row[field]; ok {
		return val
	}
	// Try titlecase
	titleField := strings.ToUpper(field[:1]) + field[1:]
	if val, ok := row[titleField]; ok {
		return val
	}
	// Try lowercase
	lowerField := strings.ToLower(field)
	if val, ok := row[lowerField]; ok {
		return val
	}
	return nil
}

// toFloat64 converts a value to float64.
func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case string:
		// Try to parse string as number
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// applyFilter filters rows based on a filter definition.
func applyFilter(data []map[string]interface{}, f *filterDef) []map[string]interface{} {
	var result []map[string]interface{}
	for _, row := range data {
		val := getField(row, f.field)
		if val == nil {
			continue
		}
		strVal := fmt.Sprintf("%v", val)
		if matchFilter(strVal, f.operator, f.value) {
			result = append(result, row)
		}
	}
	return result
}

// matchFilter checks if a value matches a filter condition.
func matchFilter(val, operator, target string) bool {
	switch operator {
	case "=":
		return strings.EqualFold(val, target)
	case "!=":
		return !strings.EqualFold(val, target)
	case "<", ">", "<=", ">=":
		fVal, ok1 := toFloat64(val)
		fTarget, ok2 := toFloat64(target)
		if !ok1 || !ok2 {
			return false
		}
		switch operator {
		case "<":
			return fVal < fTarget
		case ">":
			return fVal > fTarget
		case "<=":
			return fVal <= fTarget
		case ">=":
			return fVal >= fTarget
		}
	}
	return false
}

// parseAggExpr parses an aggregation expression like "sum(amount)" into an aggDef.
func parseAggExpr(outputField, expr string) (aggDef, error) {
	expr = strings.TrimSpace(expr)
	match := aggPattern.FindStringSubmatch(expr)
	if match == nil {
		return aggDef{}, fmt.Errorf("invalid aggregate expression %q (expected: sum(field), count(), avg(field), min(field), max(field))", expr)
	}
	return aggDef{
		outputField: outputField,
		function:    match[1],
		inputField:  strings.TrimSpace(match[2]),
	}, nil
}

// parseFilter parses a filter expression like "status = active" into a filterDef.
func parseFilter(expr string) (*filterDef, error) {
	expr = strings.TrimSpace(expr)
	match := filterPattern.FindStringSubmatch(expr)
	if match == nil {
		return nil, fmt.Errorf("invalid filter expression %q (expected: field operator value)", expr)
	}
	return &filterDef{
		field:    match[1],
		operator: match[2],
		value:    strings.TrimSpace(match[3]),
	}, nil
}
