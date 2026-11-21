package security

import (
	"io"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

// Checks for a page from a URL. The page controls its markdown and its lvt
// templates, and so where the operator's log rows end up. The checks read
// markup the way a browser does, with the HTML tokenizer, rather than
// guessing from the raw text: raw-text elements, comments, bogus comments
// and attribute quoting all follow the spec without being copied here.

// forbiddenElements run script, load from another origin, or navigate
// without a click.
var forbiddenElements = []string{
	"script", "meta", "base", "link", "iframe", "frame", "frameset",
	"object", "embed", "applet", "portal",
}

// rawTextElements hold text the tokenizer doesn't parse as markup. A
// template action inside one could close it early and start new markup.
var rawTextElements = []string{
	"style", "textarea", "title", "xmp", "noembed", "noframes", "noscript", "plaintext",
}

// urlAttributes can navigate or load when they hold a javascript: URL.
var urlAttributes = []string{"href", "src", "action", "formaction", "xlink:href", "poster", "data", "background"}

// actionMarker stands in for each template action when a template is
// tokenized. It is letters only, so it can show up in a tag or attribute
// name, which is exactly what the check looks for.
const actionMarker = "tdtemplateaction"

var templateAction = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

// UnsafeHTML reports which kinds of unsafe markup doc contains: forbidden
// elements, on* event handlers, srcdoc, and javascript: or vbscript: URLs.
func UnsafeHTML(doc string) []string {
	return scan(doc, false)
}

// UnsafeTemplate reports UnsafeHTML's findings for an lvt template, and
// any template action outside a text node or an attribute value: in a tag
// or attribute name, a comment, or a raw-text element. Those build markup
// at render time that no check of the source can see. It also refuses the
// trim markers {{- and -}}, which delete the text around an action and so
// change how the rendered markup tokenizes.
func UnsafeTemplate(tmpl string) []string {
	var found []string
	if strings.Contains(tmpl, "{{-") || strings.Contains(tmpl, "-}}") {
		found = append(found, "a template trim marker ({{- or -}})")
	}
	return append(found, scan(templateAction.ReplaceAllString(tmpl, actionMarker), true)...)
}

func scan(doc string, template bool) []string {
	var found []string
	add := func(what string) {
		if !slices.Contains(found, what) {
			found = append(found, what)
		}
	}
	const misplaced = "a template action in a tag name, attribute name, comment or raw-text element (only text and attribute values may hold {{ }})"
	z := html.NewTokenizer(strings.NewReader(doc))
	rawText := false
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() != io.EOF {
				add("markup the HTML tokenizer can't read")
			}
			return found
		}
		tok := z.Token()
		marked := template && strings.Contains(tok.Data, actionMarker)
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			rawText = slices.Contains(rawTextElements, tok.Data) && tt == html.StartTagToken
			if marked {
				add(misplaced)
			}
			if slices.Contains(forbiddenElements, tok.Data) {
				add("a <" + tok.Data + "> element")
			}
			for _, a := range tok.Attr {
				key := strings.ToLower(a.Key)
				switch {
				case template && strings.Contains(key, actionMarker):
					add(misplaced)
				case strings.HasPrefix(key, "on"):
					add("an on* event handler attribute")
				case key == "srcdoc":
					add("an srcdoc attribute")
				case slices.Contains(urlAttributes, key) && scriptURL(a.Val):
					add("a javascript: or vbscript: URL")
				}
			}
		case html.EndTagToken:
			rawText = false
		case html.CommentToken, html.DoctypeToken:
			if marked {
				add(misplaced)
			}
		case html.TextToken:
			if rawText && marked {
				add(misplaced)
			}
		}
	}
}

// scriptURL reports whether a URL attribute value runs script, ignoring the
// whitespace and control characters browsers strip from a scheme.
func scriptURL(v string) bool {
	v = strings.Map(func(r rune) rune {
		if r <= ' ' {
			return -1
		}
		return r
	}, strings.ToLower(v))
	return strings.HasPrefix(v, "javascript:") || strings.HasPrefix(v, "vbscript:")
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
