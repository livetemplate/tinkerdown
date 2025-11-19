package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/livetemplate/livepage"
	"github.com/livetemplate/livepage/internal/assets"
)

// Route represents a discovered page route.
type Route struct {
	Pattern  string         // URL pattern (e.g., "/counter")
	FilePath string         // Relative file path (e.g., "counter.md")
	Page     *livepage.Page // Parsed page
}

// Server is the livepage development server.
type Server struct {
	rootDir     string
	routes      []*Route
	mu          sync.RWMutex
	connections map[*websocket.Conn]bool // Track connected WebSocket clients
	connMu      sync.RWMutex              // Separate mutex for connections
	watcher     *Watcher                  // File watcher for live reload
}

// New creates a new server for the given root directory.
func New(rootDir string) *Server {
	return &Server{
		rootDir:     rootDir,
		routes:      make([]*Route, 0),
		connections: make(map[*websocket.Conn]bool),
	}
}

// Discover scans the directory for .md files and creates routes.
func (s *Server) Discover() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.routes = make([]*Route, 0)

	err := filepath.WalkDir(s.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			// Skip directories starting with _ or .
			name := d.Name()
			if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .md files
		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(s.rootDir, path)
		if err != nil {
			return err
		}

		// Skip files in _ directories
		if strings.Contains(relPath, "/_") || strings.HasPrefix(relPath, "_") {
			return nil
		}

		// Generate route pattern
		pattern := mdToPattern(relPath)

		// Parse the page
		page, err := livepage.ParseFile(path)
		if err != nil {
			log.Printf("Warning: Failed to parse %s: %v", relPath, err)
			return nil // Continue with other files
		}

		route := &Route{
			Pattern:  pattern,
			FilePath: relPath,
			Page:     page,
		}

		s.routes = append(s.routes, route)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort routes (index routes first)
	sortRoutes(s.routes)

	return nil
}

// Routes returns the discovered routes.
func (s *Server) Routes() []*Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.routes
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve WebSocket endpoint
	if r.URL.Path == "/ws" {
		s.serveWebSocket(w, r)
		return
	}

	// Serve assets
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		s.serveAsset(w, r)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find matching route
	for _, route := range s.routes {
		if route.Pattern == r.URL.Path {
			s.servePage(w, r, route)
			return
		}
	}

	// No route found
	http.NotFound(w, r)
}

// serveWebSocket handles WebSocket connections for interactive blocks.
func (s *Server) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get the page from query parameter (for now, use first route)
	// TODO: Support multiple pages via query param or path
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.routes) == 0 {
		http.Error(w, "No pages available", http.StatusNotFound)
		return
	}

	// Use the first route's page for now
	route := s.routes[0]

	// Create WebSocket handler for this page
	wsHandler := NewWebSocketHandler(route.Page, s, true) // debug=true
	wsHandler.ServeHTTP(w, r)
}

// serveAsset serves embedded client assets.
func (s *Server) serveAsset(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/assets/")

	// Serve client JS
	if path == "livepage-client.js" {
		js, err := assets.GetClientJS()
		if err != nil {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(js)
		return
	}

	// Serve client CSS
	if path == "livepage-client.css" {
		css, err := assets.GetClientCSS()
		if err != nil {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/css")
		w.Write(css)
		return
	}

	http.NotFound(w, r)
}

// servePage serves a page.
func (s *Server) servePage(w http.ResponseWriter, r *http.Request, route *Route) {
	// For now, just serve the static HTML
	// TODO: Add WebSocket support for interactivity
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := s.renderPage(route.Page)
	w.Write([]byte(html))
}

// renderPage renders a page to HTML.
func (s *Server) renderPage(page *livepage.Page) string {
	// Render code blocks with metadata for client discovery
	content := s.renderContent(page)

	// Basic HTML wrapper with the static content
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="livepage-ws-url" content="ws://localhost:8080/ws">
    <meta name="livepage-debug" content="true">
    <title>%s</title>
    <!-- Tailwind CSS Play CDN -->
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="stylesheet" href="/assets/livepage-client.css">
    <style>
        /* Theme Variables */
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: linear-gradient(135deg, #f5f7fa 0%%, #e8ecf1 100%%);
            --text-primary: #333;
            --text-secondary: #555;
            --text-heading: #2c3e50;
            --border-color: #e1e4e8;
            --code-bg: #f4f4f4;
            --code-border: #e1e4e8;
            --pre-bg: #282c34;
            --pre-text: #abb2bf;
            --button-bg: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            --button-shadow: rgba(102, 126, 234, 0.3);
            --card-bg: #ffffff;
            --card-border: rgba(0,0,0,0.06);
            --card-shadow: rgba(0,0,0,0.08);
            --accent: #0066cc;
        }

        [data-theme="dark"] {
            --bg-primary: #1a1a1a;
            --bg-secondary: linear-gradient(135deg, #1a1a1a 0%%, #2d2d2d 100%%);
            --text-primary: #e0e0e0;
            --text-secondary: #b0b0b0;
            --text-heading: #f0f0f0;
            --border-color: #404040;
            --code-bg: #2d2d2d;
            --code-border: #404040;
            --pre-bg: #1e1e1e;
            --pre-text: #d4d4d4;
            --button-bg: linear-gradient(135deg, #4da6ff 0%%, #357abd 100%%);
            --button-shadow: rgba(77, 166, 255, 0.3);
            --card-bg: #242424;
            --card-border: rgba(255,255,255,0.1);
            --card-shadow: rgba(0,0,0,0.3);
            --accent: #4da6ff;
        }

        /* Theme transition */
        * {
            transition: background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease;
        }

        /* Base styles */
        * {
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem 1.5rem;
            color: var(--text-primary);
            background: var(--bg-secondary);
            min-height: 100vh;
        }

        /* Content wrapper for readable line lengths */
        .content-wrapper {
            max-width: 800px;
            margin: 0 auto;
        }

        /* Typography */
        h1, h2, h3 {
            color: var(--text-heading);
            font-weight: 600;
            letter-spacing: -0.02em;
        }

        h1 {
            font-size: 2.5rem;
            margin-bottom: 1.5rem;
        }

        h2 {
            font-size: 1.75rem;
            margin-top: 2.5rem;
            margin-bottom: 1rem;
        }

        p {
            margin-bottom: 1.25rem;
            color: var(--text-secondary);
        }

        /* Code blocks */
        code {
            background: var(--code-bg);
            padding: 0.2rem 0.4rem;
            border-radius: 4px;
            font-size: 0.9em;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            border: 1px solid var(--code-border);
            color: var(--text-primary);
        }

        pre {
            background: var(--pre-bg);
            color: var(--pre-text);
            padding: 1.25rem 1rem;
            border-radius: 8px;
            overflow-x: auto;
            margin: 1.5rem calc((800px - 100%%) / 2 * -1);
            max-width: 1000px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
            border: 1px solid var(--border-color);
        }

        pre code {
            background: none;
            border: none;
            padding: 0;
            color: inherit;
        }

        /* Interactive blocks */
        .livepage-wasm-block,
        .livepage-interactive-block {
            margin: 2rem calc((800px - 100%%) / 2 * -1);
            max-width: 1000px;
            padding: 1.5rem;
            background: var(--card-bg);
            border-radius: 16px;
            box-shadow: 0 4px 16px var(--card-shadow);
            border: 1px solid var(--card-border);
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }

        .livepage-wasm-block:hover,
        .livepage-interactive-block:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px var(--card-shadow);
        }

        /* Buttons */
        button {
            background: var(--button-bg);
            color: white;
            border: none;
            padding: 0.75rem 1.5rem;
            border-radius: 8px;
            font-size: 1rem;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s ease;
            box-shadow: 0 2px 8px var(--button-shadow);
            margin: 0.25rem;
            font-family: inherit;
        }

        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px var(--button-shadow);
        }

        button:active {
            transform: translateY(0);
            box-shadow: 0 1px 4px var(--button-shadow);
        }

        /* Counter display */
        .counter-display {
            font-size: 3rem;
            font-weight: 700;
            text-align: center;
            margin: 2rem 0;
            padding: 1.5rem;
            background: linear-gradient(135deg, #f5f7fa 0%%, #ffffff 100%%);
            border-radius: 16px;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            border: 2px solid #e1e4e8;
        }

        .counter-display.positive {
            color: #10b981;
            border-color: #10b981;
            box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
        }

        .counter-display.negative {
            color: #ef4444;
            border-color: #ef4444;
            box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
        }

        .counter-display.zero {
            color: #6b7280;
            border-color: #d1d5db;
        }

        /* Number transition animation */
        @keyframes numberPulse {
            0%%, 100%% { transform: scale(1); }
            50%% { transform: scale(1.1); }
        }

        .counter-display.changed {
            animation: numberPulse 0.3s ease;
        }

        /* Button groups */
        .button-group {
            display: flex;
            justify-content: center;
            flex-wrap: wrap;
            gap: 0.5rem;
            margin: 1rem 0;
        }

        /* Responsive design */
        @media (max-width: 768px) {
            body {
                padding: 1rem;
            }

            h1 {
                font-size: 2rem;
            }

            h2 {
                font-size: 1.5rem;
            }

            .livepage-wasm-block,
            .livepage-interactive-block {
                padding: 1.5rem;
                border-radius: 12px;
            }

            .counter-display {
                font-size: 2.5rem;
                padding: 1.5rem;
            }

            button {
                padding: 0.625rem 1.25rem;
                font-size: 0.9rem;
            }
        }

        @media (max-width: 480px) {
            body {
                padding: 0.75rem;
            }

            h1 {
                font-size: 1.75rem;
            }

            .livepage-wasm-block,
            .livepage-interactive-block {
                padding: 1rem;
                margin: 1rem 0;
            }

            .counter-display {
                font-size: 2rem;
                padding: 1rem;
            }

            button {
                width: 100%%;
                margin: 0.25rem 0;
            }

            .button-group {
                flex-direction: column;
            }
        }

        /* Theme Toggle */
        .theme-toggle {
            position: fixed;
            top: 1rem;
            right: 1rem;
            z-index: 1000;
            display: flex;
            gap: 0.5rem;
            background: var(--card-bg);
            padding: 0.5rem;
            border-radius: 8px;
            box-shadow: 0 2px 8px var(--card-shadow);
            border: 1px solid var(--card-border);
        }

        .theme-toggle button {
            background: transparent;
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            padding: 0.5rem;
            margin: 0;
            border-radius: 6px;
            font-size: 1.2rem;
            min-width: 2.5rem;
            box-shadow: none;
        }

        .theme-toggle button:hover {
            background: var(--code-bg);
            transform: none;
            box-shadow: none;
        }

        .theme-toggle button.active {
            background: var(--accent);
            color: white;
            border-color: var(--accent);
        }

        .theme-toggle button:active {
            transform: scale(0.95);
        }


        /* Tutorial Navigation - Sidebar TOC */
        .livepage-nav-sidebar {
            position: fixed;
            left: 0;
            top: 0;
            bottom: 0;
            width: 180px;
            background: var(--card-bg);
            border-right: 1px solid var(--card-border);
            box-shadow: 2px 0 8px var(--card-shadow);
            z-index: 900;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }

        .nav-sidebar-header {
            padding: 1.5rem;
            border-bottom: 1px solid var(--border-color);
            background: var(--bg-secondary);
        }

        .nav-sidebar-header h3 {
            margin: 0;
            font-size: 1rem;
            font-weight: 600;
            color: var(--text-heading);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .nav-sidebar-steps {
            flex: 1;
            overflow-y: auto;
            padding: 0;
            margin: 0;
            list-style: none;
        }

        .nav-step {
            border-bottom: 1px solid var(--border-color);
        }

        .nav-step a {
            display: flex;
            align-items: center;
            gap: 1rem;
            padding: 1rem 1.5rem;
            text-decoration: none;
            color: var(--text-secondary);
            transition: all 0.2s ease;
        }

        .nav-step:hover a {
            background: var(--code-bg);
            color: var(--text-primary);
        }

        .nav-step.active a {
            background: var(--accent);
            color: white;
            font-weight: 500;
        }

        .step-number {
            display: flex;
            align-items: center;
            justify-content: center;
            width: 28px;
            height: 28px;
            border-radius: 50%%;
            background: var(--code-bg);
            color: var(--text-primary);
            font-size: 0.875rem;
            font-weight: 600;
            flex-shrink: 0;
        }

        .nav-step.active .step-number {
            background: rgba(255, 255, 255, 0.2);
            color: white;
        }

        .step-title {
            flex: 1;
            font-size: 0.9rem;
            line-height: 1.4;
        }

        /* Tutorial Navigation - Bottom Bar */
        .livepage-nav-bottom {
            position: fixed;
            bottom: 0;
            left: 180px;
            right: 0;
            height: 60px;
            background: var(--card-bg);
            border-top: 1px solid var(--card-border);
            box-shadow: 0 -2px 8px var(--card-shadow);
            z-index: 900;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 2rem;
            gap: 1rem;
        }

        .nav-btn {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.75rem 1.5rem;
            background: var(--accent);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 1rem;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s ease;
            box-shadow: 0 2px 8px rgba(0, 102, 204, 0.3);
        }

        .nav-btn:hover:not(:disabled) {
            background: #0052a3;
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(0, 102, 204, 0.4);
        }

        .nav-btn:active:not(:disabled) {
            transform: translateY(0);
        }

        .nav-btn:disabled {
            background: var(--code-bg);
            color: var(--text-secondary);
            cursor: not-allowed;
            box-shadow: none;
        }

        .nav-arrow {
            font-size: 1.2rem;
            line-height: 1;
        }

        .nav-progress {
            font-size: 0.9rem;
            color: var(--text-secondary);
            font-weight: 500;
        }

        .current-step,
        .total-steps {
            color: var(--accent);
            font-weight: 600;
        }

        /* Adjust main content to make room for navigation */
        body:has(.livepage-nav-sidebar) {
            margin-left: 180px;
            margin-bottom: 60px;
        }

        /* Responsive Navigation */
        @media (max-width: 1024px) {
            .livepage-nav-sidebar {
                width: 200px;
            }

            .livepage-nav-bottom {
                left: 200px;
            }

            body:has(.livepage-nav-sidebar) {
                margin-left: 200px;
            }
        }

        @media (max-width: 768px) {
            /* Hide sidebar on mobile, show hamburger menu */
            .livepage-nav-sidebar {
                transform: translateX(-100%%);
                transition: transform 0.3s ease;
            }

            .livepage-nav-sidebar.open {
                transform: translateX(0);
            }

            .livepage-nav-bottom {
                left: 0;
                padding: 0 1rem;
            }

            body:has(.livepage-nav-sidebar) {
                margin-left: 0;
            }

            .nav-btn {
                padding: 0.5rem 1rem;
                font-size: 0.9rem;
            }

            .nav-label {
                display: none;
            }

            .nav-arrow {
                font-size: 1.5rem;
            }

            .nav-progress {
                font-size: 0.85rem;
            }
        }

        @media (max-width: 480px) {
            .livepage-nav-bottom {
                height: 60px;
                padding: 0 0.75rem;
            }

            body:has(.livepage-nav-sidebar) {
                margin-bottom: 60px;
            }

            .nav-btn {
                padding: 0.5rem 0.75rem;
                min-width: 40px;
            }

            .nav-progress {
                font-size: 0.75rem;
            }
        }

        /* Counter Variations Styling */
        .counter-container {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 12px;
            padding: 1.5rem;
            margin: 1rem 0;
        }

        .counter-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 1rem;
        }

        .bounds-label {
            font-size: 0.875rem;
            color: var(--text-secondary);
            font-weight: 500;
        }

        .counter-display.at-max {
            background: #fef3c7;
            border-color: #f59e0b;
            color: #78350f;
        }

        .counter-display.at-min {
            background: #fee2e2;
            border-color: #ef4444;
            color: #7f1d1d;
        }

        .counter-display.in-range {
            background: var(--accent);
            color: white;
        }

        .bounds-bar {
            width: 100%%;
            height: 6px;
            background: var(--code-bg);
            border-radius: 3px;
            margin-top: 1rem;
            overflow: hidden;
        }

        .bounds-progress {
            height: 100%%;
            background: var(--accent);
            transition: width 0.3s ease;
        }

        /* Step Counter */
        .step-buttons {
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
        }

        .button-row {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .row-label {
            font-weight: 500;
            min-width: 80px;
            color: var(--text-secondary);
        }

        .step-btn {
            flex: 1;
            padding: 0.5rem 1rem;
            font-size: 0.9rem;
        }

        .reset-btn {
            width: 100%%;
            margin-top: 0.5rem;
        }

        /* Dual Counter */
        .dual-counter-container {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 1.5rem;
            margin: 1.5rem 0;
        }

        .dual-counter-item {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 12px;
            padding: 1.5rem;
        }

        .counter-label {
            font-size: 0.875rem;
            font-weight: 600;
            color: var(--text-secondary);
            margin-bottom: 1rem;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        @media (max-width: 768px) {
            .dual-counter-container {
                grid-template-columns: 1fr;
            }
        }

        /* Shopping Cart Product */
        .product-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 12px;
            padding: 1.5rem;
            margin: 1.5rem 0;
            box-shadow: 0 2px 8px var(--card-shadow);
        }

        .product-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--border-color);
        }

        .product-header h4 {
            margin: 0;
            color: var(--text-heading);
        }

        .product-price {
            font-size: 1.25rem;
            font-weight: 600;
            color: var(--accent);
        }

        .quantity-selector {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin: 1rem 0;
            padding: 0.75rem;
            background: var(--code-bg);
            border-radius: 8px;
        }

        .quantity-label {
            font-weight: 500;
            color: var(--text-secondary);
        }

        .quantity-controls {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .qty-btn {
            width: 36px;
            height: 36px;
            padding: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.25rem;
            border-radius: 50%%;
        }

        .quantity-display {
            min-width: 40px;
            text-align: center;
            font-weight: 600;
            font-size: 1.125rem;
        }

        .product-total {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 1rem;
            margin: 1rem 0;
            background: var(--bg-secondary);
            border-radius: 8px;
        }

        .total-label {
            font-weight: 500;
            color: var(--text-secondary);
        }

        .total-amount {
            font-size: 1.5rem;
            font-weight: 700;
            color: var(--accent);
        }

        .remove-btn {
            background: transparent;
            border: 1px solid #ef4444;
            color: #ef4444;
            box-shadow: none;
        }

        .remove-btn:hover {
            background: #ef4444;
            color: white;
        }

        .removed-message {
            text-align: center;
            padding: 2rem;
            color: var(--text-secondary);
            font-style: italic;
        }

        /* Prism.js syntax highlighting overrides */
        pre[class*="language-"] {
            margin: 1.5rem 0;
            padding: 1rem;
            border-radius: 8px;
            overflow-x: auto;
        }

        code[class*="language-"],
        pre[class*="language-"] {
            font-family: 'Consolas', 'Monaco', 'Andale Mono', 'Ubuntu Mono', monospace;
            font-size: 0.9rem;
            line-height: 1.5;
        }

        /* Ensure code blocks are styled properly */
        :not(pre) > code[class*="language-"] {
            padding: 0.1em 0.3em;
            border-radius: 0.3em;
        }
    </style>

    <!-- Prism.js for syntax highlighting -->
    <link href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism-tomorrow.min.css" rel="stylesheet" />
</head>
<body>
    <!-- Theme Toggle -->
    <div class="theme-toggle">
        <button id="theme-light" title="Light theme" aria-label="Light theme">☀️</button>
        <button id="theme-dark" title="Dark theme" aria-label="Dark theme">🌙</button>
        <button id="theme-auto" title="Auto theme (system preference)" aria-label="Auto theme">🌓</button>
    </div>

    %s

    <script>
        // Theme management
        (function() {
            const STORAGE_KEY = 'livepage-theme';
            const html = document.documentElement;

            // Get current theme from localStorage or default to 'auto'
            function getStoredTheme() {
                return localStorage.getItem(STORAGE_KEY) || 'auto';
            }

            // Get system preference
            function getSystemTheme() {
                return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
            }

            // Apply theme to HTML element
            function applyTheme(theme) {
                const effectiveTheme = theme === 'auto' ? getSystemTheme() : theme;
                html.setAttribute('data-theme', effectiveTheme);

                // Update button states
                document.querySelectorAll('.theme-toggle button').forEach(btn => {
                    btn.classList.remove('active');
                });
                const activeBtn = document.getElementById('theme-' + theme);
                if (activeBtn) {
                    activeBtn.classList.add('active');
                }
            }

            // Set and save theme
            function setTheme(theme) {
                localStorage.setItem(STORAGE_KEY, theme);
                applyTheme(theme);
            }

            // Initialize theme on page load (before paint to avoid flash)
            const storedTheme = getStoredTheme();
            applyTheme(storedTheme);

            // Listen for system theme changes when in auto mode
            window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
                if (getStoredTheme() === 'auto') {
                    applyTheme('auto');
                }
            });

            // Add click handlers after DOM is ready
            window.addEventListener('DOMContentLoaded', () => {
                document.getElementById('theme-light').addEventListener('click', () => setTheme('light'));
                document.getElementById('theme-dark').addEventListener('click', () => setTheme('dark'));
                document.getElementById('theme-auto').addEventListener('click', () => setTheme('auto'));

                // Keyboard shortcut: Ctrl+Shift+D
                document.addEventListener('keydown', (e) => {
                    if (e.ctrlKey && e.shiftKey && e.key === 'D') {
                        e.preventDefault();
                        const current = getStoredTheme();
                        const next = current === 'light' ? 'dark' : current === 'dark' ? 'auto' : 'light';
                        setTheme(next);
                    }
                });
            });
        })();
    </script>

    <script src="/assets/livepage-client.js"></script>

    <!-- Prism.js for syntax highlighting -->
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/prism.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-go.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-javascript.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-jsx.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-markup.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-css.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-yaml.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-json.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-bash.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-markdown.min.js"></script>
    <script>
        // Highlight all code blocks on page load
        document.addEventListener('DOMContentLoaded', function() {
            Prism.highlightAll();
        });
    </script>

    <!-- Mermaid.js for diagrams -->
    <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
    <script>
        // Initialize Mermaid for diagram rendering
        mermaid.initialize({
            startOnLoad: false, // We'll trigger manually after conversion
            theme: document.documentElement.classList.contains('theme-dark') ? 'dark' : 'default',
            flowchart: {
                useMaxWidth: true,
                htmlLabels: true,
                curve: 'basis'
            },
            sequence: {
                diagramMarginX: 50,
                diagramMarginY: 10,
                actorMargin: 50,
                width: 150,
                height: 65,
                boxMargin: 10,
                boxTextMargin: 5,
                noteMargin: 10,
                messageMargin: 35,
                mirrorActors: true,
                useMaxWidth: true
            }
        });

        // Convert mermaid code blocks to rendered diagrams
        document.addEventListener('DOMContentLoaded', function() {
            // Find all code blocks with language-mermaid class
            const mermaidBlocks = document.querySelectorAll('code.language-mermaid');

            mermaidBlocks.forEach(function(codeBlock) {
                // Get the mermaid code
                const code = codeBlock.textContent;

                // Create a new div for the rendered diagram
                const mermaidDiv = document.createElement('div');
                mermaidDiv.className = 'mermaid';
                mermaidDiv.textContent = code;

                // Replace the pre>code structure with just the mermaid div
                const preBlock = codeBlock.parentElement;
                preBlock.parentNode.replaceChild(mermaidDiv, preBlock);
            });

            // Now render all mermaid diagrams
            mermaid.run();
        });

        // Re-initialize Mermaid when theme changes
        document.addEventListener('themeChanged', function(e) {
            mermaid.initialize({
                theme: e.detail.theme === 'dark' ? 'dark' : 'default'
            });
            // Re-render all diagrams
            mermaid.run();
        });
    </script>
</body>
</html>`, page.Title, content)

	return html
}

// renderContent renders the page content with code blocks
func (s *Server) renderContent(page *livepage.Page) string {
	content := page.StaticHTML

	// TODO: Enhance markdown parser to add data attributes to code blocks
	// For now, the client will need to discover blocks by parsing the HTML
	// In Phase 4.5, we'll improve this to inject proper data attributes during parsing

	// Wrap content in a readable-width container for better typography
	// Navigation elements (sidebar, bottom nav) are outside this wrapper
	wrappedContent := fmt.Sprintf(`<div class="content-wrapper">%s</div>`, content)

	return wrappedContent
}

// mdToPattern converts a markdown file path to a URL pattern.
// Examples:
//   - "index.md" → "/"
//   - "counter.md" → "/counter"
//   - "tutorials/intro.md" → "/tutorials/intro"
//   - "tutorials/index.md" → "/tutorials/"
func mdToPattern(relPath string) string {
	// Remove .md extension
	path := strings.TrimSuffix(relPath, ".md")

	// Convert to URL path
	path = filepath.ToSlash(path)

	// Handle index files
	if path == "index" {
		return "/"
	}
	if strings.HasSuffix(path, "/index") {
		return "/" + strings.TrimSuffix(path, "index")
	}

	return "/" + path
}

// sortRoutes sorts routes with index routes first.
func sortRoutes(routes []*Route) {
	// Simple sort: / first, then /foo/, then /foo
	// This is a basic implementation; could be more sophisticated
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if shouldSwap(routes[i], routes[j]) {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}
}

func shouldSwap(a, b *Route) bool {
	// Root path comes first
	if a.Pattern == "/" {
		return false
	}
	if b.Pattern == "/" {
		return true
	}

	// Directory index paths come before other paths
	aIsIndex := strings.HasSuffix(a.Pattern, "/")
	bIsIndex := strings.HasSuffix(b.Pattern, "/")

	if aIsIndex && !bIsIndex {
		return false
	}
	if !aIsIndex && bIsIndex {
		return true
	}

	// Alphabetical otherwise
	return a.Pattern > b.Pattern
}

// RegisterConnection adds a WebSocket connection to the tracked connections.
func (s *Server) RegisterConnection(conn *websocket.Conn) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	s.connections[conn] = true
	log.Printf("[Server] WebSocket connection registered: %d active connections", len(s.connections))
}

// UnregisterConnection removes a WebSocket connection from tracked connections.
func (s *Server) UnregisterConnection(conn *websocket.Conn) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	delete(s.connections, conn)
	log.Printf("[Server] WebSocket connection unregistered: %d active connections", len(s.connections))
}

// BroadcastReload sends a reload message to all connected WebSocket clients.
func (s *Server) BroadcastReload(filePath string) {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	if len(s.connections) == 0 {
		return
	}

	msg := map[string]interface{}{
		"action":   "reload",
		"filePath": filePath,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[Server] Failed to marshal reload message: %v", err)
		return
	}

	log.Printf("[Server] Broadcasting reload for %s to %d connections", filePath, len(s.connections))

	for conn := range s.connections {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[Server] Failed to send reload to connection: %v", err)
		}
	}
}

// EnableWatch enables file watching for live reload.
func (s *Server) EnableWatch(debug bool) error {
	watcher, err := NewWatcher(s.rootDir, func(filePath string) error {
		log.Printf("[Watch] File changed: %s", filePath)

		// Re-discover pages
		if err := s.Discover(); err != nil {
			return fmt.Errorf("failed to re-discover pages: %w", err)
		}

		// Broadcast reload to all connected clients
		s.BroadcastReload(filePath)

		return nil
	}, debug)

	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	s.watcher = watcher
	s.watcher.Start()

	log.Printf("[Watch] File watcher started for %s", s.rootDir)
	return nil
}

// StopWatch stops the file watcher if it's running.
func (s *Server) StopWatch() error {
	if s.watcher != nil {
		return s.watcher.Stop()
	}
	return nil
}
