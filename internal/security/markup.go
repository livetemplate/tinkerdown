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
	return found
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
