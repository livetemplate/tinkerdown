package server

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"unicode/utf8"
)

// urlCandidate finds the text that could be a URL: http:// or https://, in
// any case, with at least one character after it, up to whitespace, a quote,
// an angle bracket or a backslash.
var urlCandidate = regexp.MustCompile(`(?i)https?://[^\s"'<>\\]+`)

// trailingPunct is punctuation that ends a sentence more often than a URL.
const trailingPunct = ".,;:!?”’。，"

// closers maps each closing bracket that can end a URL to its opener.
var closers = map[rune]rune{')': '(', '）': '（', ']': '[', '}': '{'}

// htmlEscaper escapes the same characters as html.EscapeString, writing
// straight into a builder.
var htmlEscaper = strings.NewReplacer(`&`, "&amp;", `'`, "&#39;", `<`, "&lt;", `>`, "&gt;", `"`, "&#34;")

// linkify returns v's text, escaped, with each http:// or https:// URL in it
// turned into a link that opens in a new tab. It escapes all of v's text, even
// a template.HTML, so the result is safe to insert into a page. It runs on
// template values at render time; markdown prose is linked separately, by
// goldmark's own linkify extension.
func linkify(v any) template.HTML {
	var s string
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		s = x
	case template.HTML:
		s = string(x)
	default:
		s = fmt.Sprint(v)
	}

	var b strings.Builder
	b.Grow(len(s) + len(s)/8)
	if !strings.Contains(s, "://") {
		htmlEscaper.WriteString(&b, s)
		return template.HTML(b.String())
	}
	last := 0
	for _, m := range urlCandidate.FindAllStringIndex(s, -1) {
		url := trimURL(s[m[0]:m[1]])
		if strings.HasSuffix(url, "://") {
			continue // only punctuation followed the scheme
		}
		htmlEscaper.WriteString(&b, s[last:m[0]])
		b.WriteString(`<a href="`)
		htmlEscaper.WriteString(&b, url)
		b.WriteString(`" target="_blank" rel="noopener noreferrer">`)
		htmlEscaper.WriteString(&b, url)
		b.WriteString(`</a>`)
		last = m[0] + len(url)
	}
	htmlEscaper.WriteString(&b, s[last:])
	return template.HTML(b.String())
}

// trimURL drops trailing punctuation and any closing bracket that has no
// opener inside the URL, so "(see https://a.com/x)." links only
// https://a.com/x while https://en.wikipedia.org/wiki/Foo_(bar) keeps its ")".
func trimURL(url string) string {
	for {
		r, size := utf8.DecodeLastRuneInString(url)
		open, isCloser := closers[r]
		unmatched := isCloser && strings.Count(url, string(open)) < strings.Count(url, string(r))
		if !unmatched && !strings.ContainsRune(trailingPunct, r) {
			return url
		}
		url = url[:len(url)-size]
	}
}
