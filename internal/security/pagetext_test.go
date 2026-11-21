package security

import (
	"strings"
	"testing"
)

func TestCleanPageText(t *testing.T) {
	in := "Logs\x1b[2J to\u202e follow\u200b\u2066x\u2069\ufeff" + strings.Repeat("y", 200)
	got := CleanPageText(in)
	if strings.ContainsAny(got, "\x1b\u202e\u200b\u2066\u2069\ufeff") {
		t.Fatalf("control or format characters left in %q", got)
	}
	if !strings.HasPrefix(got, "Logs[2J to followx") || len([]rune(got)) != MaxPageText {
		t.Fatalf("got %q (%d runes)", got, len([]rune(got)))
	}
}
