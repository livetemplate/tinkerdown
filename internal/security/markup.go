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
	if actionInTag(src) {
		found = append(found, "a template action in a tag name or attribute name (only quoted attribute values may hold {{ }})")
	}
	return found
}

// actionInTag reports whether a template action sits inside an HTML tag but
// outside a quoted attribute value, as in <{{"meta"}} ...> or
// <img {{"onerror"}}=...>. Those build markup at render time that the raw
// text checks above can't see, so a page from a URL may not use them.
func actionInTag(src string) bool {
	inTag, quote := false, byte(0)
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case !inTag:
			if c == '<' && i+1 < len(src) && (isTagStart(src[i+1]) || strings.HasPrefix(src[i+1:], "{{")) {
				inTag = true
			}
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '>':
			inTag = false
		case strings.HasPrefix(src[i:], "{{"):
			return true
		}
	}
	return false
}

func isTagStart(c byte) bool {
	return c == '/' || c == '!' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
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
