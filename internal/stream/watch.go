package stream

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchQuiet is how long a watch waits after the last file event before it
// reruns the hub's list inputs, so a burst, such as a commit writing dozens of
// files, starts one run.
var WatchQuiet = 200 * time.Millisecond

// dirWatch reruns a hub's list inputs (RerunLists) when a file under a folder
// changes. It watches every folder below it, dot folders such as .git
// included, and adds a folder made later. One dirWatch serves every list
// input of a hub, and runs while any of them does.
type dirWatch struct {
	w    *fsnotify.Watcher
	refs int // list inputs using it; guarded by hub.watchMu
}

// watchFiles starts the hub's watch over WatchDir for a list input, or shares
// the one running. stop ends the input's share, and the watch with the last
// one. A watch that failed to start isn't tried again: the walk over a large
// folder is what fails, and it would fail the same way.
func (h *Hub) watchFiles() (stop func(), err error) {
	h.watchMu.Lock()
	defer h.watchMu.Unlock()
	if h.watchErr != nil {
		return nil, h.watchErr
	}
	if h.watch == nil {
		if h.watch, h.watchErr = newDirWatch(h.WatchDir, h.RerunLists); h.watchErr != nil {
			return nil, h.watchErr
		}
	}
	d := h.watch
	d.refs++
	return func() {
		h.watchMu.Lock()
		defer h.watchMu.Unlock()
		if d.refs--; d.refs == 0 {
			d.w.Close()
			h.watch = nil
		}
	}, nil
}

func newDirWatch(dir string, changed func()) (*dirWatch, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	d := &dirWatch{w: w}
	if err := d.addTree(dir); err != nil {
		w.Close()
		return nil, err
	}
	go d.loop(changed)
	return d, nil
}

// addTree watches dir and every folder below it.
func (d *dirWatch) addTree(dir string) error {
	return filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			if path != dir && os.IsNotExist(err) {
				return nil // removed while the walk ran
			}
			return err
		}
		if !e.IsDir() {
			return nil
		}
		if err := d.w.Add(path); err != nil && !(path != dir && os.IsNotExist(err)) {
			return err // a folder removed after the walk found it is skipped
		}
		return nil
	})
}

// loop calls changed WatchQuiet after the last event that changed a file. It
// ends when the watcher is closed.
func (d *dirWatch) loop(changed func()) {
	quiet := time.NewTimer(time.Hour)
	quiet.Stop()
	defer quiet.Stop()
	for {
		select {
		case e, ok := <-d.w.Events:
			if !ok {
				return
			}
			if e.Op == fsnotify.Chmod {
				continue // a mode or time change, not new content
			}
			if e.Has(fsnotify.Create) {
				if info, err := os.Lstat(e.Name); err == nil && info.IsDir() {
					if err := d.addTree(e.Name); err != nil {
						log.Printf("[stream] can't watch new folder %s: %v", e.Name, err)
					}
				}
			}
			quiet.Reset(WatchQuiet)
		case err, ok := <-d.w.Errors:
			if !ok {
				return
			}
			// An error, such as the kernel's event queue overflowing, can
			// hide a change, so the lists run again.
			log.Printf("[stream] file watch: %v", err)
			quiet.Reset(WatchQuiet)
		case <-quiet.C:
			changed()
		}
	}
}
