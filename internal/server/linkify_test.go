package server

import (
	"html/template"
	"strings"
	"testing"
)

// link is the markup linkify makes for one URL, given its escaped form.
func link(escaped string) string {
	return `<a href="` + escaped + `" target="_blank" rel="noopener noreferrer">` + escaped + `</a>`
}

func TestLinkify(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		// text
		{"plain text", "no links here", "no links here"},
		{"one URL", "see https://a.com/x now", "see " + link("https://a.com/x") + " now"},
		{"two URLs", "http://a.com and https://b.com/y", link("http://a.com") + " and " + link("https://b.com/y")},
		{"a number", 42, "42"},
		{"nil", nil, ""},
		{"a map value", map[string]string{"u": "https://a.com"}, "map[u:" + link("https://a.com") + "]"},
		{"a slice value", []string{"x", "http://b.com"}, "[x " + link("http://b.com") + "]"},

		// URL edges
		{"trailing period", "go to https://a.com/x.", "go to " + link("https://a.com/x") + "."},
		{"trailing punctuation", "https://a.com/x?!,;:", link("https://a.com/x") + "?!,;:"},
		{"a query keeps its ?", "https://a.com/x?q=1", link("https://a.com/x?q=1")},
		{"in parentheses", "(see https://a.com/x).", "(see " + link("https://a.com/x") + ")."},
		{"balanced parentheses", "https://en.wikipedia.org/wiki/Foo_(bar)", link("https://en.wikipedia.org/wiki/Foo_(bar)")},
		{"nested parentheses", "https://a.com/((x))", link("https://a.com/((x))")},
		{"in brackets", "[https://a.com/x]", "[" + link("https://a.com/x") + "]"},
		{"mixed brackets inside", "https://a.com/[x](y)", link("https://a.com/[x](y)")},
		{"an apostrophe ends the URL", "https://a.com/it's", link("https://a.com/it") + "&#39;s"},
		{"in braces", `{"u":https://a.com/x}`, "{&#34;u&#34;:" + link("https://a.com/x") + "}"},
		{"nothing after the scheme", "https:// and http://", "https:// and http://"},
		{"only punctuation after the scheme", "https://.", "https://."},
		{"capital scheme", "HTTP://A.COM/X", link("HTTP://A.COM/X")},
		{"non-ASCII quote after", "“https://a.com/x”", "“" + link("https://a.com/x") + "”"},
		{"ideographic full stop after", "https://a.com/x。", link("https://a.com/x") + "。"},
		{"full-width parenthesis after", "（https://a.com/x）", "（" + link("https://a.com/x") + "）"},

		// escaping
		{"script", "<script>alert(1)</script>", "&lt;script&gt;alert(1)&lt;/script&gt;"},
		{"javascript scheme", "javascript:alert(1)", "javascript:alert(1)"},
		{"mixed-case javascript scheme", "JavaScript:alert(1)", "JavaScript:alert(1)"},
		{"quote ends the URL", `https://a.com/x"onmouseover=alert(1)`, link("https://a.com/x") + "&#34;onmouseover=alert(1)"},
		{"javascript after a URL", "https://a.com/javascript:alert(1)", link("https://a.com/javascript:alert(1)")},
		{"& in a query", "https://a.com/?a=1&b=2", link("https://a.com/?a=1&amp;b=2")},
		{"URL ending in &", "https://a.com/?a=1&", link("https://a.com/?a=1&amp;")},
		{"&quot; next to a URL", "&quot;https://a.com&quot;", "&amp;quot;" + link("https://a.com&amp;quot") + ";"},
		{"backslash ends the URL", `https://a.com/x\y`, link("https://a.com/x") + `\y`},
		{"template.HTML is still escaped", template.HTML(`<b>https://a.com</b>`), "&lt;b&gt;" + link("https://a.com") + "&lt;/b&gt;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(linkify(tt.in)); got != tt.want {
				t.Errorf("linkify(%q)\n got %s\nwant %s", tt.in, got, tt.want)
			}
		})
	}
}

// Trimming a URL that ends in many unmatched ")" stays linear.
func TestLinkifyManyClosers(t *testing.T) {
	in := "https://a.com/x" + strings.Repeat(")", 100000)
	if got, want := string(linkify(in)), link("https://a.com/x")+strings.Repeat(")", 100000); got != want {
		t.Fatalf("got %d bytes, want %d", len(got), len(want))
	}
}

func BenchmarkLinkify(b *testing.B) {
	for _, bc := range []struct{ name, line string }{
		{"with a URL", `time=2026-11-22T13:17:53Z level=INFO msg=email body="Sign in: http://localhost:8080/signin/link?t=abc&u=1"`},
		{"no URL", `time=2026-11-22T13:17:53Z level=INFO msg="request done" method=GET path=/api/users status=200 ms=12`},
	} {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				linkify(bc.line)
			}
		})
	}
}
