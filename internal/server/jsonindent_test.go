package server

import "testing"

func TestJSONIndent(t *testing.T) {
	for in, want := range map[string]string{
		`{"a":1,"b":[true,null]}`: "{\n  \"a\": 1,\n  \"b\": [\n    true,\n    null\n  ]\n}",
		` [1,2] `:                 "[\n  1,\n  2\n]",
		`{"a":`:                   `{"a":`,
		`HTTP/1.1 200 OK`:         `HTTP/1.1 200 OK`,
		`"just a string"`:         `"just a string"`,
		`42`:                      `42`,
		``:                        ``,
	} {
		if got := jsonindent(in); got != want {
			t.Errorf("jsonindent(%q) = %q, want %q", in, got, want)
		}
	}
	if got := jsonindent(nil); got != "" {
		t.Errorf("jsonindent(nil) = %q", got)
	}
}
