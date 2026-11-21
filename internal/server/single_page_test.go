package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/livetemplate/tinkerdown/internal/config"
)

func TestServeSinglePage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "logs.md"), []byte("# Logs\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "secret.md"), []byte("# Secret\n"), 0o644)

	for _, remote := range []bool{false, true} {
		srv := NewWithConfig(dir, config.DefaultConfig())
		srv.ServeSinglePage("logs.md", remote)
		if err := srv.Discover(); err != nil {
			t.Fatal(err)
		}
		if routes := srv.Routes(); len(routes) != 1 || routes[0].Pattern != "/" || routes[0].FilePath != "logs.md" {
			t.Fatalf("remote=%v: routes = %+v, want only logs.md at /", remote, routes)
		}

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		body, csp := rec.Body.String(), rec.Header().Get("Content-Security-Policy")
		if !strings.Contains(body, "Logs") {
			t.Fatalf("remote=%v: page not served", remote)
		}
		if !remote {
			if strings.Contains(csp, "nonce-") {
				t.Errorf("a local page got the strict policy: %s", csp)
			}
			continue
		}
		m := regexp.MustCompile(`'nonce-([^']+)'`).FindStringSubmatch(csp)
		if m == nil || strings.Contains(csp, "unsafe-inline' 'unsafe-eval") || !strings.Contains(csp, "img-src 'self' data:") {
			t.Fatalf("strict policy missing for a page from a URL: %s", csp)
		}
		if strings.Contains(body, "<script>") || !strings.Contains(body, `<script nonce="`+m[1]+`">`) {
			t.Error("inline scripts must carry the response's nonce")
		}
		rec2 := httptest.NewRecorder()
		srv.ServeHTTP(rec2, httptest.NewRequest("GET", "/", nil))
		if rec2.Header().Get("Content-Security-Policy") == csp {
			t.Error("the nonce must change per response")
		}
	}
}

// A page's title goes into <title>; it must not open markup there.
func TestRenderedTitleIsEscaped(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.md"), []byte("---\ntitle: \"</title><script>alert(1)</script>\"\n---\n# Hi\n"), 0o644)
	srv := NewWithConfig(dir, config.DefaultConfig())
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if body := rec.Body.String(); strings.Contains(body, "<script>alert(1)") || !strings.Contains(body, "&lt;/title&gt;") {
		t.Fatalf("title not escaped:\n%.600s", body)
	}
}
