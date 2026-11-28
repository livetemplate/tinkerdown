//go:build darwin

package stream

import (
	"errors"
	"syscall"
	"testing"
)

// On macOS a folder with more files than the open-file limit can't be
// watched. The failed watch must close what it opened, so the list's commands
// can still start and it runs on its timer instead.
func TestFailedWatchClosesItsFiles(t *testing.T) {
	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit) })

	h := NewHub()
	defer h.Close()
	h.WatchDir = fileTree(t, 10, 60)
	before := openFiles(t)
	low := limit
	low.Cur = uint64(before + 200) // fewer than the tree's 611 files and folders
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &low); err != nil {
		t.Fatal(err)
	}
	_, err := h.watchFiles()
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(err, syscall.EMFILE) {
		t.Fatalf("watch over 611 entries with %d files allowed: err %v, want EMFILE", low.Cur, err)
	}
	waitForOpenFiles(t, before, "after the watch failed")
}
