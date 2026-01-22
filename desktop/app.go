package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds the application state.
type App struct {
	ctx        context.Context
	server     *server.Server
	httpServer *http.Server
	serverPort int
	currentDir string
	mu         sync.RWMutex
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	a.stopServer()
}

// stopServer stops the current server if running.
func (a *App) stopServer() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.httpServer != nil {
		a.httpServer.Close()
		a.httpServer = nil
	}
	if a.server != nil {
		a.server.StopSchedules()
		a.server.StopWatch()
		a.server = nil
	}
	a.serverPort = 0
}

// OpenFile opens a file dialog to select a markdown file or directory.
func (a *App) OpenFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Markdown File or Directory",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Markdown Files (*.md)",
				Pattern:     "*.md",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return "", err
	}

	if selection == "" {
		return "", nil
	}

	// Check if it's a file or directory
	info, err := os.Stat(selection)
	if err != nil {
		return "", err
	}

	var dir string
	if info.IsDir() {
		dir = selection
	} else {
		dir = filepath.Dir(selection)
	}

	if err := a.loadDirectory(dir); err != nil {
		return "", err
	}

	return dir, nil
}

// OpenDirectory opens a directory dialog.
func (a *App) OpenDirectory() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Tinkerdown Directory",
	})
	if err != nil {
		return "", err
	}

	if selection == "" {
		return "", nil
	}

	if err := a.loadDirectory(selection); err != nil {
		return "", err
	}

	return selection, nil
}

// loadDirectory loads a directory and starts the tinkerdown server.
func (a *App) loadDirectory(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Stop existing server if running
	a.stopServer()

	// Load configuration
	cfg, err := config.LoadFromDir(absDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Enable hot reload for desktop app
	cfg.Features.HotReload = true

	// Create tinkerdown server
	srv := server.NewWithConfig(absDir, cfg)

	// Discover pages
	if err := srv.Discover(); err != nil {
		return fmt.Errorf("failed to discover pages: %w", err)
	}

	// Enable file watching
	if err := srv.EnableWatch(true); err != nil {
		return fmt.Errorf("failed to enable watch mode: %w", err)
	}

	// Start schedules
	if err := srv.StartSchedules(a.ctx); err != nil {
		return fmt.Errorf("failed to start schedules: %w", err)
	}

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to find free port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: server.WithCompression(srv),
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Store references
	a.mu.Lock()
	a.server = srv
	a.httpServer = httpServer
	a.serverPort = port
	a.currentDir = absDir
	a.mu.Unlock()

	// Update window title
	runtime.WindowSetTitle(a.ctx, fmt.Sprintf("Tinkerdown - %s", filepath.Base(absDir)))

	// Navigate to the local server
	serverURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	runtime.EventsEmit(a.ctx, "navigate", serverURL)

	return nil
}

// GetCurrentDirectory returns the currently loaded directory.
func (a *App) GetCurrentDirectory() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentDir
}

// GetServerURL returns the URL of the running server, or empty string if not running.
func (a *App) GetServerURL() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.serverPort == 0 {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d/", a.serverPort)
}

// GetRoutes returns the list of discovered routes.
func (a *App) GetRoutes() []RouteInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.server == nil {
		return nil
	}

	routes := a.server.Routes()
	result := make([]RouteInfo, len(routes))
	for i, r := range routes {
		result[i] = RouteInfo{
			Pattern:  r.Pattern,
			FilePath: r.FilePath,
		}
	}
	return result
}

// RouteInfo represents a route for the frontend.
type RouteInfo struct {
	Pattern  string `json:"pattern"`
	FilePath string `json:"filePath"`
}

// GetHandler returns the HTTP handler.
// If a directory is loaded, it serves tinkerdown content.
// Otherwise, it serves the welcome screen.
func (a *App) GetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mu.RLock()
		srv := a.server
		a.mu.RUnlock()

		if srv != nil {
			// Serve tinkerdown content
			server.WithCompression(srv).ServeHTTP(w, r)
			return
		}

		// Serve welcome screen
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(welcomeHTML))
	})
}

const welcomeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8"/>
    <meta content="width=device-width, initial-scale=1.0" name="viewport"/>
    <title>Tinkerdown Desktop</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #fff;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 2rem;
        }
        .container {
            text-align: center;
            max-width: 600px;
        }
        h1 {
            font-size: 2.5rem;
            margin-bottom: 1rem;
            background: linear-gradient(90deg, #00d4ff, #7c3aed);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }
        p {
            color: #94a3b8;
            font-size: 1.1rem;
            line-height: 1.6;
            margin-bottom: 2rem;
        }
        .actions {
            display: flex;
            gap: 1rem;
            justify-content: center;
            flex-wrap: wrap;
        }
        button {
            background: linear-gradient(135deg, #7c3aed 0%, #00d4ff 100%);
            border: none;
            color: white;
            padding: 0.875rem 1.75rem;
            font-size: 1rem;
            border-radius: 8px;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 20px rgba(124, 58, 237, 0.4);
        }
        button:active {
            transform: translateY(0);
        }
        .keyboard-hint {
            margin-top: 2rem;
            color: #64748b;
            font-size: 0.875rem;
        }
        kbd {
            background: #334155;
            border-radius: 4px;
            padding: 0.25rem 0.5rem;
            font-family: monospace;
        }
        #status {
            margin-top: 1rem;
            font-size: 0.875rem;
            min-height: 1.5em;
        }
        .error { color: #ef4444; }
        .success { color: #22c55e; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Tinkerdown Desktop</h1>
        <p>Open a markdown file or directory to get started with interactive documentation and tutorials.</p>
        <div class="actions">
            <button id="openFile">Open File</button>
            <button id="openDir">Open Directory</button>
        </div>
        <p class="keyboard-hint">
            <kbd>Cmd+O</kbd> / <kbd>Ctrl+O</kbd> to open
        </p>
        <p id="status"></p>
    </div>
    <script>
        function initApp() {
            const statusEl = document.getElementById('status');

            function showStatus(message, type) {
                statusEl.textContent = message;
                statusEl.className = type || '';
            }

            showStatus('Ready', 'success');

            document.getElementById('openFile').addEventListener('click', async function() {
                try {
                    showStatus('Opening file dialog...');
                    const dir = await window.go.main.App.OpenFile();
                    if (dir) {
                        showStatus('Loading ' + dir + '...', 'success');
                    } else {
                        showStatus('Ready', 'success');
                    }
                } catch (err) {
                    showStatus('Error: ' + err, 'error');
                }
            });

            document.getElementById('openDir').addEventListener('click', async function() {
                try {
                    showStatus('Opening directory dialog...');
                    const dir = await window.go.main.App.OpenDirectory();
                    if (dir) {
                        showStatus('Loading ' + dir + '...', 'success');
                        // Check server URL after a short delay
                        setTimeout(async () => {
                            const url = await window.go.main.App.GetServerURL();
                            if (url) {
                                showStatus('Server running at: ' + url + ' - Navigating...', 'success');
                                window.location.href = url;
                            } else {
                                showStatus('Server not started', 'error');
                            }
                        }, 500);
                    } else {
                        showStatus('Ready', 'success');
                    }
                } catch (err) {
                    showStatus('Error: ' + err, 'error');
                }
            });
        }

        // Wait for Wails runtime to be available
        function waitForWails() {
            if (window.go && window.runtime) {
                initApp();
                // Listen for navigate event from Go
                window.runtime.EventsOn('navigate', function(url) {
                    console.log('Navigating to:', url);
                    window.location.href = url;
                });
            } else {
                setTimeout(waitForWails, 50);
            }
        }

        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', waitForWails);
        } else {
            waitForWails();
        }
    </script>
</body>
</html>`
