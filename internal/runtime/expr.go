// Package runtime provides expression evaluation for computed expressions.
// Expressions are inline code spans starting with = that evaluate against source data.
// Example: `=count(tasks where done)` evaluates the count of done tasks.
package runtime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Expression represents a parsed computed expression.
type Expression interface {
	// Eval evaluates the expression against the given context.
	Eval(ctx *EvalContext) (interface{}, error)
}

// EvalContext provides data sources for expression evaluation.
type EvalContext struct {
	// Sources maps source names to their data rows.
	Sources map[string][]map[string]interface{}
}

// NewEvalContext creates a new evaluation context.
func NewEvalContext() *EvalContext {
	return &EvalContext{
		Sources: make(map[string][]map[string]interface{}),
	}
}

// ExprResult represents the result of expression evaluation.
type ExprResult struct {
	Value interface{}
	Error string
}

// Render returns an HTML representation of the result.
func (r *ExprResult) Render() string {
	if r.Error != "" {
		return fmt.Sprintf(`<span class="expr-error" title="%s">⚠</span>`, escapeAttr(r.Error))
	}
	return fmt.Sprintf(`%v`, r.Value)
}

// escapeAttr escapes a string for use in an HTML attribute.
func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// ParseExpr parses an expression string into an evaluable Expression.
// Supported syntax:
//   - count(source) - count all rows
//   - count(source where field) - count rows where field is truthy
//   - count(source where field = value) - count rows where field equals value
//   - sum(source.field) - sum a numeric field
//   - avg(source.field) - average a numeric field
//   - min(source.field) - minimum value of a field
//   - max(source.field) - maximum value of a field
func ParseExpr(input string) (Expression, error) {
	input = strings.TrimSpace(input)

	// Try to parse as function call
	funcExpr, err := parseFuncExpr(input)
	if err != nil {
		return nil, err
	}
	if funcExpr != nil {
		return funcExpr, nil
	}

	return nil, fmt.Errorf("invalid expression syntax: %s", input)
}

// FuncExpr represents a function call expression.
type FuncExpr struct {
	Name   string       // count, sum, avg, min, max
	Source string       // source name (e.g., "tasks")
	Field  string       // field name for sum/avg/min/max (e.g., "amount")
	Where  *WhereClause // optional where filter
}

// WhereClause represents a filter condition.
type WhereClause struct {
	Field    string      // field to filter on
	Operator string      // comparison operator: =, !=, <, >, <=, >=
	Value    interface{} // value to compare against
}

// funcPattern matches function calls like:
// count(tasks), count(tasks where done), sum(expenses.amount)
var funcPattern = regexp.MustCompile(`^(\w+)\(([^)]+)\)$`)

// parseFuncExpr parses a function expression.
func parseFuncExpr(input string) (*FuncExpr, error) {
	match := funcPattern.FindStringSubmatch(input)
	if match == nil {
		return nil, nil
	}

	funcName := strings.ToLower(match[1])
	args := strings.TrimSpace(match[2])

	// Validate function name
	switch funcName {
	case "count", "sum", "avg", "min", "max":
		// Valid
	default:
		return nil, fmt.Errorf("unknown function: %s", funcName)
	}

	expr := &FuncExpr{Name: funcName}

	// Check for "where" clause
	whereParts := strings.SplitN(args, " where ", 2)
	sourceField := strings.TrimSpace(whereParts[0])

	// Parse source and optional field (source.field)
	if strings.Contains(sourceField, ".") {
		parts := strings.SplitN(sourceField, ".", 2)
		expr.Source = parts[0]
		expr.Field = parts[1]
	} else {
		expr.Source = sourceField
	}

	// Parse where clause if present
	if len(whereParts) > 1 {
		where, err := parseWhereClause(whereParts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid where clause: %w", err)
		}
		expr.Where = where
	}

	// Validate: sum/avg/min/max require a field
	if funcName != "count" && expr.Field == "" {
		return nil, fmt.Errorf("%s requires a field: %s(source.field)", funcName, funcName)
	}

	return expr, nil
}

// parseWhereClause parses a where clause string.
// Supports:
//   - "field" -> field is truthy
//   - "field = value" -> field equals value
//   - "field != value" -> field not equals value
//   - "field > value" -> field greater than value
//   - "field < value" -> field less than value
//   - "field >= value" -> field greater or equal
//   - "field <= value" -> field less or equal
func parseWhereClause(input string) (*WhereClause, error) {
	input = strings.TrimSpace(input)

	// Try to match comparison operators (check multi-char first)
	operators := []string{"!=", ">=", "<=", "=", ">", "<"}
	for _, op := range operators {
		if idx := strings.Index(input, op); idx > 0 {
			field := strings.TrimSpace(input[:idx])
			valueStr := strings.TrimSpace(input[idx+len(op):])
			value := parseValue(valueStr)
			return &WhereClause{
				Field:    field,
				Operator: op,
				Value:    value,
			}, nil
		}
	}

	// No operator - treat as boolean field check
	return &WhereClause{
		Field:    input,
		Operator: "=",
		Value:    true,
	}, nil
}

// parseValue parses a value string into a typed value.
func parseValue(s string) interface{} {
	s = strings.TrimSpace(s)

	// Remove quotes if present
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		return s[1 : len(s)-1]
	}

	// Boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// String (unquoted)
	return s
}

// Eval evaluates the function expression.
func (e *FuncExpr) Eval(ctx *EvalContext) (interface{}, error) {
	data, ok := ctx.Sources[e.Source]
	if !ok {
		return nil, fmt.Errorf("source '%s' not found", e.Source)
	}

	// Apply where filter if present
	if e.Where != nil {
		data = filterData(data, e.Where)
	}

	switch e.Name {
	case "count":
		return len(data), nil
	case "sum":
		return sumField(data, e.Field)
	case "avg":
		sum, err := sumField(data, e.Field)
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return 0.0, nil
		}
		return sum / float64(len(data)), nil
	case "min":
		return minField(data, e.Field)
	case "max":
		return maxField(data, e.Field)
	}

	return nil, fmt.Errorf("unknown function: %s", e.Name)
}

// filterData filters rows based on a where clause.
func filterData(data []map[string]interface{}, where *WhereClause) []map[string]interface{} {
	var result []map[string]interface{}
	for _, row := range data {
		val := getFieldValue(row, where.Field)
		if matchesCondition(val, where.Operator, where.Value) {
			result = append(result, row)
		}
	}
	return result
}

// getFieldValue gets a field value from a row, checking both lowercase and titlecase.
func getFieldValue(row map[string]interface{}, field string) interface{} {
	if val, ok := row[field]; ok {
		return val
	}
	// Try titlecase
	if len(field) > 0 {
		titleField := strings.ToUpper(field[:1]) + field[1:]
		if val, ok := row[titleField]; ok {
			return val
		}
	}
	return nil
}

// matchesCondition checks if a value matches a condition.
func matchesCondition(val interface{}, operator string, target interface{}) bool {
	switch operator {
	case "=":
		return valuesEqual(val, target)
	case "!=":
		return !valuesEqual(val, target)
	case ">":
		return exprCompareValues(val, target) > 0
	case "<":
		return exprCompareValues(val, target) < 0
	case ">=":
		return exprCompareValues(val, target) >= 0
	case "<=":
		return exprCompareValues(val, target) <= 0
	}
	return false
}

// valuesEqual checks if two values are equal.
func valuesEqual(a, b interface{}) bool {
	// Handle nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle boolean target with truthy check
	if boolTarget, ok := b.(bool); ok {
		return isTruthy(a) == boolTarget
	}

	// Try numeric comparison
	aNum, aOk := tryFloat64(a)
	bNum, bOk := tryFloat64(b)
	if aOk && bOk {
		return aNum == bNum
	}

	// String comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// isTruthy checks if a value is truthy.
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != "" && val != "false" && val != "0"
	default:
		return true
	}
}

// exprCompareValues compares two values numerically.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func exprCompareValues(a, b interface{}) int {
	aNum, aOk := tryFloat64(a)
	bNum, bOk := tryFloat64(b)
	if aOk && bOk {
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
		return 0
	}

	// String comparison fallback
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if aStr < bStr {
		return -1
	}
	if aStr > bStr {
		return 1
	}
	return 0
}

// tryFloat64 converts a value to float64.
// Returns the float64 value and true if conversion succeeded, 0 and false otherwise.
func tryFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// sumField sums a numeric field across all rows.
func sumField(data []map[string]interface{}, field string) (float64, error) {
	var sum float64
	for _, row := range data {
		val := getFieldValue(row, field)
		if val == nil {
			continue // Skip nil values
		}
		num, ok := tryFloat64(val)
		if !ok {
			return 0, fmt.Errorf("field '%s' contains non-numeric value: %v", field, val)
		}
		sum += num
	}
	return sum, nil
}

// minField finds the minimum value of a field.
func minField(data []map[string]interface{}, field string) (interface{}, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var minVal interface{}
	var minNum float64
	first := true

	for _, row := range data {
		val := getFieldValue(row, field)
		if val == nil {
			continue
		}
		num, ok := tryFloat64(val)
		if !ok {
			// For non-numeric, compare as strings
			if first || fmt.Sprintf("%v", val) < fmt.Sprintf("%v", minVal) {
				minVal = val
				first = false
			}
			continue
		}
		if first || num < minNum {
			minNum = num
			minVal = val
			first = false
		}
	}

	return minVal, nil
}

// maxField finds the maximum value of a field.
func maxField(data []map[string]interface{}, field string) (interface{}, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var maxVal interface{}
	var maxNum float64
	first := true

	for _, row := range data {
		val := getFieldValue(row, field)
		if val == nil {
			continue
		}
		num, ok := tryFloat64(val)
		if !ok {
			// For non-numeric, compare as strings
			if first || fmt.Sprintf("%v", val) > fmt.Sprintf("%v", maxVal) {
				maxVal = val
				first = false
			}
			continue
		}
		if first || num > maxNum {
			maxNum = num
			maxVal = val
			first = false
		}
	}

	return maxVal, nil
}

// EvaluateExpressions evaluates a map of expression strings.
// Returns a map of expression ID to result.
func EvaluateExpressions(expressions map[string]string, ctx *EvalContext) map[string]*ExprResult {
	results := make(map[string]*ExprResult)
	for id, exprStr := range expressions {
		expr, err := ParseExpr(exprStr)
		if err != nil {
			results[id] = &ExprResult{Error: err.Error()}
			continue
		}

		value, err := expr.Eval(ctx)
		if err != nil {
			results[id] = &ExprResult{Error: err.Error()}
		} else {
			results[id] = &ExprResult{Value: value}
		}
	}
	return results
}
