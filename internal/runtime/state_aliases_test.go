package runtime

import (
	"reflect"
	"testing"
)

// State keys, a state map's keys and a row's keys get a capitalised alias, so
// {{.Data}}, {{.Accepts.Hint}} and {{.Name}} work. A map nested in a row keeps
// only its own keys, so ranging over a log row's .fields shows each once.
func TestProcessStateMapAliases(t *testing.T) {
	got := processStateMap(map[string]interface{}{
		"accepts": map[string]interface{}{"hint": "h"},
		"data": []interface{}{map[string]interface{}{
			"row_cap": 2.0,
			"fields":  map[string]interface{}{"ms": 29.0, "user": map[string]interface{}{"name": "a"}},
			"tags":    []interface{}{map[string]interface{}{"name": "x"}},
		}},
	})

	if got["Data"] == nil || got["Accepts"].(map[string]interface{})["Hint"] != "h" {
		t.Fatalf("state aliases missing: %v", got)
	}
	row := got["data"].([]interface{})[0].(map[string]interface{})
	if row["RowCap"] != 2 || row["Fields"] == nil {
		t.Fatalf("row aliases missing: %v", row)
	}
	wantFields := map[string]interface{}{"ms": 29, "user": map[string]interface{}{"name": "a"}}
	if !reflect.DeepEqual(row["fields"], wantFields) {
		t.Errorf("fields = %v, want %v (no aliases, whole numbers as ints)", row["fields"], wantFields)
	}
	wantTags := []interface{}{map[string]interface{}{"name": "x"}}
	if !reflect.DeepEqual(row["tags"], wantTags) {
		t.Errorf("tags = %v, want %v", row["tags"], wantTags)
	}
}
