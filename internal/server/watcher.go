package server

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches for file changes and triggers reload.
type Watcher struct {
	watcher   *fsnotify.Watcher
	rootDir   string
	onReload  func(filePath string) error
	done      chan bool
	debug     bool
}

// NewWatcher creates a new file watcher for the given directory.
func NewWatcher(rootDir string, onReload func(string) error, debug bool) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:  fsWatcher,
		rootDir:  rootDir,
		onReload: onReload,
		done:     make(chan bool),
		debug:    debug,
	}

	// Add root directory
	if err := w.addDirectoryRecursive(rootDir); err != nil {
		fsWatcher.Close()
		return nil, err
	}

	return w, nil
}

// addDirectoryRecursive adds a directory and all its subdirectories to the watcher.
func (w *Watcher) addDirectoryRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories starting with . (hidden dirs like .git)
		// Note: We DO watch directories starting with _ (like _data/) because
		// they may contain external data files referenced by sources
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}

			if err := w.watcher.Add(path); err != nil {
				return err
			}

			if w.debug {
				log.Printf("[Watch] Added directory: %s", path)
			}
		}

		return nil
	})
}

// Start begins watching for file changes.
func (w *Watcher) Start() {
	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}

				// Only respond to write/create events for .md files
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					if filepath.Ext(event.Name) == ".md" {
						relPath, err := filepath.Rel(w.rootDir, event.Name)
						if err != nil {
							relPath = event.Name
						}

						if w.debug {
							log.Printf("[Watch] File changed: %s", relPath)
						}

						if err := w.onReload(relPath); err != nil {
							log.Printf("[Watch] Reload failed for %s: %v", relPath, err)
						}
					}
				}

			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[Watch] Error: %v", err)

			case <-w.done:
				return
			}
		}
	}()
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	close(w.done)
	return w.watcher.Close()
}
