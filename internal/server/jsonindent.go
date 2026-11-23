package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// jsonindent returns v indented two spaces per level when its text is a JSON
// object or array, and its text unchanged otherwise. It returns a string, not
// HTML, so the template still escapes it, and {{linkify (jsonindent .x)}}
// links the URLs in it.
func jsonindent(v any) string {
	s, ok := v.(string)
	if !ok {
		if v == nil {
			return ""
		}
		s = fmt.Sprint(v)
	}
	t := strings.TrimSpace(s)
	if t == "" || (t[0] != '{' && t[0] != '[') {
		return s
	}
	var b bytes.Buffer
	if err := json.Indent(&b, []byte(t), "", "  "); err != nil {
		return s
	}
	return b.String()
}
