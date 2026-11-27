package stream

import (
	"context"
	"io"
	"os"
	"time"
)

// pollInterval is how often the tailer checks the file for new data.
var pollInterval = 250 * time.Millisecond

const readChunk = 64 * 1024

// tailFile follows spec.File until ctx ends. It starts with the last
// spec.Backlog lines, then publishes each appended line. It reopens the file
// when it is replaced (rotation) and rereads it from the start when it shrinks
// (truncation).
func tailFile(ctx context.Context, spec InputSpec, b *Buffer) {
	t := &tailer{ctx: ctx, spec: spec, b: b, buf: make([]byte, readChunk)}
	t.split.max = spec.LineMax
	t.split.emit = func(line string, truncated bool) {
		t.batch = append(t.batch, spec.row(line, truncated, b.hub.now()))
	}
	defer t.close()
	tick := time.NewTicker(pollInterval)
	defer tick.Stop()
	for first := true; ; first = false {
		t.poll(first)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

type tailer struct {
	ctx     context.Context // the run, for publish
	spec    InputSpec
	b       *Buffer
	f       *os.File
	info    os.FileInfo
	offset  int64
	split   lineSplitter // holds a last line still waiting for its newline
	buf     []byte       // reused read buffer
	batch   []Row
	missing bool // a "waiting for file" row was published
}

func (t *tailer) close() {
	if t.f != nil {
		t.f.Close()
	}
}

func (t *tailer) reset(offset int64) {
	t.offset = offset
	t.split.pending, t.split.discarding = t.split.pending[:0], false
}

func (t *tailer) poll(first bool) {
	info, err := os.Stat(t.spec.File)
	if err != nil {
		if !t.missing {
			t.missing = true
			t.b.system(t.ctx, "warn", "waiting for "+t.spec.File)
		}
		return
	}
	t.missing = false

	switch {
	case t.f == nil || !os.SameFile(t.info, info):
		rotated := t.f != nil
		t.close()
		f, err := os.Open(t.spec.File)
		if err != nil {
			t.f = nil
			return
		}
		t.f = f
		t.reset(0)
		if rotated {
			t.b.system(t.ctx, "info", "file rotated, reading the new file")
		} else if first {
			t.reset(backlogOffset(f, info.Size(), t.spec.Backlog))
		}
	case info.Size() < t.offset:
		t.reset(0)
		t.b.system(t.ctx, "info", "file truncated, reading from the start")
	}
	t.info = info

	for t.offset < info.Size() {
		n, err := t.f.ReadAt(t.buf[:min(int64(len(t.buf)), info.Size()-t.offset)], t.offset)
		t.offset += int64(n)
		t.split.Write(t.buf[:n])
		if err != nil && err != io.EOF || n == 0 {
			break
		}
	}
	t.b.hub.publish(t.ctx, t.b, "file", t.batch...)
	t.batch = t.batch[:0]
}

// backlogOffset returns the offset of the start of the last n lines of f,
// reading backwards in chunks from size.
func backlogOffset(f *os.File, size int64, n int) int64 {
	buf := make([]byte, readChunk)
	lines := 0
	for pos := size; pos > 0; {
		readLen := min(pos, readChunk)
		pos -= readLen
		if _, err := f.ReadAt(buf[:readLen], pos); err != nil && err != io.EOF {
			return 0
		}
		for i := readLen - 1; i >= 0; i-- {
			// The newline ending the file closes the last line; it doesn't start one.
			if buf[i] != '\n' || pos+i == size-1 {
				continue
			}
			if lines++; lines == n {
				return pos + i + 1
			}
		}
	}
	return 0
}
