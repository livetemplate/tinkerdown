package stream

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// MaxReadFile caps the bytes a file reader reads in one go: a file, or a
// folder's files together.
const MaxReadFile = 4 * 1024 * 1024

// MaxFolderFiles caps the files a folder source reads.
const MaxFolderFiles = 1000

// textFile is a file a reader read whole: its name and content.
type textFile struct {
	name string // the file's base name
	data []byte
}

// fileStat is what followFiles compares to see that a file changed.
type fileStat struct {
	path string
	size int64
	mod  int64 // modification time, in Unix nanoseconds
}

// followFiles follows path: it reads it whole at start and whenever a file's
// size or modification time changes, and each read's rows, from parse,
// replace the input's. With ext set, path may also be a folder: its files
// ending in ext, in name order, without subfolders or dotfiles. A read that
// parse refuses keeps the last good rows and reports why.
func followFiles(ctx context.Context, b *Buffer, path, ext string, parse func([]textFile) ([]Row, []Private, error)) {
	tick := time.NewTicker(pollInterval)
	defer tick.Stop()
	var seen []fileStat // nil until a read; an empty folder's is empty, not nil
	var last []textFile // the content last read, so a touch changes nothing
	for {
		stats, err := statFiles(path, ext)
		switch {
		case err != nil:
			msg := err.Error()
			if errors.Is(err, fs.ErrNotExist) {
				msg = "waiting for " + path
			}
			b.hub.setErr(ctx, b, msg)
			seen = nil
		case seen == nil || !slices.Equal(stats, seen):
			seen = stats
			if last = readFilesOnce(ctx, b, stats, parse, last); last == nil {
				seen = nil // a read that failed is tried again next poll
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// statFiles lists what followFiles reads at path, refusing more than
// MaxFolderFiles files or MaxReadFile bytes. A folder with no files is an
// empty list, never nil, and not an error.
func statFiles(path, ext string) ([]fileStat, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if info.Size() > MaxReadFile {
			return nil, fmt.Errorf("%s is over %d bytes", path, MaxReadFile)
		}
		return []fileStat{{path, info.Size(), info.ModTime().UnixNano()}}, nil
	}
	if ext == "" {
		return nil, fmt.Errorf("%s is a folder, not a file", path)
	}
	entries, err := os.ReadDir(path) // sorted by name
	if err != nil {
		return nil, err
	}
	stats := []fileStat{}
	var total int64
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.EqualFold(filepath.Ext(name), ext) {
			continue // editors leave dotfiles, such as Emacs's .#q.sql lock
		}
		p := filepath.Join(path, name)
		fi, err := os.Stat(p) // follows a symlink, as reading the file does
		if err != nil || !fi.Mode().IsRegular() {
			continue // a dangling link or a subfolder
		}
		if len(stats) == MaxFolderFiles {
			return nil, fmt.Errorf("%s has over %d %s files", path, MaxFolderFiles, ext)
		}
		if total += fi.Size(); total > MaxReadFile {
			return nil, fmt.Errorf("the %s files in %s are over %d bytes together", ext, path, MaxReadFile)
		}
		stats = append(stats, fileStat{p, fi.Size(), fi.ModTime().UnixNano()})
	}
	return stats, nil
}

// readFilesOnce reads the files and, unless their content is last's,
// replaces the rows with parse's. It returns the content it read, or nil when
// a file couldn't be read.
func readFilesOnce(ctx context.Context, b *Buffer, stats []fileStat, parse func([]textFile) ([]Row, []Private, error), last []textFile) []textFile {
	files := make([]textFile, len(stats))
	var total int
	for i, st := range stats {
		data, err := os.ReadFile(st.path)
		if err != nil {
			b.hub.setErr(ctx, b, err.Error())
			return nil
		}
		// The file can grow between the stat and the read.
		if total += len(data); total > MaxReadFile {
			b.hub.setErr(ctx, b, fmt.Sprintf("%s is over %d bytes", st.path, MaxReadFile))
			return nil
		}
		files[i] = textFile{filepath.Base(st.path), data}
	}
	if last != nil && slices.EqualFunc(files, last, func(a, b textFile) bool {
		return a.name == b.name && bytes.Equal(a.data, b.data)
	}) {
		return last
	}
	rows, privs, err := parse(files)
	if err != nil {
		b.hub.setErr(ctx, b, err.Error())
		return files
	}
	b.hub.replace(ctx, b, "file", rows, privs)
	b.hub.setErr(ctx, b, "")
	return files
}
