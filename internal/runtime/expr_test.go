package runtime

import (
	"testing"
)

func TestParseExpr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"count simple", "count(tasks)", false},
		{"count with where", "count(tasks where done)", false},
		{"count with where equals", "count(tasks where status = done)", false},
		{"sum with field", "sum(expenses.amount)", false},
		{"avg with field", "avg(scores.value)", false},
		{"min with field", "min(dates.created)", false},
		{"max with field", "max(prices.cost)", false},
		{"sum without field", "sum(expenses)", true},
		{"unknown function", "unknown(tasks)", true},
		{"invalid syntax", "not a valid expression", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseExpr(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExpr(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestEvalCount(t *testing.T) {
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"tasks": {
				{"id": 1, "done": false, "title": "Task 1"},
				{"id": 2, "done": true, "title": "Task 2"},
				{"id": 3, "done": false, "title": "Task 3"},
			},
		},
	}

	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		{"count all", "count(tasks)", 3},
		{"count where done", "count(tasks where done)", 1},
		{"count where not done", "count(tasks where done = false)", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpr(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpr failed: %v", err)
			}

			result, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestEvalSum(t *testing.T) {
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"expenses": {
				{"amount": 50.50, "category": "food"},
				{"amount": 100.00, "category": "transport"},
				{"amount": 25.25, "category": "food"},
			},
		},
	}

	tests := []struct {
		name     string
		expr     string
		expected float64
	}{
		{"sum all", "sum(expenses.amount)", 175.75},
		{"sum with where", "sum(expenses.amount where category = food)", 75.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpr(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpr failed: %v", err)
			}

			result, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvalAvg(t *testing.T) {
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"scores": {
				{"value": 80.0},
				{"value": 90.0},
				{"value": 100.0},
			},
		},
	}

	expr, err := ParseExpr("avg(scores.value)")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	result, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	expected := 90.0
	if result != expected {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestEvalAvgEmpty(t *testing.T) {
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"scores": {},
		},
	}

	expr, err := ParseExpr("avg(scores.value)")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	result, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	expected := 0.0
	if result != expected {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestEvalMinMax(t *testing.T) {
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"items": {
				{"price": 10.0},
				{"price": 5.0},
				{"price": 15.0},
			},
		},
	}

	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		{"min", "min(items.price)", 5.0},
		{"max", "max(items.price)", 15.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpr(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpr failed: %v", err)
			}

			result, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvalWhereClause(t *testing.T) {
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"products": {
				{"name": "Apple", "price": 1.5, "category": "fruit"},
				{"name": "Milk", "price": 2.0, "category": "dairy"},
				{"name": "Banana", "price": 0.5, "category": "fruit"},
				{"name": "Cheese", "price": 5.0, "category": "dairy"},
			},
		},
	}

	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		{"equals string", "count(products where category = fruit)", 2},
		{"not equals", "count(products where category != fruit)", 2},
		{"greater than", "count(products where price > 1.0)", 3},
		{"less than", "count(products where price < 2.0)", 2},
		{"greater or equal", "count(products where price >= 2.0)", 2},
		{"less or equal", "count(products where price <= 2.0)", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpr(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpr failed: %v", err)
			}

			result, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvalMissingSource(t *testing.T) {
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{},
	}

	expr, err := ParseExpr("count(nonexistent)")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	_, err = expr.Eval(ctx)
	if err == nil {
		t.Error("expected error for missing source")
	}
}

func TestEvalNilValues(t *testing.T) {
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"items": {
				{"value": 10.0},
				{"value": nil},
				{"value": 20.0},
			},
		},
	}

	expr, err := ParseExpr("sum(items.value)")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	result, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	// nil values should be skipped
	expected := 30.0
	if result != expected {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestEvalTitlecaseFields(t *testing.T) {
	// Test that titlecase field names work (e.g., "Done" instead of "done")
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"tasks": {
				{"Done": true},
				{"Done": false},
				{"Done": true},
			},
		},
	}

	expr, err := ParseExpr("count(tasks where done)")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	result, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	expected := 2
	if result != expected {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestEvaluateExpressions(t *testing.T) {
	expressions := map[string]string{
		"expr-0": "count(tasks)",
		"expr-1": "count(tasks where done)",
		"expr-2": "sum(expenses.amount)",
		"expr-3": "count(nonexistent)", // Should produce an error
	}

	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"tasks": {
				{"done": true},
				{"done": false},
				{"done": true},
			},
			"expenses": {
				{"amount": 10.0},
				{"amount": 20.0},
			},
		},
	}

	results := EvaluateExpressions(expressions, ctx)

	if len(results) != 4 {
		t.Errorf("expected 4 results, got %d", len(results))
	}

	// Check expr-0: count(tasks)
	if results["expr-0"].Value != 3 || results["expr-0"].Error != "" {
		t.Errorf("expr-0: got value=%v error=%v, want value=3", results["expr-0"].Value, results["expr-0"].Error)
	}

	// Check expr-1: count(tasks where done)
	if results["expr-1"].Value != 2 || results["expr-1"].Error != "" {
		t.Errorf("expr-1: got value=%v error=%v, want value=2", results["expr-1"].Value, results["expr-1"].Error)
	}

	// Check expr-2: sum(expenses.amount)
	if results["expr-2"].Value != 30.0 || results["expr-2"].Error != "" {
		t.Errorf("expr-2: got value=%v error=%v, want value=30.0", results["expr-2"].Value, results["expr-2"].Error)
	}

	// Check expr-3: should have error
	if results["expr-3"].Error == "" {
		t.Error("expr-3: expected error for missing source")
	}
}

func TestExprResultRender(t *testing.T) {
	tests := []struct {
		name     string
		result   *ExprResult
		contains string
	}{
		{"value", &ExprResult{Value: 42}, "42"},
		{"error", &ExprResult{Error: "something went wrong"}, "expr-error"},
		{"float", &ExprResult{Value: 3.14}, "3.14"},
		{"string", &ExprResult{Value: "hello"}, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := tt.result.Render()
			if len(rendered) == 0 {
				t.Error("render returned empty string")
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"true", true},
		{"false", false},
		{"42", int64(42)},
		{"3.14", 3.14},
		{`"hello"`, "hello"},
		{`'world'`, "world"},
		{"unquoted", "unquoted"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseValue(tt.input)
			if result != tt.expected {
				t.Errorf("parseValue(%q) = %v (%T), want %v (%T)",
					tt.input, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestIntegerValues(t *testing.T) {
	// Test that integer values work correctly (not just float64)
	ctx := &EvalContext{
		Sources: map[string][]map[string]interface{}{
			"items": {
				{"count": 5},
				{"count": 10},
				{"count": 15},
			},
		},
	}

	expr, err := ParseExpr("sum(items.count)")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	result, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	expected := 30.0 // sum always returns float64
	if result != expected {
		t.Errorf("got %v, want %v", result, expected)
	}
}
