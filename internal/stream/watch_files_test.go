package stream

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// openFiles counts this process's open file descriptors.
func openFiles(t *testing.T) int {
	t.Helper()
	fds, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Fatal(err)
	}
	return len(fds)
}

// fileTree makes dirs folders of files files each under a temp dir.
func fileTree(t *testing.T, dirs, files int) string {
	t.Helper()
	root := t.TempDir()
	for d := 0; d < dirs; d++ {
		dir := filepath.Join(root, fmt.Sprintf("d%d", d))
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for f := 0; f < files; f++ {
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d", f)), []byte("x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

// waitForOpenFiles waits for the count to fall back to want: a watcher
// closes its event queue from its own goroutine.
func waitForOpenFiles(t *testing.T, want int, what string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for got := openFiles(t); got != want; got = openFiles(t) {
		if time.Now().After(deadline) {
			t.Fatalf("%s: %d files open, want %d as before the watch", what, got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// On macOS a watch holds one open file per watched file and folder. A watch
// that leaked them, as fsnotify 1.9 did on Close, ran the server out of files
// after a few page visits.
func TestWatchStopClosesItsFiles(t *testing.T) {
	h := NewHub()
	defer h.Close()
	h.WatchDir = fileTree(t, 10, 30)
	before := openFiles(t)
	stop, err := h.watchFiles()
	if err != nil {
		t.Fatal(err)
	}
	stop()
	waitForOpenFiles(t, before, "after the watch stopped")
}
