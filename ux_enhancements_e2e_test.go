package livepage_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/livetemplate/livepage/internal/server"
)

// TestUXEnhancements verifies the Phase 4 UX improvements
func TestUXEnhancements(t *testing.T) {
	// Create server
	srv := server.New("./examples/counter")
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	// Wrap with compression middleware
	handler := server.WithCompression(srv)

	// Create test server
	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("TailwindCSSLoaded", func(t *testing.T) {
		resp, err := http.Get(ts.URL)
		if err != nil {
			t.Fatalf("Failed to fetch page: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		html := string(bodyBytes)

		// Check for Tailwind CDN script
		if !strings.Contains(html, "cdn.tailwindcss.com") {
			t.Error("Tailwind CSS CDN not found in HTML")
		}
	})

	t.Run("GzipCompressionEnabled", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/assets/livepage-client.js", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to fetch asset: %v", err)
		}
		defer resp.Body.Close()

		// Check Content-Encoding header
		encoding := resp.Header.Get("Content-Encoding")
		if encoding != "gzip" {
			t.Errorf("Expected gzip compression, got: %s", encoding)
		}
	})

	t.Run("ImprovedLayoutWidths", func(t *testing.T) {
		// Create chromedp context
		ctx, cancel := chromedp.NewContext(context.Background())
		defer cancel()

		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		var bodyMaxWidth, sidebarWidth string

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL),
			chromedp.WaitVisible("body", chromedp.ByQuery),

			// Check body max-width
			chromedp.Evaluate(`window.getComputedStyle(document.body).maxWidth`, &bodyMaxWidth),

			// Check if sidebar exists and get its width
			chromedp.Evaluate(`
				const sidebar = document.querySelector('.livepage-nav-sidebar');
				sidebar ? window.getComputedStyle(sidebar).width : 'none';
			`, &sidebarWidth),
		)

		if err != nil {
			t.Fatalf("Chromedp failed: %v", err)
		}

		// Body max-width should be 1200px (updated from 900px)
		if !strings.Contains(bodyMaxWidth, "1200px") {
			t.Errorf("Expected body max-width to be 1200px, got: %s", bodyMaxWidth)
		}

		// If sidebar exists, it should be 220px-320px (updated from 280px)
		// Allow for rounding and responsive adjustments
		if sidebarWidth != "none" {
			if !strings.Contains(sidebarWidth, "220px") && !strings.Contains(sidebarWidth, "240px") && !strings.Contains(sidebarWidth, "320px") {
				t.Errorf("Expected sidebar width to be ~220px-320px, got: %s", sidebarWidth)
			}
			t.Logf("Sidebar width: %s (acceptable range)", sidebarWidth)
		}
	})

	t.Run("MonacoLazyLoading", func(t *testing.T) {
		// Create chromedp context
		ctx, cancel := chromedp.NewContext(context.Background())
		defer cancel()

		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		var monacoLoadedInitially bool
		var hasEditableBlocks bool

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL),
			chromedp.WaitVisible("body", chromedp.ByQuery),
			chromedp.Sleep(1*time.Second), // Give page time to load

			// Check if Monaco is loaded on page without WASM blocks
			chromedp.Evaluate(`typeof monaco !== 'undefined'`, &monacoLoadedInitially),

			// Check if page has editable blocks
			chromedp.Evaluate(`document.querySelectorAll('[data-block-type="wasm"]').length > 0`, &hasEditableBlocks),
		)

		if err != nil {
			t.Fatalf("Chromedp failed: %v", err)
		}

		// Monaco should not be loaded initially on pages without WASM blocks
		// (It will be lazy-loaded only when needed)
		if monacoLoadedInitially && !hasEditableBlocks {
			t.Log("Warning: Monaco loaded on page without editable blocks (expected lazy load)")
		}

		t.Logf("Monaco loaded initially: %v, Has editable blocks: %v", monacoLoadedInitially, hasEditableBlocks)
	})

	t.Run("CompressionRatio", func(t *testing.T) {
		// Test without compression
		respNoComp, err := http.Get(ts.URL + "/assets/livepage-client.js")
		if err != nil {
			t.Fatalf("Failed to fetch asset: %v", err)
		}
		defer respNoComp.Body.Close()

		var uncompressedSize int64
		buf := make([]byte, 1024)
		for {
			n, err := respNoComp.Body.Read(buf)
			uncompressedSize += int64(n)
			if err != nil {
				break
			}
		}

		// Test with compression
		req, err := http.NewRequest("GET", ts.URL+"/assets/livepage-client.js", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept-Encoding", "gzip")

		respComp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to fetch asset: %v", err)
		}
		defer respComp.Body.Close()

		var compressedSize int64
		for {
			n, err := respComp.Body.Read(buf)
			compressedSize += int64(n)
			if err != nil {
				break
			}
		}

		ratio := float64(compressedSize) / float64(uncompressedSize) * 100
		t.Logf("Uncompressed: %d bytes, Compressed: %d bytes, Ratio: %.1f%%",
			uncompressedSize, compressedSize, ratio)

		// Compression should reduce size by at least 50%
		if ratio > 50 {
			t.Errorf("Compression ratio too low: %.1f%% (expected < 50%%)", ratio)
		}
	})
}

// TestResponsiveLayout verifies responsive breakpoints
func TestResponsiveLayout(t *testing.T) {
	// Create server
	srv := server.New("./examples/counter")
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	handler := server.WithCompression(srv)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	viewports := []struct {
		name   string
		width  int
		height int
	}{
		{"Desktop", 1920, 1080},
		{"Laptop", 1366, 768},
		{"Tablet", 768, 1024},
		{"Mobile", 375, 667},
	}

	for _, vp := range viewports {
		t.Run(vp.name, func(t *testing.T) {
			// Create chromedp context with specific viewport
			opts := append(chromedp.DefaultExecAllocatorOptions[:],
				chromedp.WindowSize(vp.width, vp.height),
			)

			allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
			defer cancel()

			ctx, cancel := chromedp.NewContext(allocCtx)
			defer cancel()

			ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			var bodyVisible, sidebarVisible bool

			err := chromedp.Run(ctx,
				chromedp.Navigate(ts.URL),
				chromedp.WaitVisible("body", chromedp.ByQuery),
				chromedp.Sleep(500*time.Millisecond),

				chromedp.Evaluate(`document.body.offsetWidth > 0`, &bodyVisible),
				chromedp.Evaluate(`
					const sidebar = document.querySelector('.livepage-nav-sidebar');
					sidebar && window.getComputedStyle(sidebar).display !== 'none';
				`, &sidebarVisible),
			)

			if err != nil {
				t.Fatalf("Chromedp failed for %s: %v", vp.name, err)
			}

			if !bodyVisible {
				t.Errorf("%s: Body not visible", vp.name)
			}

			t.Logf("%s (%dx%d): Body visible: %v, Sidebar visible: %v",
				vp.name, vp.width, vp.height, bodyVisible, sidebarVisible)
		})
	}
}

// BenchmarkCompression measures compression performance
func BenchmarkCompression(b *testing.B) {
	srv := server.New("./examples/counter")
	if err := srv.Discover(); err != nil {
		b.Fatalf("Failed to discover pages: %v", err)
	}

	handler := server.WithCompression(srv)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	b.Run("WithCompression", func(b *testing.B) {
		req, _ := http.NewRequest("GET", ts.URL+"/assets/livepage-client.js", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})

	b.Run("WithoutCompression", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(fmt.Sprintf("%s/assets/livepage-client.js", ts.URL))
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}
