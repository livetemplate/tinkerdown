package security

import (
	"strings"
	"unicode"
)

// MaxPageText is how much page-provided text reaches the terminal.
const MaxPageText = 120

// CleanPageText makes text from a page safe to print in the operator's
// terminal: control characters (escape sequences that could clear the
// screen, fake a prompt or write the clipboard) are dropped, and the text is
// cut to MaxPageText runes. Callers still label it as the page's words.
func CleanPageText(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return -1
		}
		return r
	}, s)
	if r := []rune(s); len(r) > MaxPageText {
		s = string(r[:MaxPageText-1]) + "…"
	}
	return s
}
