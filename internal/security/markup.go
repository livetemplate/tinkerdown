package security

import (
	"regexp"
	"slices"
	"strings"
)

// unsafeMarkup lists markup that runs script, loads from another origin or
// navigates without a click. A page from a URL may not contain any of it: its
// template controls where the operator's log rows end up.
var unsafeMarkup = []struct {
	what string
	re   *regexp.Regexp
}{
	{"a <script>, <meta>, <base>, <link>, <iframe>, <frame>, <object>, <embed> or <applet> element",
		regexp.MustCompile(`(?i)<\s*(script|meta|base|link|iframe|frame|frameset|object|embed|applet)\b`)},
	{"an on* event handler attribute", regexp.MustCompile(`(?i)[\s"'/]on[a-z]+\s*=`)},
	{"a javascript: or vbscript: URL", regexp.MustCompile(`(?i)(java|vb)script\s*:`)},
	{"an srcdoc attribute", regexp.MustCompile(`(?i)\ssrcdoc\s*=`)},
}

// UnsafeMarkup reports which kinds of unsafe markup src contains, if any.
// It scans the raw page, so it covers markdown, raw HTML and lvt templates
// alike. It errs toward refusing: prose that quotes "<script>" is refused too.
func UnsafeMarkup(src string) []string {
	var found []string
	for _, u := range unsafeMarkup {
		if u.re.MatchString(src) && !slices.Contains(found, u.what) {
			found = append(found, u.what)
		}
	}
	if strings.Contains(src, "{{-") || strings.Contains(src, "-}}") {
		// Trim markers delete the text around an action, so "< {{- x }}"
		// renders as "<x": markup that isn't in the source.
		found = append(found, "a template trim marker ({{- or -}})")
	}
	if actionOutsideValue(src) {
		found = append(found, "a template action in a tag name, attribute name, unquoted value or comment (only text and quoted attribute values may hold {{ }})")
	}
	return found
}

// actionOutsideValue reports whether a template action sits inside markup
// anywhere but a quoted attribute value: in a tag or attribute name, an
// unquoted value or a comment, as in <{{"meta"}}>, <img {{"onerror"}}=...>
// or <img x"{{"onerror"}}">. Those build markup at render time that the raw
// text checks can't see. It follows the HTML tokenizer: a quote opens a value
// only right after "=", and is part of the name anywhere else.
func actionOutsideValue(src string) bool {
	const (
		text     = iota
		tag      // inside <...>, outside any value
		afterEq  // after "=", before the value starts
		unquoted // inside an unquoted value
		quoted   // inside a quoted value
		comment  // inside <!-- ... -->
	)
	state, quote := text, byte(0)
	for i := 0; i < len(src); i++ {
		c := src[i]
		action := strings.HasPrefix(src[i:], "{{")
		switch state {
		case text:
			switch {
			case strings.HasPrefix(src[i:], "<!--"):
				state, i = comment, i+3
			case c == '<' && i+1 < len(src) && strings.HasPrefix(src[i+1:], "{{"):
				return true
			case c == '<' && i+1 < len(src) && isTagStart(src[i+1]):
				state = tag
			}
		case comment:
			if action {
				return true
			}
			if strings.HasPrefix(src[i:], "-->") {
				state, i = text, i+2
			}
		case tag:
			switch {
			case action:
				return true
			case c == '=':
				state = afterEq
			case c == '>':
				state = text
			}
		case afterEq:
			switch {
			case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f':
			case c == '"' || c == '\'':
				state, quote = quoted, c
			case c == '>':
				state = text
			case action:
				return true
			default:
				state = unquoted
			}
		case unquoted:
			switch {
			case action:
				return true
			case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f':
				state = tag
			case c == '>':
				state = text
			}
		case quoted:
			if c == quote {
				state = tag
			}
		}
	}
	return false
}

func isTagStart(c byte) bool {
	return c == '/' || c == '!' || c == '?' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// StrictCSP is the policy for a page from a URL: no inline script except
// tinkerdown's own (by nonce), no eval, and nothing loaded from or sent to
// another origin.
func StrictCSP(nonce string) string {
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'nonce-" + nonce + "'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"media-src 'self'",
		"font-src 'self' data:",
		"connect-src 'self'",
		"form-action 'self'",
		"base-uri 'none'",
		"frame-src 'none'",
		"object-src 'none'",
		"frame-ancestors 'none'",
	}, "; ")
}
