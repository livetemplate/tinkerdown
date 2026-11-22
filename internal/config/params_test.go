package config

import (
	"slices"
	"strings"
	"testing"
)

func TestCheckTDValue(t *testing.T) {
	for _, ok := range []string{"web", "my app_1", "$(id)", `a"b`, strings.Repeat("x", MaxTDValue)} {
		if err := CheckTDValue("name", ok); err != nil {
			t.Errorf("CheckTDValue(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", strings.Repeat("x", MaxTDValue+1), "-rf", "--output=x", "a\nb", "a\x00b", "a\tb", "a\u0085b"} {
		if err := CheckTDValue("name", bad); err == nil {
			t.Errorf("CheckTDValue(%q) = nil, want an error", bad)
		}
	}
}

func TestTDEnv(t *testing.T) {
	env, err := TDEnv(map[string]interface{}{"name": "web", "id": float64(7), "foo-bar": "skipped", "ok": true})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"TD_id=7", "TD_name=web", "TD_ok=true"}; !slices.Equal(env, want) {
		t.Fatalf("TDEnv = %q, want %q", env, want)
	}
	for _, bad := range []map[string]interface{}{{"name": ""}, {"name": nil}, {"name": "-x"}} {
		if _, err := TDEnv(bad); err == nil {
			t.Errorf("TDEnv(%v) = nil error, want refused", bad)
		}
	}
}
